REMARK Gobi demo - people database
REMARK Run from repo root: gobi demos/people.prg

SET DEFAULT demos
USE people

REMARK All records
LIST

REMARK Active members only
LIST NAME, AGE FOR ACTIVE

REMARK Record counts
COUNT
COUNT FOR AGE > 30

REMARK Average age (DO WHILE loop)
SET TALK OFF
COUNT TO recCount
STORE 0 TO total
STORE 0 TO n
GO TOP
DO WHILE n < recCount
  STORE total + AGE TO total
  STORE n + 1 TO n
  SKIP
ENDDO
? total / n

RETURN
