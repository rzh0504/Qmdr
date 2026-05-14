from pathlib import Path

import flet as ft

APP_ROOT = Path(__file__).resolve().parent
APP_ICON_ICO = APP_ROOT / "assets" / "qmdr.ico"


async def main(page: ft.Page) -> None:
    page.title = "Qmdr"
    if APP_ICON_ICO.exists():
        page.window.icon = str(APP_ICON_ICO)
    page.update()

    from qmdr.app import QmdrApp

    await QmdrApp(page).start()


if __name__ == "__main__":
    ft.run(main)
