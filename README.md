# Qmdr

Qmdr is a Flet desktop GUI for downloading QQ Music songs and playlists locally.
It is based on the GPL-3.0 project `tooplick/qq-music-download`.

## Run

```powershell
uv run flet run
```

## Features

- QQ / WeChat QR login and local credential storage.
- Song search with quality fallback.
- Playlist preview and batch download.
- Cover, lyric, and basic metadata writing for downloaded files.
- Local download queue with progress and per-song status.

## Notes

- Credentials are saved under the app data directory as `qqmusic_cred.pkl`.
- The app can still read an old `qqmusic_cred.pkl` from the current working directory.
- Default downloads go to `~/Music/Qmdr`, or `./music` if that directory cannot be created.

## Disclaimer

This project is for learning and research only. Respect copyright, support licensed music, and comply with QQ Music's terms. Do not use this project for commercial or infringing activity.
