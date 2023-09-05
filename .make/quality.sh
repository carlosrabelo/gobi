#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
./.make/fmt-check.sh
./.make/lint.sh
./.make/test.sh
