#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
packages="$(go list ./... 2>/dev/null || true)"
if [ -z "$packages" ]; then
	exit 0
fi
go test -v ./...
