#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

go clean
rm -rf bin/
mkdir -p bin
touch bin/.gitkeep
echo "Cleaned build artifacts."
