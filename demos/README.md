# Gobi demos

Sample `.dbf` tables and `.prg` command files for the interactive interpreter.

## Tables

| File | Contents |
| --- | --- |
| `customers.dbf` | Customer code, name, city, balance |
| `orders.dbf` | Order id, customer code, amount, date |
| `products.dbf` | SKU, description, price, quantity |

Regenerate the tables (overwrites the `.dbf` files):

```bash
go run ./gobi/cmd/gendata
```

## Run from the repo root

Build once, then run a script (paths assume the working directory is the repo root):

```bash
make build
bin/gobi demos/hello.prg
bin/gobi demos/list_customers.prg
bin/gobi demos/filter_balance.prg
bin/gobi demos/index_seek.prg
bin/gobi demos/relation.prg
bin/gobi demos/products_report.prg
```

Or from the prompt:

```text
. SET DEFAULT TO demos
. DO hello
. DO list_customers
. USE customers
. LIST
. QUIT
```

`index_seek.prg` and `relation.prg` create `.ndx` files under `demos/` (`custcode.ndx`, `ordcode.ndx`). Those are gitignored and safe to delete; the scripts rebuild them with `SET SAFETY OFF`.
