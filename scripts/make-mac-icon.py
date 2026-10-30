#!/usr/bin/env python3
"""给 macOS 图标加上系统程序坞要求的透明边。

Apple 图标网格是 1024 画布里居中的 824 内容区。满幅图会被程序坞铺满整格，
看起来比其他图标大一圈。Linux / Windows 仍用满幅的 build/icon.png、build/icon.ico。

用法：python3 scripts/make-mac-icon.py
"""

from __future__ import annotations

import struct
import subprocess
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "build" / "icon.png"
DOCK_ICON = ROOT / "resources" / "icon-mac.png"
ICNS = ROOT / "build" / "icon.icns"

CANVAS = 1024
CONTENT = 824  # (1024 - 824) / 2 = 100px 边距


def read_png(path: Path):
    data = path.read_bytes()
    if data[:8] != b"\x89PNG\r\n\x1a\n":
        raise SystemExit(f"不是 PNG: {path}")
    pos = 8
    width = height = color_type = None
    idat = b""
    while pos < len(data):
        length = struct.unpack(">I", data[pos : pos + 4])[0]
        ctype = data[pos + 4 : pos + 8]
        chunk = data[pos + 8 : pos + 8 + length]
        pos += 12 + length
        if ctype == b"IHDR":
            width, height, bit_depth, color_type = struct.unpack(">IIBB", chunk[:10])
            if bit_depth != 8 or color_type not in (2, 6):
                raise SystemExit(f"只支持 8-bit RGB/RGBA: {path}")
        elif ctype == b"IDAT":
            idat += chunk
        elif ctype == b"IEND":
            break
    raw = zlib.decompress(idat)
    channels = 4 if color_type == 6 else 3
    stride = width * channels
    rows = []
    index = 0
    prev = bytearray(stride)
    for _y in range(height):
        filter_type = raw[index]
        index += 1
        row = bytearray(raw[index : index + stride])
        index += stride
        if filter_type == 1:
            for x in range(stride):
                left = row[x - channels] if x >= channels else 0
                row[x] = (row[x] + left) & 255
        elif filter_type == 2:
            for x in range(stride):
                row[x] = (row[x] + prev[x]) & 255
        elif filter_type == 3:
            for x in range(stride):
                left = row[x - channels] if x >= channels else 0
                row[x] = (row[x] + ((left + prev[x]) // 2)) & 255
        elif filter_type == 4:
            for x in range(stride):
                left = row[x - channels] if x >= channels else 0
                up = prev[x]
                up_left = prev[x - channels] if x >= channels else 0
                estimate = left + up - up_left
                pa, pb, pc = abs(estimate - left), abs(estimate - up), abs(estimate - up_left)
                predictor = left if pa <= pb and pa <= pc else (up if pb <= pc else up_left)
                row[x] = (row[x] + predictor) & 255
        elif filter_type != 0:
            raise SystemExit(f"不支持的 PNG 过滤器: {filter_type}")
        if channels == 3:
            rgba = bytearray(width * 4)
            for x in range(width):
                rgba[x * 4 : x * 4 + 3] = row[x * 3 : x * 3 + 3]
                rgba[x * 4 + 3] = 255
            row = rgba
        rows.append(bytes(row))
        prev = row
    return width, height, rows


def write_png(path: Path, width: int, height: int, rows: list[bytes]) -> None:
    raw = b"".join(b"\x00" + row for row in rows)
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)

    def chunk(tag: bytes, payload: bytes) -> bytes:
        return (
            struct.pack(">I", len(payload))
            + tag
            + payload
            + struct.pack(">I", zlib.crc32(tag + payload) & 0xFFFFFFFF)
        )

    path.write_bytes(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )


def padded_canvas(content_png: Path) -> tuple[int, int, list[bytes]]:
    width, height, rows = read_png(content_png)
    if width != CONTENT or height != CONTENT:
        raise SystemExit(f"缩放结果应为 {CONTENT}x{CONTENT}，实际 {width}x{height}")
    offset = (CANVAS - CONTENT) // 2
    canvas = [b"\x00" * (CANVAS * 4) for _ in range(CANVAS)]
    stride = CONTENT * 4
    for y, row in enumerate(rows):
        start = offset * 4
        dest = bytearray(canvas[offset + y])
        dest[start : start + stride] = row
        canvas[offset + y] = bytes(dest)
    return CANVAS, CANVAS, canvas


def build_icns(master: Path) -> None:
    iconset = Path("/tmp/every-code-mac.iconset")
    if iconset.exists():
        for child in iconset.iterdir():
            child.unlink()
    else:
        iconset.mkdir()
    sizes = {
        "icon_16x16.png": 16,
        "icon_16x16@2x.png": 32,
        "icon_32x32.png": 32,
        "icon_32x32@2x.png": 64,
        "icon_128x128.png": 128,
        "icon_128x128@2x.png": 256,
        "icon_256x256.png": 256,
        "icon_256x256@2x.png": 512,
        "icon_512x512.png": 512,
        "icon_512x512@2x.png": 1024,
    }
    for name, size in sizes.items():
        if size == CANVAS:
            (iconset / name).write_bytes(master.read_bytes())
            continue
        subprocess.check_call(
            ["sips", "-z", str(size), str(size), str(master), "--out", str(iconset / name)],
            stdout=subprocess.DEVNULL,
        )
    subprocess.check_call(["iconutil", "-c", "icns", str(iconset), "-o", str(ICNS)])


def main() -> None:
    if not SOURCE.exists():
        raise SystemExit(f"找不到源图标: {SOURCE}")
    content = Path("/tmp/every-code-icon-824.png")
    subprocess.check_call(
        ["sips", "-z", str(CONTENT), str(CONTENT), str(SOURCE), "--out", str(content)],
        stdout=subprocess.DEVNULL,
    )
    width, height, rows = padded_canvas(content)
    write_png(DOCK_ICON, width, height, rows)
    build_icns(DOCK_ICON)
    print(f"wrote {DOCK_ICON.relative_to(ROOT)}")
    print(f"wrote {ICNS.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
