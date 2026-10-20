# Qmdr

Qmdr 是一个使用 Go + MyGo 原生 UI 的 QQ 音乐桌面下载工具，目标平台为 Windows、macOS 和 Linux。

## 当前进度

- 原生亮色 UI：搜索下载、歌单预览、下载队列、设置
- 真实 QQ 音乐搜索
- 读取本地凭证，真实获取 FLAC / MP3 下载链接
- 真实文件下载：队列进度、取消、跳过已存在文件、任务日志
- 待接入：扫码登录、歌单 API、封面/歌词元数据、设置持久化、打包与签名

歌单页目前展示演示数据，下载按钮会提示接口待接入；扫码登录界面是占位。

## 运行

需要 Go 1.27.1 或更高版本。

```powershell
go run .
# 或
go build -o build/qmdr.exe .
```

## 凭证

Go 客户端从系统配置目录读取 `qqmusic_cred.json`：

- Windows：`%AppData%\Qmdr\qqmusic_cred.json`
- macOS：`~/Library/Application Support/Qmdr/qqmusic_cred.json`
- Linux：`$XDG_CONFIG_HOME/Qmdr/qqmusic_cred.json`

该文件包含登录令牌，不会提交到仓库。设置页可以重新加载或导出 JSON。旧版 Python 的 pickle 凭证已迁移为上述 JSON，`reference/python` 保留旧后端代码用于行为参考。

## 目录

- `main.go`、`view_*.go`、`components.go`、`theme.go`：MyGo 原生 UI
- `download.go`：下载队列、文件命名与落盘
- `internal/qqmusic`：请求封装、搜索、下载链接、歌词、文件下载
- `reference/python`：旧 Python 后端，仅作参考
- `assets`：应用图标资源

## 免责声明

本项目仅供学习和研究使用。请尊重版权、支持正版音乐，并遵守 QQ 音乐相关服务条款。禁止将本项目用于商业用途或任何侵权行为。
