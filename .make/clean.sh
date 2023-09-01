#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
find bin -mindepth 1 -maxdepth 1 ! -name '.gitkeep' -exec rm -rf {} +
rm -f coverage.out coverage.html
