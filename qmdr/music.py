from __future__ import annotations

from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any

import aiofiles
import aiohttp
from mutagen.flac import FLAC, Picture
from mutagen.id3 import APIC, ID3, TALB, TIT2, TPE1, USLT
from qqmusic_api import search
from qqmusic_api.login import Credential
from qqmusic_api.lyric import get_lyric
from qqmusic_api.song import SongFileType, get_song_urls

from .models import DownloadEvent, DownloadOptions, DownloadResult, SongItem
from .quality import get_quality_strategy
from .settings import DOWNLOAD_TIMEOUT, MIN_FILE_SIZE, SEARCH_RESULTS_COUNT
from .utils import emit_event, ensure_directory, sanitize_filename

DownloadCallback = Callable[[DownloadEvent], Awaitable[None] | None]


class DownloadError(Exception):
    pass


class MetadataError(Exception):
    pass


class NetworkManager:
    def __init__(self, timeout: int = DOWNLOAD_TIMEOUT) -> None:
        self.timeout = timeout
        self.session: aiohttp.ClientSession | None = None

    async def get_session(self) -> aiohttp.ClientSession:
        if self.session is None or self.session.closed:
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            self.session = aiohttp.ClientSession(timeout=timeout)
        return self.session

    async def close(self) -> None:
        if self.session is not None and not self.session.closed:
            await self.session.close()
        self.session = None


class CoverManager:
    @staticmethod
    def get_cover_url_by_album_mid(mid: str, size: int = 800) -> str | None:
        if not mid:
            return None
        if size not in {150, 300, 500, 800}:
            raise ValueError("不支持的封面尺寸")
        return f"https://y.gtimg.cn/music/photo_new/T002R{size}x{size}M000{mid}.jpg"

    @staticmethod
    def get_cover_url_by_vs(vs: str, size: int = 800) -> str | None:
        if not vs:
            return None
        if size not in {150, 300, 500, 800}:
            raise ValueError("不支持的封面尺寸")
        return f"https://y.qq.com/music/photo_new/T062R{size}x{size}M000{vs}.jpg"

    @staticmethod
    async def get_valid_cover_url(
        song_data: dict[str, Any],
        network: NetworkManager,
        size: int = 800,
    ) -> str | None:
        album_mid = song_data.get("album", {}).get("mid", "")
        if album_mid:
            url = CoverManager.get_cover_url_by_album_mid(album_mid, size)
            if await CoverManager.download_cover(url, network):
                return url

        candidates: list[tuple[int, str]] = []
        for i, vs in enumerate(song_data.get("vs", [])):
            if vs and isinstance(vs, str) and len(vs) >= 3 and "," not in vs:
                candidates.append((i, vs))
        for i, vs in enumerate(song_data.get("vs", [])):
            if vs and isinstance(vs, str) and "," in vs:
                for part in (item.strip() for item in vs.split(",")):
                    if len(part) >= 3:
                        candidates.append((100 + i, part))

        for _, value in sorted(candidates, key=lambda item: item[0]):
            url = CoverManager.get_cover_url_by_vs(value, size)
            if await CoverManager.download_cover(url, network):
                return url
        return None

    @staticmethod
    async def download_cover(url: str | None, network: NetworkManager) -> bytes | None:
        if not url:
            return None
        try:
            session = await network.get_session()
            async with session.get(url) as response:
                if response.status != 200:
                    return None
                content = await response.read()
        except (aiohttp.ClientError, TimeoutError):
            return None
        if len(content) <= MIN_FILE_SIZE:
            return None
        if content.startswith(b"\xff\xd8") or content.startswith(b"\x89PNG"):
            return content
        return None


