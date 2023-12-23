* index_seek.prg — INDEX ON and SEEK
SET TALK OFF
SET SAFETY OFF
SET DEFAULT TO demos
USE customers
INDEX ON CODE TO custcode
SET INDEX TO custcode
SEEK "C003"
IF FOUND()
  ? "found:", CODE, NAME, CITY
ELSE
  ? "not found"
ENDIF
USE
RETURN
