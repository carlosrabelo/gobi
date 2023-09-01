# Gobi

Interpretador dBase III Plus escrito em Go.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://go.dev/)

## Highlights

- Sessão interativa dBase III Plus com o prompt clássico de ponto
- Lê e grava tabelas `.dbf`, memos `.dbt`, índices `.ndx` e arquivos `.mem` do dBase III
- Motor de expressões com o conjunto de funções do III Plus, datas e operadores de string
- Dez áreas de trabalho, relações, filtros, procedimentos e arquivos de comando
- Tela cheia com `@ SAY` / `@ GET`, `READ`, `EDIT` e `BROWSE`
- Reescrita guiada por specs: o comportamento fica em `specs/` antes dos pacotes

## Prerequisites

- Go 1.21 ou mais novo — [download](https://go.dev/dl/)

## Installation

Este repositório ainda não tem programa hospedeiro. A spec `019` acrescenta o binário `gobi` e os alvos de instalação. Até lá, clone a árvore e use os alvos de Development abaixo.

```bash
git clone https://github.com/carlosrabelo/gobi.git
cd gobi
```

## Usage

Não há programa hospedeiro nesta spec. Os contribuidores trabalham pelos alvos do Makefile em Development.

## Development

```bash
make build     # go build ./... (não grava bin/gobi)
make test      # go test ./...
make fmt       # gofmt -w nos fontes em gobi/
make lint      # go vet ./...
make quality   # fmt, lint e test
make clean     # remove bin/* (mantém .gitkeep), coverage.out e coverage.html
```

## License

MIT. Veja [LICENSE](LICENSE).
