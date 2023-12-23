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

```bash
git clone https://github.com/carlosrabelo/gobi.git
cd gobi
make build
make install          # ~/.local/bin/gobi
# make install-system # /usr/local/bin/gobi
```

## Usage

```bash
gobi                  # dot prompt
gobi script.prg       # run a command file, then exit
gobi -e 'USE c; LIST' # run semicolon-separated commands, then exit
make run -- -e QUIT   # build if needed and execute bin/gobi
bin/gobi demos/hello.prg   # sample tables and .prg files under demos/
```

## Development

```bash
make build            # write bin/gobi
make run              # build if needed; run bin/gobi (args after --)
make install          # copy bin/gobi to ~/.local/bin (no compile)
make install-system   # copy bin/gobi to /usr/local/bin (no compile)
make uninstall
make uninstall-system
make test             # go test ./...
make test-cover       # coverage.out and coverage.html at the git root
make fmt              # gofmt -w on sources under gobi/
make lint             # go vet ./...
make quality          # gofmt -l check, lint, and test
make clean            # remove bin/* (keep .gitkeep), coverage.out, coverage.html
```

## License

MIT. See [LICENSE](LICENSE).
