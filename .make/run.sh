#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME="${BINARY_NAME:-gobi}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
if [ ! -x "bin/$BINARY_NAME" ]; then
	./.make/build.sh
fi
exec "bin/$BINARY_NAME" "$@"
