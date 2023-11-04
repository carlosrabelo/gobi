#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
if [ ! -x bin/gobi ]; then
	./.make/build.sh
fi
exec bin/gobi "$@"
