* products_report.prg — stock value with a DO WHILE loop
SET TALK OFF
SET DEFAULT TO demos
USE products
STORE 0 TO TOTAL
? "SKU    DESCRIPTION              PRICE   QTY   LINE"
? "------ ------------------------ ------- ----- --------"
GO TOP
DO WHILE .NOT. EOF()
  STORE ROUND(PRICE * QTY, 2) TO LINE
  STORE ROUND(TOTAL + LINE, 2) TO TOTAL
  ? SKU, DESCR, PRICE, QTY, LINE
  SKIP
ENDDO
? "------ ------------------------ ------- ----- --------"
? "TOTAL STOCK VALUE:", TOTAL
USE
RETURN
