* list_customers.prg — open a table and list records
SET TALK OFF
SET DEFAULT TO demos
USE customers
? "=== customers ==="
LIST
COUNT TO N
? "records:", N
USE
RETURN
