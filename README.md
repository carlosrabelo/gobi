# Gobi

Fully functional dBase II clone written in Go, implementing the classic interactive dot prompt, expression engine, database commands, B-Tree indexes, and a retro VT100 TUI.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://go.dev/)

## Highlights

- Interactive dot prompt REPL with command history and command-line editing
- Expression evaluation engine supporting memory variables, database fields, logical operations, and built-in functions
- High-fidelity parsing and writing of standard dBase II `.dbf` database files
- Disk-backed NDX index engine implementing balanced B-Trees in 512-byte pages
- Procedural scripting support (`.prg` scripts) with loops, conditional branching, and procedure nesting
- Retro VT100-compliant terminal UI supporting `@ SAY / GET` layouts and `READ` screen forms
- Interactive full-screen spreadsheet `BROWSE` data editor

## Prerequisites

- **Go 1.21+** — required to build from source; [download](https://go.dev/dl/)

## Installation

### Build from Source

```bash
git clone https://github.com/carlosrabelo/gobi.git
cd gobi
make build
```

Install to `~/.local/bin` (default), or system-wide to `/usr/local/bin` (sudo only for the copy):

```bash
make install
make install-system
make uninstall
make uninstall-system
```

### Using Go Install

```bash
go install github.com/carlosrabelo/gobi/gobi/cmd/gobi@latest
```

## Quick Start

Run the interactive shell:

```bash
make build
./bin/gobi
```

Or run the included demo script:

```bash
./bin/gobi demos/people.prg
```

Inside the Gobi shell, use standard commands to open and inspect database tables:

```
. SET DEFAULT demos
. USE people
. LIST NAME, AGE FOR AGE > 30
. QUIT
```

## Usage

### Interactive Shell (Dot Prompt)

Launch Gobi without parameters to drop into the classic dot prompt shell:

```bash
gobi
```

### Running Scripts

Execute a dBase script program directly:

```bash
gobi demos/people.prg
```

### Executing Inline Commands

Run semicolon-separated commands directly from your terminal and exit:

```bash
gobi -e "SET DEFAULT demos; USE people; COUNT"
```

## Project Layout

```
gobi/cmd/gobi/       # Go entry point and REPL shell
gobi/internal/       # Internal command handlers and REPL context
gobi/pkg/dbf/        # DBF file parser, decoders, and writer
gobi/pkg/docs/       # Embedded language specification for HELP
gobi/pkg/expr/       # Expression lexer, parser, and evaluator
gobi/pkg/ndx/        # NDX index engine (512-byte B-Tree pages)
gobi/pkg/script/     # `.prg` script loader and controller
docs/                # File format specifications
bin/                 # Compiled binaries (git-ignored)
.make/               # Build and install scripts
demos/               # Sample databases and demo scripts
```

## Development

```bash
make build             # Compile binary to bin/gobi
make test              # Run all tests
make quality           # Format, vet, and lint
make install           # Install binary to ~/.local/bin
make install-system    # Install binary to /usr/local/bin
make uninstall         # Remove from ~/.local/bin
make uninstall-system  # Remove from /usr/local/bin
```

## License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.
