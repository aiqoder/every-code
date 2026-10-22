#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

APP_NAME="chengzi-agent-server"
CMD_PATH="./cmd/server"
DIST_DIR="${DIST_DIR:-dist/server}"
VERSION="${VERSION:-dev}"
SKIP_WEB="${SKIP_WEB:-0}"

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}/bin" "${DIST_DIR}/web" "${DIST_DIR}/packages" "${DIST_DIR}/staging"

if [[ "${SKIP_WEB}" != "1" ]]; then
  echo "building admin web"
  (
    cd web
    npm ci --no-audit --no-fund
    test -x node_modules/.bin/vite
    npm run build-only
  )
fi
if [[ -d web/dist && -f web/dist/index.html ]]; then
  cp -R web/dist/. "${DIST_DIR}/web/"
fi

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
  fi

  local bin_name="${APP_NAME}-${goos}-${goarch}${ext}"
  echo "building ${bin_name}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${DIST_DIR}/bin/${bin_name}" "${CMD_PATH}"

  local pkg_name="${APP_NAME}-${goos}-${goarch}"
  local pkg_dir="${DIST_DIR}/staging/${pkg_name}"
  rm -rf "${pkg_dir}"
  mkdir -p "${pkg_dir}/etc" "${pkg_dir}/web"
  cp "${DIST_DIR}/bin/${bin_name}" "${pkg_dir}/${APP_NAME}${ext}"
  cp etc/chengzi.yaml "${pkg_dir}/etc/"
  if [[ -d "${DIST_DIR}/web" && -n "$(ls -A "${DIST_DIR}/web" 2>/dev/null || true)" ]]; then
    cp -R "${DIST_DIR}/web/." "${pkg_dir}/web/"
  fi

  if [[ "${goos}" == "windows" ]]; then
    python3 - <<PY
import pathlib, zipfile
root = pathlib.Path("${DIST_DIR}/staging")
pkg = root / "${pkg_name}"
out = pathlib.Path("${DIST_DIR}/packages/${pkg_name}-${VERSION}.zip")
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as zf:
    for path in pkg.rglob("*"):
        if path.is_file():
            zf.write(path, path.relative_to(root))
PY
  else
    tar -C "${DIST_DIR}/staging" -czf "${DIST_DIR}/packages/${pkg_name}-${VERSION}.tar.gz" "${pkg_name}"
  fi
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64
build_one windows arm64

echo "verifying binaries"
DIST_DIR="${DIST_DIR}" python3 - <<'PY'
import os
import pathlib
import struct
import sys

dist = pathlib.Path(os.environ["DIST_DIR"]) / "bin"
expected = {
    "chengzi-agent-server-linux-amd64": ("elf", 62),
    "chengzi-agent-server-linux-arm64": ("elf", 183),
    "chengzi-agent-server-darwin-amd64": ("macho", 0x01000007),
    "chengzi-agent-server-darwin-arm64": ("macho", 0x0100000C),
    "chengzi-agent-server-windows-amd64.exe": ("pe", 0x8664),
    "chengzi-agent-server-windows-arm64.exe": ("pe", 0xAA64),
}

errors = []
for name, spec in expected.items():
    path = dist / name
    if not path.is_file() or path.stat().st_size < 1024:
        errors.append(f"{name}: missing or too small")
        continue
    data = path.read_bytes()[:64]
    kind, want = spec
    if kind == "elf":
        if data[:4] != b"\x7fELF":
            errors.append(f"{name}: not ELF")
            continue
        machine = struct.unpack_from("<H", data, 18)[0]
        if machine != want:
            errors.append(f"{name}: ELF machine={machine}, want {want}")
    elif kind == "macho":
        magic = struct.unpack_from(">I", data, 0)[0]
        if magic not in (0xCFFAEDFE, 0xCEFAEDFE, 0xFEEDFACF, 0xFEEDFACE):
            errors.append(f"{name}: not Mach-O")
            continue
        endian = "<" if magic in (0xCFFAEDFE, 0xCEFAEDFE) else ">"
        cputype = struct.unpack_from(endian + "I", data, 4)[0]
        if cputype != want:
            errors.append(f"{name}: Mach-O cputype=0x{cputype:x}, want 0x{want:x}")
    elif kind == "pe":
        if data[:2] != b"MZ":
            errors.append(f"{name}: not PE")
            continue
        pe_off = struct.unpack_from("<I", data, 0x3C)[0]
        pe = path.read_bytes()[pe_off:pe_off + 8]
        if pe[:4] != b"PE\x00\x00":
            errors.append(f"{name}: missing PE header")
            continue
        machine = struct.unpack_from("<H", pe, 4)[0]
        if machine != want:
            errors.append(f"{name}: PE machine=0x{machine:x}, want 0x{want:x}")

if errors:
    print("\n".join(errors), file=sys.stderr)
    sys.exit(1)
print("binary formats ok")
PY

echo "done -> ${DIST_DIR}/packages/"
ls -lh "${DIST_DIR}/packages/"