class MetadataManager:
    def __init__(self, network: NetworkManager) -> None:
        self.network = network

    async def add_metadata(
        self,
        file_path: Path,
        song: SongItem,
        song_data: dict[str, Any],
        cover_size: int,
    ) -> None:
        lyrics_data = await self._get_lyrics(song.mid)
        suffix = file_path.suffix.lower()
        if suffix == ".flac":
            await self._add_metadata_to_flac(file_path, song, lyrics_data, song_data, cover_size)
        elif suffix in {".mp3", ".m4a"}:
            await self._add_metadata_to_mp3(file_path, song, lyrics_data, song_data, cover_size)

    async def _add_metadata_to_flac(
        self,
        file_path: Path,
        song: SongItem,
        lyrics_data: dict[str, Any] | None,
        song_data: dict[str, Any],
        cover_size: int,
    ) -> None:
        try:
            audio = FLAC(file_path)
            audio["title"] = song.title
            audio["artist"] = song.singer
            audio["album"] = song.album_name
            cover_url = await CoverManager.get_valid_cover_url(song_data, self.network, cover_size)
            cover_data = await CoverManager.download_cover(cover_url, self.network)
            if cover_data:
                image = Picture()
                image.type = 3
                image.mime = "image/png" if cover_data.startswith(b"\x89PNG") else "image/jpeg"
                image.desc = "Cover"
                image.data = cover_data
                audio.clear_pictures()
                audio.add_picture(image)
            if lyrics_data:
                if lyric_text := lyrics_data.get("lyric"):
                    audio["lyrics"] = lyric_text
                if trans_text := lyrics_data.get("trans"):
                    audio["translyrics"] = trans_text
            audio.save()
        except Exception as exc:  # noqa: BLE001
            raise MetadataError(f"FLAC 元数据处理失败: {exc}") from exc

    async def _add_metadata_to_mp3(
        self,
        file_path: Path,
        song: SongItem,
        lyrics_data: dict[str, Any] | None,
        song_data: dict[str, Any],
        cover_size: int,
    ) -> None:
        try:
            try:
                audio = ID3(file_path)
            except Exception:
                audio = ID3()
            for tag in ["APIC:", "USLT:", "TIT2", "TPE1", "TALB"]:
                if tag in audio:
                    del audio[tag]
            audio.add(TIT2(encoding=3, text=song.title))
            audio.add(TPE1(encoding=3, text=song.singer))
            audio.add(TALB(encoding=3, text=song.album_name))
            cover_url = await CoverManager.get_valid_cover_url(song_data, self.network, cover_size)
            cover_data = await CoverManager.download_cover(cover_url, self.network)
            if cover_data:
                audio.add(
                    APIC(
                        encoding=3,
                        mime="image/png" if cover_data.startswith(b"\x89PNG") else "image/jpeg",
                        type=3,
                        desc="Cover",
                        data=cover_data,
                    )
                )
            if lyrics_data and (lyric_text := lyrics_data.get("lyric")):
                audio.add(USLT(encoding=3, lang="eng", desc="Lyrics", text=lyric_text))
            audio.save(file_path, v2_version=3)
        except Exception as exc:  # noqa: BLE001
            raise MetadataError(f"MP3 元数据处理失败: {exc}") from exc

    async def _get_lyrics(self, song_mid: str) -> dict[str, Any] | None:
        try:
            return await get_lyric(song_mid)
        except Exception:  # noqa: BLE001
            return None


