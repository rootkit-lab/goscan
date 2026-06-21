#!/usr/bin/env python3
"""Gera assets/icon/goscan.png (256x256) a partir do SVG ou desenho inline."""

from __future__ import annotations

import struct
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "assets" / "icon" / "goscan.png"
SIZE = 256


def _png_chunk(tag: bytes, data: bytes) -> bytes:
    crc = zlib.crc32(tag + data) & 0xFFFFFFFF
    return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", crc)


def write_png(path: Path, rgba: bytes, w: int, h: int) -> None:
    raw = b"".join(b"\x00" + rgba[y * w * 4 : (y + 1) * w * 4] for y in range(h))
    ihdr = struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0)
    png = b"\x89PNG\r\n\x1a\n"
    png += _png_chunk(b"IHDR", ihdr)
    png += _png_chunk(b"IDAT", zlib.compress(raw, 9))
    png += _png_chunk(b"IEND", b"")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(png)


def _blend(fg: tuple[int, int, int, int], bg: tuple[int, int, int]) -> tuple[int, int, int]:
    fa = fg[3] / 255
    return (
        int(fg[0] * fa + bg[0] * (1 - fa)),
        int(fg[1] * fa + bg[1] * (1 - fa)),
        int(fg[2] * fa + bg[2] * (1 - fa)),
    )


def _set_px(buf: bytearray, x: int, y: int, color: tuple[int, int, int]) -> None:
    if 0 <= x < SIZE and 0 <= y < SIZE:
        i = (y * SIZE + x) * 4
        buf[i : i + 3] = bytes(color)
        buf[i + 3] = 255


def _stroke_circle(buf: bytearray, cx: float, cy: float, r: float, rgb: tuple[int, int, int], w: int = 3) -> None:
    for y in range(SIZE):
        for x in range(SIZE):
            d = abs(((x - cx) ** 2 + (y - cy) ** 2) ** 0.5 - r)
            if d <= w:
                _set_px(buf, x, y, rgb)


def _fill_round_rect(buf: bytearray, rgb: tuple[int, int, int]) -> None:
    bg = (30, 30, 30)
    radius = 48
    for y in range(SIZE):
        for x in range(SIZE):
            inside = True
            for cx, cy in (
                (radius, radius),
                (SIZE - radius - 1, radius),
                (radius, SIZE - radius - 1),
                (SIZE - radius - 1, SIZE - radius - 1),
            ):
                if (x < radius or x >= SIZE - radius) and (y < radius or y >= SIZE - radius):
                    if (x - cx) ** 2 + (y - cy) ** 2 > radius**2:
                        inside = False
                        break
            if inside:
                t = (x + y) / (2 * SIZE)
                c = tuple(int(bg[i] * (1 - t) + rgb[i] * t * 0.15) for i in range(3))
                _set_px(buf, x, y, c)


def render() -> bytes:
    buf = bytearray(SIZE * SIZE * 4)
    _fill_round_rect(buf, (37, 37, 38))
    teal = (78, 201, 176)
    blue = (86, 156, 214)
    orange = (206, 145, 120)
    _stroke_circle(buf, 128, 128, 88, (60, 60, 60), 4)
    _stroke_circle(buf, 128, 128, 64, teal, 3)
    _stroke_circle(buf, 128, 128, 40, teal, 3)
    for angle in range(0, 90):
        import math

        rad = math.radians(angle - 90)
        x = int(128 + 88 * math.cos(rad))
        y = int(128 + 88 * math.sin(rad))
        _set_px(buf, x, y, teal)
    for t in range(0, 70):
        x = int(128 + t * 0.85)
        y = int(128 - t * 0.55)
        _set_px(buf, x, y, blue)
        _set_px(buf, x + 1, y, blue)
    for dx, dy, c in ((0, 0, teal), (40, -32, orange), (-32, 28, (220, 220, 170))):
        cx, cy = 128 + dx, 128 + dy
        for y in range(int(cy) - 8, int(cy) + 9):
            for x in range(int(cx) - 8, int(cx) + 9):
                if (x - cx) ** 2 + (y - cy) ** 2 <= 64:
                    _set_px(buf, x, y, c)
    return bytes(buf)


def main() -> None:
    write_png(OUT, render(), SIZE, SIZE)
    print(f"OK — {OUT} ({SIZE}x{SIZE})")


if __name__ == "__main__":
    main()
