#!/usr/bin/env bash
# Cross-compile gobi for a GOOS/GOARCH pair. Usage: build-cross.sh <linux|windows> [amd64|arm64]
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

target="${1:?usage: build-cross.sh <linux|windows> [amd64|arm64]}"
arch="${2:-amd64}"

case "$target" in
linux)
	export GOOS=linux
	ext=""
	;;
windows)
	export GOOS=windows
	ext=".exe"
	;;
*)
	echo "unsupported target: $target (want linux or windows)" >&2
	exit 1
	;;
esac

case "$arch" in
amd64 | arm64) ;;
*)
	echo "unsupported arch: $arch (want amd64 or arm64)" >&2
	exit 1
	;;
esac

export GOARCH="$arch"
export CGO_ENABLED=0

mkdir -p bin
out="bin/gobi-${GOOS}-${GOARCH}${ext}"
go build -o "$out" ./gobi/cmd/gobi
echo "$out"
