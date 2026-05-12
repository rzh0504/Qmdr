from __future__ import annotations

import os
from pathlib import Path

APP_NAME = "Qmdr"
CREDENTIAL_FILE_NAME = "qqmusic_cred.pkl"
MIN_FILE_SIZE = 1024
DOWNLOAD_TIMEOUT = 30
SEARCH_RESULTS_COUNT = 5


def app_data_dir() -> Path:
    if os.name == "nt":
        base = Path(os.environ.get("APPDATA") or Path.home() / "AppData" / "Roaming")
        return base / APP_NAME
    if os.name == "posix" and "darwin" in os.sys.platform:
        return Path.home() / "Library" / "Application Support" / APP_NAME
    return Path(os.environ.get("XDG_DATA_HOME", Path.home() / ".local" / "share")) / APP_NAME.lower()


def primary_credential_path() -> Path:
    return app_data_dir() / CREDENTIAL_FILE_NAME


def legacy_credential_path(cwd: Path | None = None) -> Path:
    return (cwd or Path.cwd()) / CREDENTIAL_FILE_NAME


def default_download_dir() -> Path:
    music_dir = Path.home() / "Music" / APP_NAME
    try:
        music_dir.mkdir(parents=True, exist_ok=True)
        return music_dir
    except OSError:
        fallback = Path.cwd() / "music"
        fallback.mkdir(parents=True, exist_ok=True)
        return fallback
