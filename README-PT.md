# Gobi

Clone completo do dBase III Plus em Go, com shell do host e terminais Linux/Windows.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://go.dev/)

## Highlights

- Sessão completa dBase III Plus com o prompt clássico de ponto
- Lê e grava tabelas `.dbf`, memos `.dbt`, índices `.ndx` e arquivos `.mem` do dBase III
- Motor de expressões com o conjunto de funções do III Plus, datas e operadores de string
- Dez áreas de trabalho, relações, filtros, procedimentos e arquivos de comando
- Tela cheia com `@ SAY` / `@ GET`, `READ`, `EDIT` e `BROWSE`
- Tamanho e capacidades do terminal hospedeiro em Linux e Windows; `RUN` / `!` usam o shell do host
- HELP estruturado (categorias, uso, exemplos) e Assist estável em ecrã completo
- Rebuild guiado por specs: `specs/` é o sílabo (prompt primeiro); o código pode ser recriado a partir dele

## Prerequisites

- Go 1.21 ou mais novo — [download](https://go.dev/dl/)

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
gobi                  # prompt com ponto
gobi script.prg       # executa um arquivo de comandos e sai
gobi -e 'USE c; LIST' # comandos separados por ponto e vírgula, depois sai
make run -- -e QUIT   # compila se preciso e executa bin/gobi
```

## Development

```bash
make build            # grava bin/gobi neste host
make build-linux      # bin/gobi-linux-amd64 (ARCH=arm64 opcional)
make build-windows    # bin/gobi-windows-amd64.exe
make build-all        # linux + windows amd64
make run              # compila se preciso; executa bin/gobi (args depois de --)
make install          # copia bin/gobi para ~/.local/bin (sem compilar)
make install-system   # copia bin/gobi para /usr/local/bin (sem compilar)
make uninstall
make uninstall-system
make test             # go test ./...
make test-cover       # coverage.out e coverage.html na raiz do git
make fmt              # gofmt -w nos fontes em gobi/
make lint             # go vet ./...
make quality          # checagem gofmt -l, lint e test
make clean            # remove bin/* (mantém .gitkeep), coverage.out e coverage.html
```

## License

MIT. Veja [LICENSE](LICENSE).
