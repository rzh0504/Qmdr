from __future__ import annotations

from qqmusic_api.song import SongFileType

QualityStrategy = list[tuple[SongFileType, str]]

QUALITY_OPTIONS: dict[int, tuple[str, QualityStrategy]] = {
    1: (
        "臻品母带 (MASTER, 24Bit 192kHz)",
        [
            (SongFileType.MASTER, "臻品母带"),
            (SongFileType.ATMOS_2, "臻品全景声"),
            (SongFileType.ATMOS_51, "臻品音质"),
            (SongFileType.FLAC, "FLAC"),
            (SongFileType.MP3_320, "MP3 320kbps"),
            (SongFileType.MP3_128, "MP3 128kbps"),
        ],
    ),
    2: (
        "臻品全景声 (ATMOS, 16Bit 44.1kHz)",
        [
            (SongFileType.ATMOS_2, "臻品全景声"),
            (SongFileType.ATMOS_51, "臻品音质"),
            (SongFileType.FLAC, "FLAC"),
            (SongFileType.MP3_320, "MP3 320kbps"),
            (SongFileType.MP3_128, "MP3 128kbps"),
        ],
    ),
    3: (
        "FLAC 无损 (16Bit~24Bit)",
        [
            (SongFileType.FLAC, "FLAC"),
            (SongFileType.MP3_320, "MP3 320kbps"),
            (SongFileType.MP3_128, "MP3 128kbps"),
        ],
    ),
    4: (
        "MP3 320kbps",
        [
            (SongFileType.MP3_320, "MP3 320kbps"),
            (SongFileType.MP3_128, "MP3 128kbps"),
        ],
    ),
}


def get_quality_strategy(quality_level: int) -> QualityStrategy:
    return QUALITY_OPTIONS.get(quality_level, QUALITY_OPTIONS[3])[1]
