#!/usr/bin/env bash
set -euo pipefail

APP_NAME="chengzi-agent-server"
CMD_PATH="./cmd/server"
BUILD_DIR="build"

mkdir -p "${BUILD_DIR}"

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
  fi
  local out="${BUILD_DIR}/${APP_NAME}-${goos}-${goarch}${ext}"
  echo "building ${out}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${out}" "${CMD_PATH}"
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64
build_one windows arm64

echo "done -> ${BUILD_DIR}/"
