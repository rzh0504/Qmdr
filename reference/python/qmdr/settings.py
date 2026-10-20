from __future__ import annotations

import json
import os
from pathlib import Path

APP_NAME = "Qmdr"
CREDENTIAL_FILE_NAME = "qqmusic_cred.pkl"
SETTINGS_FILE_NAME = "settings.json"
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


def app_settings_path() -> Path:
    return app_data_dir() / SETTINGS_FILE_NAME


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


def load_download_dir(settings_path: Path | None = None, default: Path | None = None) -> Path:
    path = settings_path or app_settings_path()
    fallback = default or default_download_dir()
    try:
        with path.open("r", encoding="utf-8") as fp:
            data = json.load(fp)
    except (OSError, ValueError, TypeError):
        return fallback
    if not isinstance(data, dict):
        return fallback
    download_dir = data.get("download_dir")
    if not isinstance(download_dir, str) or not download_dir.strip():
        return fallback
    return Path(download_dir).expanduser()


def save_download_dir(download_dir: Path, settings_path: Path | None = None) -> Path:
    path = settings_path or app_settings_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    data = {}
    try:
        with path.open("r", encoding="utf-8") as fp:
            existing = json.load(fp)
        if isinstance(existing, dict):
            data.update(existing)
    except (OSError, ValueError, TypeError):
        pass
    data["download_dir"] = str(download_dir.expanduser())
    with path.open("w", encoding="utf-8") as fp:
        json.dump(data, fp, ensure_ascii=False, indent=2)
    return path
