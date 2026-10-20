from __future__ import annotations

import inspect
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import TypeVar

from .models import DownloadEvent

T = TypeVar("T")

WINDOWS_RESERVED_NAMES = {
    "CON",
    "PRN",
    "AUX",
    "NUL",
    *(f"COM{index}" for index in range(1, 10)),
    *(f"LPT{index}" for index in range(1, 10)),
}


def sanitize_filename(filename: str, max_length: int = 180) -> str:
    illegal_chars = ['<', '>', ':', '"', '/', '\\', '|', '?', '*']
    for char in illegal_chars:
        filename = filename.replace(char, "_")
    filename = filename.strip().rstrip(". ") or "untitled"
    stem = filename.split(".", 1)[0].upper()
    if stem in WINDOWS_RESERVED_NAMES:
        filename = f"_{filename}"
    if len(filename) > max_length:
        filename = filename[:max_length].rstrip(". ") or "untitled"
    return filename


def ensure_directory(path: Path) -> Path:
    path.mkdir(parents=True, exist_ok=True)
    return path


def clamp_int(value: str | int | None, default: int, minimum: int, maximum: int) -> int:
    try:
        number = int(value)  # type: ignore[arg-type]
    except (TypeError, ValueError):
        return default
    return max(minimum, min(maximum, number))


def mask_secret(value: object, keep: int = 6) -> str:
    text = "" if value is None else str(value)
    if len(text) <= keep * 2:
        return text
    return f"{text[:keep]}...{text[-keep:]}"


async def emit_event(
    callback: Callable[[DownloadEvent], Awaitable[None] | None] | None,
    event: DownloadEvent,
) -> None:
    if callback is None:
        return
    result = callback(event)
    if inspect.isawaitable(result):
        await result
