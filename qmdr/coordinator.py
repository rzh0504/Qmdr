from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable
from pathlib import Path

from qqmusic_api.login import Credential

from .models import DownloadEvent, DownloadOptions, DownloadResult, PlaylistItem, SongItem
from .music import MusicService
from .playlist import PlaylistService
from .utils import emit_event

DownloadCallback = Callable[[DownloadEvent], Awaitable[None] | None]


class DownloadCoordinator:
    def __init__(self, music_service: MusicService, playlist_service: PlaylistService | None = None) -> None:
        self.music_service = music_service
        self.playlist_service = playlist_service
        self.cancel_requested = False

    def cancel(self) -> None:
        self.cancel_requested = True

    def reset(self) -> None:
        self.cancel_requested = False

    async def download_songs(
        self,
        songs: list[SongItem],
        options: DownloadOptions,
        credential: Credential | None,
        on_event: DownloadCallback | None = None,
        folder: Path | None = None,
    ) -> list[DownloadResult]:
        self.reset()
        total = len(songs)
        results: list[DownloadResult] = []
        if total == 0:
            return results

        await emit_event(on_event, DownloadEvent(kind="start", message=f"开始下载 {total} 首歌曲", total=total))

        batch_size = max(1, options.batch_size)
        for start in range(0, total, batch_size):
            if self.cancel_requested:
                await emit_event(on_event, DownloadEvent(kind="cancelled", message="下载已取消", total=total))
                break

            batch = songs[start : start + batch_size]
            tasks = [
                self.music_service.download_song(
                    song=song,
                    options=options,
                    credential=credential,
                    on_event=on_event,
                    folder=folder,
                    current=start + index + 1,
                    total=total,
                )
                for index, song in enumerate(batch)
            ]
            batch_results = await asyncio.gather(*tasks)
            results.extend(batch_results)

            finished = min(start + len(batch), total)
            success = sum(1 for item in results if item.success and not item.skipped)
            skipped = sum(1 for item in results if item.skipped)
            failed = sum(1 for item in results if not item.success)
            await emit_event(
                on_event,
                DownloadEvent(
                    kind="progress",
                    message=f"进度 {finished}/{total}，成功 {success}，跳过 {skipped}，失败 {failed}",
                    current=finished,
                    total=total,
                ),
            )

        success = sum(1 for item in results if item.success and not item.skipped)
        skipped = sum(1 for item in results if item.skipped)
        failed = sum(1 for item in results if not item.success)
        await emit_event(
            on_event,
            DownloadEvent(
                kind="done",
                message=f"下载完成: 成功 {success}，跳过 {skipped}，失败 {failed}",
                current=len(results),
                total=total,
            ),
        )
        return results

    async def download_playlist(
        self,
        playlist: PlaylistItem,
        user_id: str,
        songs: list[SongItem],
        options: DownloadOptions,
        credential: Credential | None,
        on_event: DownloadCallback | None = None,
    ) -> list[DownloadResult]:
        if self.playlist_service is None:
            raise RuntimeError("PlaylistService 未初始化")
        folder = self.playlist_service.playlist_folder(options.download_dir, playlist, user_id)
        await emit_event(
            on_event,
            DownloadEvent(kind="playlist", message=f"开始下载歌单: {playlist.name}", total=len(songs), file_path=folder),
        )
        return await self.download_songs(songs, options, credential, on_event=on_event, folder=folder)
