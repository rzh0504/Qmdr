from __future__ import annotations

import asyncio
import os
from pathlib import Path

import flet as ft

from .coordinator import DownloadCoordinator
from .credential_service import CredentialService
from .models import DownloadEvent, DownloadOptions, PlaylistItem, SongItem
from .music import MusicService
from .playlist import CredentialRequiredError, PlaylistService
from .quality import QUALITY_OPTIONS
from .settings import default_download_dir, load_download_dir, save_download_dir
from .utils import clamp_int


def _border(color: str = "#d8dee9") -> ft.Border:
    side = ft.BorderSide(1, color)
    return ft.Border(side, side, side, side)


def _section(title: str, controls: list[ft.Control], expand: bool | int | None = None) -> ft.Container:
    return ft.Container(
        expand=expand,
        padding=14,
        border=_border(),
        border_radius=6,
        bgcolor="#ffffff",
        content=ft.Column(
            spacing=12,
            controls=[
                ft.Text(title, size=15, weight=ft.FontWeight.BOLD),
                *controls,
            ],
        ),
    )


class QmdrApp:
    def __init__(self, page: ft.Page) -> None:
        self.page = page
        self.credential_service = CredentialService()
        self.music_service = MusicService()
        self.playlist_service = PlaylistService(self.music_service)
        self.coordinator = DownloadCoordinator(self.music_service, self.playlist_service)

        self.credential = None
        self.search_songs: list[SongItem] = []
        self.playlists: list[PlaylistItem] = []
        self.playlist_songs: list[SongItem] = []
        self.selected_playlist: PlaylistItem | None = None
        self.active_download = False
        self.qr_cancelled = False
        self.search_request_id = 0
        self.playlist_request_id = 0
        self.preview_request_id = 0

        self.quality_value = "3"
        self.search_quality_dropdown = self.make_quality_dropdown()
        self.playlist_quality_dropdown = self.make_quality_dropdown()
        self.settings_quality_dropdown = self.make_quality_dropdown()
        self.download_dir_input = ft.TextField(
            label="下载目录",
            value=str(load_download_dir()),
            expand=True,
            on_blur=self.on_download_dir_commit,
            on_submit=self.on_download_dir_commit,
        )
        self.batch_size_input = ft.TextField(label="并发", value="5", width=92)
        self.cover_size_dropdown = ft.Dropdown(
            label="封面",
            value="800",
            width=120,
            options=[ft.DropdownOption(key=str(size), text=f"{size}px") for size in (150, 300, 500, 800)],
        )
        self.overwrite_checkbox = ft.Checkbox(label="覆盖已存在文件", value=False)
        self.external_api_input = ft.TextField(label="外部凭证 API（可选）", value="", expand=True)

        self.credential_text = ft.Text("正在检查凭证...", size=13)
        self.search_input = ft.TextField(label="歌曲关键词", expand=True)
        self.search_status = ft.Text("", size=13, color="#52616b")
        self.search_results_list = ft.ListView(expand=True, spacing=8, padding=0)

        self.musicid_input = ft.TextField(label="musicid(通常为 qq 号或微信 id)", expand=True)
        self.playlist_status = ft.Text("", size=13, color="#52616b")
        self.playlist_list = ft.ListView(expand=True, spacing=8, padding=0)
        self.playlist_preview = ft.ListView(expand=True, spacing=4, padding=0)

        self.current_task_text = ft.Text("暂无任务", size=14, weight=ft.FontWeight.BOLD)
        self.progress_bar = ft.ProgressBar(value=0, height=8)
        self.progress_text = ft.Text("0/0", size=13, color="#52616b")
        self.download_log = ft.ListView(expand=True, spacing=6, padding=0)
        self.file_picker = ft.FilePicker()

        self.views: list[ft.Control] = []
        self.nav = ft.NavigationRail(
            selected_index=0,
            extended=True,
            min_extended_width=172,
            bgcolor="#f7f9fc",
            destinations=[
                ft.NavigationRailDestination(icon=ft.Icons.SEARCH, label="搜索下载"),
                ft.NavigationRailDestination(icon=ft.Icons.LIBRARY_MUSIC, label="歌单下载"),
                ft.NavigationRailDestination(icon=ft.Icons.QUEUE_MUSIC, label="下载队列"),
                ft.NavigationRailDestination(icon=ft.Icons.SETTINGS, label="凭证设置"),
            ],
            on_change=self.on_nav_change,
        )

    async def start(self) -> None:
        self.page.title = "Qmdr"
        self.page.bgcolor = "#f3f6fa"
        self.page.padding = 0
        self.page.theme_mode = ft.ThemeMode.LIGHT

        self.search_input.on_submit = self.on_search
        self.musicid_input.on_submit = self.on_load_playlists
        self.page.services.append(self.file_picker)

        self.views = [
            self.build_search_view(),
            self.build_playlist_view(),
            self.build_queue_view(),
            self.build_settings_view(),
        ]
        for index, view in enumerate(self.views):
            view.visible = index == 0

        self.page.add(
            ft.SafeArea(
                expand=True,
                content=ft.Row(
                    expand=True,
                    spacing=0,
                    controls=[
                        self.nav,
                        ft.VerticalDivider(width=1),
                        ft.Container(expand=True, padding=18, content=ft.Column(expand=True, controls=self.views)),
                    ],
                ),
            )
        )
        await self.reload_credential(show_message=False)

    def build_search_view(self) -> ft.Container:
        return ft.Container(
            expand=True,
            content=ft.Column(
                expand=True,
                spacing=14,
                controls=[
                    _section(
                        "搜索单曲",
                        [
                            ft.Row(
                                controls=[
                                    self.search_input,
                                    ft.Button("搜索", icon=ft.Icons.SEARCH, on_click=self.on_search),
                                ]
                            ),
                            ft.Row(
                                wrap=True,
                                spacing=10,
                                controls=[
                                    self.search_quality_dropdown,
                                ],
                            ),
                            self.search_status,
                        ],
                    ),
                    _section("搜索结果", [self.search_results_list], expand=True),
                ],
            ),
        )

    def build_playlist_view(self) -> ft.Container:
        return ft.Container(
            expand=True,
            content=ft.Column(
                expand=True,
                spacing=14,
                controls=[
                    _section(
                        "歌单",
                        [
                            ft.Row(
                                controls=[
                                    self.musicid_input,
                                    ft.Button("获取歌单", icon=ft.Icons.PLAYLIST_PLAY, on_click=self.on_load_playlists),
                                    ft.Button("下载全部", icon=ft.Icons.DOWNLOAD, on_click=self.on_download_all_playlists),
                                ]
                            ),
                            self.playlist_quality_dropdown,
                            self.playlist_status,
                        ],
                    ),
                    ft.Row(
                        expand=True,
                        spacing=14,
                        controls=[
                            _section("歌单列表", [self.playlist_list], expand=1),
                            _section("歌曲预览", [self.playlist_preview], expand=2),
                        ],
                    ),
                ],
            ),
        )

    def build_queue_view(self) -> ft.Container:
        return ft.Container(
            expand=True,
            content=ft.Column(
                expand=True,
                spacing=14,
                controls=[
                    _section(
                        "当前任务",
                        [
                            self.current_task_text,
                            self.progress_bar,
                            ft.Row(
                                controls=[
                                    self.progress_text,
                                    ft.Button("取消", icon=ft.Icons.CANCEL, on_click=self.on_cancel_download),
                                    ft.Button("打开目录", icon=ft.Icons.FOLDER_OPEN, on_click=self.on_open_download_dir),
                                ]
                            ),
                        ],
                    ),
                    _section("日志", [self.download_log], expand=True),
                ],
            ),
        )

    def build_settings_view(self) -> ft.Container:
        return ft.Container(
            expand=True,
            content=ft.Column(
                expand=True,
                spacing=14,
                controls=[
                    _section(
                        "凭证",
                        [
                            self.credential_text,
                            ft.Row(
                                wrap=True,
                                spacing=10,
                                controls=[
                                    ft.Button("QQ 登录", icon=ft.Icons.LOGIN, on_click=lambda e: self.page.run_task(self.on_qr_login, "qq")),
                                    ft.Button("微信登录", icon=ft.Icons.LOGIN, on_click=lambda e: self.page.run_task(self.on_qr_login, "wx")),
                                    ft.Button("刷新凭证", icon=ft.Icons.REFRESH, on_click=self.on_refresh_credential),
                                    ft.Button("导出脱敏 JSON", icon=ft.Icons.SAVE, on_click=self.on_export_credential),
                                ],
                            ),
                        ],
                    ),
                    _section(
                        "下载设置",
                        [
                            ft.Row(
                                controls=[
                                    self.download_dir_input,
                                    ft.Button("选择", icon=ft.Icons.FOLDER_OPEN, on_click=self.on_pick_download_dir),
                                ]
                            ),
                            ft.Row(
                                wrap=True,
                                spacing=10,
                                controls=[
                                    self.settings_quality_dropdown,
                                    self.cover_size_dropdown,
                                    self.batch_size_input,
                                    self.overwrite_checkbox,
                                ],
                            ),
                        ],
                    ),
                    _section(
                        "高级",
                        [
                            ft.Row(
                                controls=[
                                    self.external_api_input,
                                    ft.Button("重新加载凭证", icon=ft.Icons.KEY, on_click=self.on_reload_with_external_api),
                                ]
                            ),
                        ],
                    ),
                ],
            ),
        )

    def make_quality_dropdown(self) -> ft.Dropdown:
        return ft.Dropdown(
            label="音质",
            value=self.quality_value,
            width=280,
            options=[ft.DropdownOption(key=str(key), text=name) for key, (name, _) in QUALITY_OPTIONS.items()],
            on_select=self.on_quality_select,
        )

    def on_quality_select(self, event: ft.Event[ft.Dropdown]) -> None:
        self.quality_value = event.control.value or "3"
        for control in (
            self.search_quality_dropdown,
            self.playlist_quality_dropdown,
            self.settings_quality_dropdown,
        ):
            if control is not event.control:
                control.value = self.quality_value
        self.page.update()

    def on_nav_change(self, event: ft.Event[ft.NavigationRail]) -> None:
        selected = event.control.selected_index or 0
        for index, view in enumerate(self.views):
            view.visible = index == selected
        self.page.update()

    def options(self) -> DownloadOptions:
        self.save_download_dir_setting()
        return DownloadOptions(
            download_dir=Path(self.download_dir_input.value or default_download_dir()),
            quality_level=clamp_int(self.quality_value, 3, 1, 4),
            cover_size=clamp_int(self.cover_size_dropdown.value, 800, 150, 800),
            batch_size=clamp_int(self.batch_size_input.value, 5, 1, 12),
            overwrite=bool(self.overwrite_checkbox.value),
        )

    def save_download_dir_setting(self, show_message: bool = False) -> None:
        value = (self.download_dir_input.value or "").strip()
        if not value:
            return
        try:
            save_download_dir(Path(value))
        except OSError as exc:
            self.toast(f"保存下载目录失败: {exc}")
            return
        if show_message:
            self.toast("下载目录已保存")

    def on_download_dir_commit(self, event: ft.Event[ft.TextField] | None = None) -> None:
        self.save_download_dir_setting()

    async def reload_credential(self, show_message: bool = True) -> None:
        self.credential_service.set_external_api_url(self.external_api_input.value or "")
        state = await self.credential_service.load_and_refresh_credential()
        self.credential = state.credential if state.loaded else None
        if state.loaded:
            source = "外部 API" if state.loaded_from_api else "本地"
            refreshed = "，已刷新" if state.refreshed else ""
            self.credential_text.value = f"已登录: {state.user_id or '未知用户'}（{source}{refreshed}）"
        else:
            self.credential_text.value = f"未登录: {state.message}"
        if show_message:
            self.toast(self.credential_text.value)
        self.page.update()

    async def on_search(self, event: ft.Event | None = None) -> None:
        keyword = (self.search_input.value or "").strip()
        if not keyword:
            self.toast("请输入歌曲关键词")
            return
        self.search_request_id += 1
        request_id = self.search_request_id
        self.search_status.value = "搜索中..."
        self.search_results_list.controls.clear()
        self.page.update()
        try:
            songs = await self.music_service.search_songs(keyword)
        except Exception as exc:  # noqa: BLE001
            if request_id != self.search_request_id:
                return
            self.search_status.value = f"搜索失败: {exc}"
            self.page.update()
            return
        if request_id != self.search_request_id:
            return
        self.search_songs = songs
        self.search_status.value = f"找到 {len(self.search_songs)} 个结果"
        self.render_search_results()

    def render_search_results(self) -> None:
        self.search_results_list.controls.clear()
        for index, song in enumerate(self.search_songs, 1):
            vip = "  VIP" if song.is_vip else ""
            self.search_results_list.controls.append(
                ft.Container(
                    padding=10,
                    border=_border("#e1e7ef"),
                    border_radius=6,
                    bgcolor="#fbfcfe",
                    content=ft.Row(
                        alignment=ft.MainAxisAlignment.SPACE_BETWEEN,
                        controls=[
                            ft.Column(
                                expand=True,
                                spacing=3,
                                controls=[
                                    ft.Text(f"{index}. {song.title}{vip}", weight=ft.FontWeight.BOLD),
                                    ft.Text(song.singer, size=12, color="#52616b"),
                                    ft.Text(song.album_name or "未知专辑", size=12, color="#8792a2"),
                                ],
                            ),
                            ft.Button("下载", icon=ft.Icons.DOWNLOAD, on_click=lambda e, item=song: self.start_song_download(item)),
                        ],
                    ),
                )
            )
        self.page.update()

    def start_song_download(self, song: SongItem) -> None:
        if self.active_download:
            self.toast("已有下载任务在运行")
            return
        self.page.run_task(self.run_single_download, song)

    async def run_single_download(self, song: SongItem) -> None:
        self.set_download_running(True)
        self.switch_to_queue()
        try:
            await self.coordinator.download_songs([song], self.options(), self.credential, self.on_download_event)
        finally:
            self.set_download_running(False)

    async def on_load_playlists(self, event: ft.Event | None = None) -> None:
        user_id = (self.musicid_input.value or "").strip()
        if not user_id:
            self.toast("请输入 musicid")
            return
        self.playlist_request_id += 1
        request_id = self.playlist_request_id
        self.playlist_status.value = "正在获取歌单..."
        self.playlist_list.controls.clear()
        self.playlist_preview.controls.clear()
        self.page.update()
        try:
            playlists = await self.playlist_service.get_user_playlists(user_id, self.credential)
        except CredentialRequiredError as exc:
            if request_id != self.playlist_request_id:
                return
            self.playlist_status.value = str(exc)
            self.page.update()
            return
        except Exception as exc:  # noqa: BLE001
            if request_id != self.playlist_request_id:
                return
            self.playlist_status.value = f"获取歌单失败: {exc}"
            self.page.update()
            return
        if request_id != self.playlist_request_id:
            return
        self.playlists = playlists
        self.playlist_status.value = f"找到 {len(self.playlists)} 个歌单"
        self.render_playlists()

    def render_playlists(self) -> None:
        self.playlist_list.controls.clear()
        for index, playlist in enumerate(self.playlists, 1):
            self.playlist_list.controls.append(
                ft.Container(
                    padding=10,
                    border=_border("#e1e7ef"),
                    border_radius=6,
                    bgcolor="#fbfcfe",
                    content=ft.Row(
                        controls=[
                            ft.Column(
                                expand=True,
                                spacing=3,
                                controls=[
                                    ft.Text(f"{index}. {playlist.name}", weight=ft.FontWeight.BOLD),
                                    ft.Text(f"{playlist.song_count} 首", size=12, color="#52616b"),
                                ],
                            ),
                            ft.IconButton(ft.Icons.OPEN_IN_NEW, tooltip="预览", on_click=lambda e, item=playlist: self.page.run_task(self.preview_playlist, item)),
                            ft.IconButton(ft.Icons.DOWNLOAD, tooltip="下载", on_click=lambda e, item=playlist: self.page.run_task(self.download_playlist, item)),
                        ],
                    ),
                )
            )
        self.page.update()

    async def preview_playlist(self, playlist: PlaylistItem) -> None:
        user_id = (self.musicid_input.value or "").strip()
        self.selected_playlist = playlist
        self.preview_request_id += 1
        request_id = self.preview_request_id
        self.playlist_preview.controls = [ft.Text("正在加载歌曲...")]
        self.page.update()
        try:
            songs = await self.playlist_service.get_playlist_songs(playlist, user_id, self.credential)
        except Exception as exc:  # noqa: BLE001
            if request_id != self.preview_request_id:
                return
            self.playlist_preview.controls = [ft.Text(f"预览失败: {exc}", color="#b42318")]
            self.page.update()
            return
        if request_id != self.preview_request_id:
            return
        self.playlist_songs = songs
        self.render_playlist_preview(playlist)

    def render_playlist_preview(self, playlist: PlaylistItem) -> None:
        self.playlist_preview.controls.clear()
        self.playlist_preview.controls.append(
            ft.Row(
                controls=[
                    ft.Text(f"{playlist.name}：{len(self.playlist_songs)} 首", weight=ft.FontWeight.BOLD, expand=True),
                    ft.Button("下载此歌单", icon=ft.Icons.DOWNLOAD, on_click=lambda e: self.page.run_task(self.download_playlist, playlist)),
                ]
            )
        )
        for index, song in enumerate(self.playlist_songs, 1):
            vip = "  VIP" if song.is_vip else ""
            self.playlist_preview.controls.append(ft.Text(f"{index}. {song.singer} - {song.title}{vip}", size=13))
        self.page.update()

    async def download_playlist(self, playlist: PlaylistItem) -> None:
        if self.active_download:
            self.toast("已有下载任务在运行")
            return
        user_id = (self.musicid_input.value or "").strip()
        if not user_id:
            self.toast("请输入 musicid")
            return
        self.set_download_running(True)
        self.switch_to_queue()
        try:
            songs = self.playlist_songs if self.selected_playlist == playlist else []
            if not songs:
                songs = await self.playlist_service.get_playlist_songs(playlist, user_id, self.credential)
            await self.coordinator.download_playlist(playlist, user_id, songs, self.options(), self.credential, self.on_download_event)
        except Exception as exc:  # noqa: BLE001
            await self.on_download_event(DownloadEvent(kind="failed", message=f"歌单下载失败: {exc}", error=str(exc)))
        finally:
            self.set_download_running(False)

    async def on_download_all_playlists(self, event: ft.Event | None = None) -> None:
        if self.active_download:
            self.toast("已有下载任务在运行")
            return
        user_id = (self.musicid_input.value or "").strip()
        if not user_id:
            self.toast("请输入 musicid")
            return
        if not self.playlists:
            await self.on_load_playlists()
        if not self.playlists:
            return
        self.set_download_running(True)
        self.switch_to_queue()
        self.coordinator.reset()
        cancelled_before_playlist = False
        try:
            for playlist in self.playlists:
                if self.coordinator.cancel_requested:
                    cancelled_before_playlist = True
                    break
                try:
                    songs = await self.playlist_service.get_playlist_songs(playlist, user_id, self.credential)
                    if self.coordinator.cancel_requested:
                        cancelled_before_playlist = True
                        break
                    await self.coordinator.download_playlist(
                        playlist,
                        user_id,
                        songs,
                        self.options(),
                        self.credential,
                        self.on_download_event,
                        reset_cancel=False,
                    )
                except Exception as exc:  # noqa: BLE001
                    await self.on_download_event(DownloadEvent(kind="failed", message=f"{playlist.name}: {exc}", error=str(exc)))
            if cancelled_before_playlist:
                await self.on_download_event(DownloadEvent(kind="cancelled", message="下载已取消"))
        finally:
            self.set_download_running(False)

    async def on_download_event(self, event: DownloadEvent) -> None:
        if event.kind in {"start", "playlist", "progress", "done", "cancelled"} and event.total:
            self.progress_bar.value = min(1, max(0, event.current / event.total))
            self.progress_text.value = f"{event.current}/{event.total}"
        if event.kind in {"start", "playlist", "downloading", "file_progress", "progress", "done", "cancelled"}:
            self.current_task_text.value = event.message
        if event.kind == "file_progress":
            self.page.update()
            return
        color = {
            "success": "#087443",
            "failed": "#b42318",
            "warning": "#b54708",
            "skipped": "#52616b",
            "fallback": "#52616b",
        }.get(event.kind, "#1f2937")
        self.download_log.controls.append(ft.Text(event.message, size=12, color=color))
        if len(self.download_log.controls) > 220:
            self.download_log.controls = self.download_log.controls[-220:]
        self.page.update()

    def set_download_running(self, running: bool) -> None:
        self.active_download = running
        if running:
            self.progress_bar.value = 0
            self.progress_text.value = "0/0"
        self.page.update()

    def switch_to_queue(self) -> None:
        self.nav.selected_index = 2
        for index, view in enumerate(self.views):
            view.visible = index == 2
        self.page.update()

    def on_cancel_download(self, event: ft.Event | None = None) -> None:
        self.coordinator.cancel()
        self.toast("已请求取消，当前批次结束后停止")

    async def on_pick_download_dir(self, event: ft.Event | None = None) -> None:
        try:
            selected = await self.file_picker.get_directory_path(dialog_title="选择下载目录")
        except Exception as exc:  # noqa: BLE001
            self.toast(f"无法打开目录选择器: {exc}")
            return
        if selected:
            self.download_dir_input.value = selected
            self.save_download_dir_setting(show_message=True)
            self.page.update()

    def on_open_download_dir(self, event: ft.Event | None = None) -> None:
        path = Path(self.download_dir_input.value or default_download_dir()).resolve()
        path.mkdir(parents=True, exist_ok=True)
        try:
            if os.name == "nt":
                os.startfile(path)  # type: ignore[attr-defined]
            else:
                self.page.launch_url(path.as_uri())
        except Exception as exc:  # noqa: BLE001
            self.toast(f"打开目录失败: {exc}")

    async def on_refresh_credential(self, event: ft.Event | None = None) -> None:
        refreshed = await self.credential_service.refresh_credential()
        if refreshed is None:
            self.toast("凭证刷新失败或不可刷新")
        await self.reload_credential(show_message=True)

    async def on_reload_with_external_api(self, event: ft.Event | None = None) -> None:
        await self.reload_credential(show_message=True)

    def on_export_credential(self, event: ft.Event | None = None) -> None:
        try:
            path = self.credential_service.export_credential_to_json(Path.cwd())
        except Exception as exc:  # noqa: BLE001
            self.toast(str(exc))
            return
        self.toast(f"已导出脱敏 JSON: {path.name}")

    async def on_qr_login(self, login_type: str) -> None:
        self.qr_cancelled = False
        status = ft.Text("正在获取二维码...", size=13)
        dialog = ft.AlertDialog(
            modal=True,
            title=f"{'QQ' if login_type == 'qq' else '微信'}扫码登录",
            content=ft.Column(
                tight=True,
                horizontal_alignment=ft.CrossAxisAlignment.CENTER,
            controls=[
                ft.Container(width=240, height=240, content=ft.ProgressBar()),
                status,
            ],
            ),
            actions=[ft.Button("取消", icon=ft.Icons.CANCEL, on_click=lambda e: self.cancel_qr_dialog())],
        )
        self.page.show_dialog(dialog)
        self.page.update()

        try:
            session = await self.credential_service.start_qr_login(login_type)
        except Exception as exc:  # noqa: BLE001
            status.value = f"获取二维码失败: {exc}"
            self.page.update()
            return

        dialog.content = ft.Column(
            tight=True,
            horizontal_alignment=ft.CrossAxisAlignment.CENTER,
            controls=[
                ft.Image(src=session.image_bytes, width=240, height=240, fit=ft.BoxFit.CONTAIN),
                status,
            ],
        )
        status.value = "请使用手机扫码"
        self.page.update()

        while not self.qr_cancelled:
            try:
                result = await self.credential_service.poll_qr_login(session)
            except Exception as exc:  # noqa: BLE001
                status.value = f"登录检查失败: {exc}"
                self.page.update()
                return
            status.value = result.message or f"状态: {result.event_name}"
            self.page.update()
            if result.done:
                self.page.pop_dialog()
                await self.reload_credential(show_message=True)
                return
            if result.failed:
                return
            await asyncio.sleep(2)

    def cancel_qr_dialog(self) -> None:
        self.qr_cancelled = True
        self.page.pop_dialog()
        self.page.update()

    def toast(self, message: str) -> None:
        self.page.show_dialog(ft.SnackBar(ft.Text(message), show_close_icon=True))
        self.page.update()


async def main(page: ft.Page) -> None:
    app = QmdrApp(page)
    await app.start()
