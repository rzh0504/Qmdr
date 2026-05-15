from __future__ import annotations

import json
from typing import Any


JSONDecodeError = json.JSONDecodeError
OPT_NON_STR_KEYS = 0
OPT_SERIALIZE_NUMPY = 0
OPT_SORT_KEYS = 0


def dumps(value: Any, *args: Any, **kwargs: Any) -> bytes:
    default = kwargs.get("default")
    sort_keys = bool(kwargs.get("option", 0) & OPT_SORT_KEYS)
    return json.dumps(
        value,
        ensure_ascii=False,
        separators=(",", ":"),
        default=default,
        sort_keys=sort_keys,
    ).encode("utf-8")


def loads(value: bytes | bytearray | memoryview | str) -> Any:
    if isinstance(value, memoryview):
        value = value.tobytes()
    if isinstance(value, (bytes, bytearray)):
        value = value.decode("utf-8")
    return json.loads(value)
