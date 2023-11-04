#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
if [ ! -x bin/gobi ]; then
	echo "bin/gobi missing; run make build first" >&2
	exit 1
fi
cp bin/gobi /usr/local/bin/gobi
