from __future__ import annotations

import asyncio
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from qqmusic_api.song import SongFileType

from qmdr.coordinator import DownloadCoordinator
from qmdr.credential_service import CredentialService
from qmdr.models import DownloadEvent, DownloadOptions, DownloadResult, SongItem
from qmdr.music import MusicService
from qmdr.quality import get_quality_strategy
from qmdr.utils import sanitize_filename


class CoreTests(unittest.TestCase):
    def test_sanitize_filename_replaces_invalid_chars(self) -> None:
        self.assertEqual(sanitize_filename('A<B>C:D"E/F\\G|H?I*'), "A_B_C_D_E_F_G_H_I_")

    def test_quality_strategy_falls_back_in_expected_order(self) -> None:
        strategy = get_quality_strategy(1)
        self.assertEqual(strategy[0][0], SongFileType.MASTER)
        self.assertEqual(strategy[-1][0], SongFileType.MP3_128)

    def test_credential_candidates_prefer_app_data_path(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            primary = root / "app" / "qqmusic_cred.pkl"
            legacy = root / "qqmusic_cred.pkl"
            service = CredentialService(credential_path=primary, legacy_path=legacy)
            self.assertEqual(service.candidate_paths(), [primary, legacy])

    def test_existing_file_is_skipped_without_network(self) -> None:
        async def run() -> None:
            with tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                song = SongItem(title="Song", singer="Artist", mid="mid")
                existing = root / "Artist - Song.flac"
                existing.write_bytes(b"already here")
                events: list[DownloadEvent] = []
                service = MusicService()
                try:
                    result = await service.download_song(
                        song,
                        DownloadOptions(download_dir=root, quality_level=3),
                        on_event=events.append,
                    )
                finally:
                    await service.close()
                self.assertTrue(result.success)
                self.assertTrue(result.skipped)
                self.assertEqual(events[-1].kind, "skipped")

        asyncio.run(run())

    def test_download_url_failure_emits_failed_result(self) -> None:
        async def fake_get_song_urls(*args, **kwargs):  # noqa: ANN002, ANN003
            return {}

        async def run() -> None:
            with tempfile.TemporaryDirectory() as temp:
                events: list[DownloadEvent] = []
                service = MusicService()
                try:
                    with patch("qmdr.music.get_song_urls", fake_get_song_urls):
                        result = await service.download_song(
                            SongItem(title="Missing", singer="Artist", mid="missing"),
                            DownloadOptions(download_dir=Path(temp), quality_level=4),
                            on_event=events.append,
                        )
                finally:
                    await service.close()
                self.assertFalse(result.success)
                self.assertEqual(events[-1].kind, "failed")

        asyncio.run(run())

    def test_coordinator_counts_batch_results(self) -> None:
        class FakeMusicService:
            async def download_song(self, song, options, credential=None, on_event=None, folder=None, current=1, total=1):
                if on_event:
                    on_event(DownloadEvent(kind="success", message=song.title, current=current, total=total, song=song))
                return DownloadResult(True, song=song, file_path=Path(f"{song.title}.mp3"))

        async def run() -> None:
            songs = [SongItem(title=f"Song {index}", singer="Artist", mid=str(index)) for index in range(5)]
            events: list[DownloadEvent] = []
            coordinator = DownloadCoordinator(FakeMusicService())  # type: ignore[arg-type]
            results = await coordinator.download_songs(
                songs,
                DownloadOptions(download_dir=Path("."), batch_size=2),
                credential=None,
                on_event=events.append,
            )
            self.assertEqual(len(results), 5)
            self.assertTrue(all(item.success for item in results))
            self.assertEqual(events[-1].kind, "done")

        asyncio.run(run())


if __name__ == "__main__":
    unittest.main()
