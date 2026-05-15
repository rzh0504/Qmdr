from pathlib import Path
import traceback

import flet as ft

APP_ROOT = Path(__file__).resolve().parent
APP_ICON_ICO = APP_ROOT / "assets" / "qmdr.ico"


def is_mobile_page(page: ft.Page) -> bool:
    platform = str(getattr(page, "platform", "")).lower()
    return "android" in platform or "ios" in platform


async def main(page: ft.Page) -> None:
    page.title = "Qmdr"
    if not is_mobile_page(page) and APP_ICON_ICO.exists():
        page.window.icon = str(APP_ICON_ICO)
    try:
        page.update()

        from qmdr.app import QmdrApp

        await QmdrApp(page).start()
    except Exception:  # noqa: BLE001 - show startup failures on mobile instead of a blank screen.
        details = traceback.format_exc()
        page.controls.clear()
        page.add(
            ft.SafeArea(
                expand=True,
                content=ft.Container(
                    expand=True,
                    padding=16,
                    bgcolor="#fff7ed",
                    content=ft.Column(
                        expand=True,
                        spacing=12,
                        scroll=ft.ScrollMode.AUTO,
                        controls=[
                            ft.Text("Qmdr 启动失败", size=18, weight=ft.FontWeight.BOLD, color="#9a3412"),
                            ft.Text("请截图或复制下面的错误信息反馈。", size=13, color="#7c2d12"),
                            ft.Text(details, size=11, selectable=True, color="#1f2937"),
                        ],
                    ),
                ),
            )
        )
        page.update()


if __name__ == "__main__":
    ft.run(main)
