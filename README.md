# Qmdr

Qmdr 是一个基于 Flet 的 QQ 音乐桌面下载工具，支持本地下载 QQ 音乐单曲和歌单。

本项目基于 GPL-3.0 项目 [qq-music-download](https://github.com/tooplick/qq-music-download) 实现图形界面。

## 运行

```powershell
uv run flet run -a assets main.py
```

## 热重载

```powershell
uv run flet run --recursive --ignore-dirs .venv,.agents,__pycache__ -a assets main.py
```

`--recursive` 会递归监听项目文件变化。建议忽略 `.venv`、`.agents` 和
`__pycache__`，避免依赖或缓存文件变化触发重复重载，也能减少启动和监听开销。

## 功能

- QQ / 微信扫码登录，并将凭证保存到本地。
- 单曲搜索下载，支持音质自动降级。
- 歌单预览和批量下载。
- 为下载文件写入封面、歌词和基础元数据。
- 本地下载队列，显示进度和单曲状态。

## 说明

- 凭证默认保存到应用数据目录，文件名为 `qqmusic_cred.pkl`。
- 应用仍可读取当前工作目录下旧版的 `qqmusic_cred.pkl`。
- 下载目录会保存到应用数据目录下的 `settings.json`。
- 默认下载目录为 `~/Music/Qmdr`；如果无法创建，则回退到 `./music`。

## 免责声明

本项目仅供学习和研究使用。请尊重版权、支持正版音乐，并遵守 QQ 音乐相关服务条款。禁止将本项目用于商业用途或任何侵权行为。
