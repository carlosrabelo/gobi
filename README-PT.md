# Gobi

Clone do dBase II totalmente funcional escrito em Go, implementando o prompt de ponto interativo clássico, motor de expressões, comandos de banco de dados, índices B-Tree e uma TUI VT100 retrô.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://go.dev/)

## Destaques

- REPL interativo com prompt de ponto (dot prompt) com histórico de comandos e edição de linha
- Motor de avaliação de expressões com suporte a variáveis de memória, campos de banco de dados, operações lógicas e funções embutidas
- Leitura e gravação física de alta fidelidade para arquivos de banco de dados `.dbf` clássicos do dBase II
- Motor de índices NDX baseado em disco implementando B-Trees balanceadas em páginas de 512 bytes
- Suporte a scripting procedural (scripts `.prg`) com loops, desvios condicionais e aninhamento de rotinas
- Interface de terminal compatível com VT100 retrô com suporte a layouts `@ SAY / GET` e telas de dados `READ`
- Visualizador e editor interativo de dados em formato de planilha em tela cheia via comando `BROWSE`

## Pré-requisitos

- **Go 1.21+** — necessário para compilar a partir do código-fonte; [download](https://go.dev/dl/)

## Instalação

### Compilação a partir do código-fonte

```bash
git clone https://github.com/carlosrabelo/gobi.git
cd gobi
make build
```

Instale em `~/.local/bin` (padrão), ou em `/usr/local/bin` no sistema (sudo apenas para a cópia):

```bash
make install
make install-system
make uninstall
make uninstall-system
```

### Usando Go Install

```bash
go install github.com/carlosrabelo/gobi/gobi/cmd/gobi@latest
```

## Início Rápido

Inicie o console interativo:

```bash
make build
./bin/gobi
```

Ou execute o script de demonstração incluído:

```bash
./bin/gobi demos/people.prg
```

No prompt do Gobi, use comandos clássicos para abrir e consultar tabelas de dados:

```
. SET DEFAULT demos
. USE people
. LIST NAME, AGE FOR AGE > 30
. QUIT
```

## Uso

### Console Interativo (Dot Prompt)

Inicie o Gobi sem parâmetros adicionais para entrar no interpretador de comandos clássico:

```bash
gobi
```

### Execução de Scripts

Execute um programa de script dBase diretamente:

```bash
gobi demos/people.prg
```

### Execução de Comandos Inline

Execute comandos separados por ponto e vírgula diretamente no terminal e saia em seguida:

```bash
gobi -e "SET DEFAULT demos; USE people; COUNT"
```

## Estrutura do Projeto

```
gobi/cmd/gobi/       # Ponto de entrada Go e shell REPL
gobi/internal/       # Handlers de comandos internos e contexto do REPL
gobi/pkg/dbf/        # Parser de arquivos DBF, decodificadores e escrita
gobi/pkg/docs/       # Especificação de linguagem embutida para o HELP
gobi/pkg/expr/       # Lexer, parser e avaliador de expressões
gobi/pkg/ndx/        # Motor de índices NDX (páginas B-Tree de 512 bytes)
gobi/pkg/script/     # Carregador e controlador de scripts `.prg`
docs/                # Especificações de formatos de arquivo
bin/                 # Binários compilados (ignorado pelo git)
.make/               # Scripts de build e instalação
demos/               # Bancos de dados e scripts de demonstração
```

## Desenvolvimento

```bash
make build             # Compila o binário para bin/gobi
make test              # Executa todos os testes
make quality           # Formata, verifica e valida (lint) o código
make install           # Instala o binário em ~/.local/bin
make install-system    # Instala o binário em /usr/local/bin
make uninstall         # Remove de ~/.local/bin
make uninstall-system  # Remove de /usr/local/bin
```

## Licença

Este projeto está licenciado sob a Licença MIT — consulte o arquivo [LICENSE](LICENSE) para obter detalhes.
