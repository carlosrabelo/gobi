# Gobi

dBase III Plus interpreter written in Go.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://go.dev/)

## Highlights

- Interactive dBase III Plus session with the classic dot prompt
- Reads and writes dBase III `.dbf` tables, `.dbt` memos, `.ndx` indexes, and `.mem` files
- Expression engine with the III Plus function set, dates, and string operators
- Ten work areas, relations, filters, procedures, and command files
- Full-screen `@ SAY` / `@ GET`, `READ`, `EDIT`, and `BROWSE`
- Spec-driven rewrite: behavior is defined under `specs/` before packages land

## Prerequisites

- Go 1.21 or newer — [download](https://go.dev/dl/)

## Installation

This repository has no host program yet. Spec `019` adds the `gobi` binary and the install targets. Until then, clone the tree and use the Development targets below.

```bash
git clone https://github.com/carlosrabelo/gobi.git
cd gobi
```

## Usage

There is no host program in this spec. Contributors work from the Makefile targets in Development.

## Development

```bash
make build     # go build ./... (does not write bin/gobi)
make test      # go test ./...
make fmt       # gofmt -w on sources under gobi/
make lint      # go vet ./...
make quality   # fmt, lint, and test
make clean     # remove bin/* (keep .gitkeep), coverage.out, coverage.html
```

## License

MIT. See [LICENSE](LICENSE).