class MusicService:
    def __init__(self, network: NetworkManager | None = None) -> None:
        self.network = network or NetworkManager()
        self.metadata = MetadataManager(self.network)

    async def close(self) -> None:
        await self.network.close()

    async def search_songs(self, keyword: str, count: int = SEARCH_RESULTS_COUNT) -> list[SongItem]:
        keyword = keyword.strip()
        if not keyword:
            raise ValueError("搜索关键词不能为空")
        results = await search.search_by_type(keyword, num=count)
        return [self.song_from_raw(item) for item in results or []]

    def song_from_raw(self, data: dict[str, Any]) -> SongItem:
        singer_info = data.get("singer", [])
        singer = "未知歌手"
        if singer_info and isinstance(singer_info, list):
            singer = singer_info[0].get("name", singer)
        return SongItem(
            title=data.get("title", "未知歌曲"),
            singer=singer,
            mid=data.get("mid", ""),
            album_name=data.get("album", {}).get("name", ""),
            album_mid=data.get("album", {}).get("mid", ""),
            is_vip=data.get("pay", {}).get("pay_play", 0) != 0,
            raw=data,
        )

    async def download_song(
        self,
        song: SongItem,
        options: DownloadOptions,
        credential: Credential | None = None,
        on_event: DownloadCallback | None = None,
        folder: Path | None = None,
        current: int = 1,
        total: int = 1,
    ) -> DownloadResult:
        target_dir = ensure_directory(folder or options.download_dir)
        safe_name = sanitize_filename(song.display_name)

        if song.is_vip and credential is None:
            await emit_event(
                on_event,
                DownloadEvent(
                    kind="warning",
                    message=f"{song.display_name} 是 VIP 歌曲，未登录时可能只能下载低音质或失败",
                    current=current,
                    total=total,
                    song=song,
                ),
            )

        last_error = "所有音质下载失败"
        for file_type, quality_name in get_quality_strategy(options.quality_level):
            file_path = target_dir / f"{safe_name}{file_type.e}"
            if file_path.exists() and not options.overwrite:
                await emit_event(
                    on_event,
                    DownloadEvent(
                        kind="skipped",
                        message=f"文件已存在，跳过: {file_path.name}",
                        current=current,
                        total=total,
                        song=song,
                        file_path=file_path,
                    ),
                )
                return DownloadResult(True, song=song, skipped=True, quality=quality_name, file_path=file_path)

            await emit_event(
                on_event,
                DownloadEvent(
                    kind="downloading",
                    message=f"尝试 {quality_name}: {song.display_name}",
                    current=current,
                    total=total,
                    song=song,
                    file_path=file_path,
                ),
            )
            result = await self._download_with_quality(
                song=song,
                file_type=file_type,
                quality_name=quality_name,
                file_path=file_path,
                credential=credential,
                options=options,
                on_event=on_event,
                current=current,
                total=total,
            )
            if result.success:
                return result
            last_error = result.error or last_error

        await emit_event(
            on_event,
            DownloadEvent(
                kind="failed",
                message=f"下载失败: {song.display_name} - {last_error}",
                current=current,
                total=total,
                song=song,
                error=last_error,
            ),
        )
        return DownloadResult(False, song=song, error=last_error)

    async def _download_with_quality(
        self,
        song: SongItem,
        file_type: SongFileType,
        quality_name: str,
        file_path: Path,
        credential: Credential | None,
        options: DownloadOptions,
        on_event: DownloadCallback | None,
        current: int,
        total: int,
    ) -> DownloadResult:
        try:
            urls = await get_song_urls([song.mid], file_type=file_type, credential=credential)
        except Exception as exc:  # noqa: BLE001
            return DownloadResult(False, song=song, quality=quality_name, error=f"获取 URL 失败: {exc}")

        url = urls.get(song.mid)
        if isinstance(url, tuple):
            url = url[0]
        if not url:
            await emit_event(
                on_event,
                DownloadEvent(
                    kind="fallback",
                    message=f"{quality_name} 不可用，尝试降级",
                    current=current,
                    total=total,
                    song=song,
                ),
            )
            return DownloadResult(False, song=song, quality=quality_name, error=f"无法获取 {quality_name} URL")

        try:
            session = await self.network.get_session()
            async with session.get(url) as response:
                if response.status != 200:
                    return DownloadResult(False, song=song, quality=quality_name, error=f"HTTP {response.status}")
                content = await response.read()
        except (aiohttp.ClientError, TimeoutError) as exc:
            return DownloadResult(False, song=song, quality=quality_name, error=f"网络错误: {exc}")

        if len(content) <= MIN_FILE_SIZE:
            return DownloadResult(False, song=song, quality=quality_name, error="文件过小，可能下载失败")

        ensure_directory(file_path.parent)
        async with aiofiles.open(file_path, "wb") as fp:
            await fp.write(content)

        try:
            await self.metadata.add_metadata(file_path, song, song.raw, options.cover_size)
        except MetadataError as exc:
            await emit_event(
                on_event,
                DownloadEvent(
                    kind="warning",
                    message=f"元数据写入失败，但音频已保存: {exc}",
                    current=current,
                    total=total,
                    song=song,
                    file_path=file_path,
                    error=str(exc),
                ),
            )

        await emit_event(
            on_event,
            DownloadEvent(
                kind="success",
                message=f"下载成功: {file_path.name}",
                current=current,
                total=total,
                song=song,
                file_path=file_path,
            ),
        )
        return DownloadResult(True, song=song, quality=quality_name, file_path=file_path)
