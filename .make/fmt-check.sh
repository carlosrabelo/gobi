#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
if [ ! -d gobi ]; then
	exit 0
fi
mapfile -t files < <(find gobi -name '*.go' -print)
if [ "${#files[@]}" -eq 0 ]; then
	exit 0
fi
unformatted="$(gofmt -l "${files[@]}")"
if [ -n "$unformatted" ]; then
	echo "$unformatted" >&2
	exit 1
fi
