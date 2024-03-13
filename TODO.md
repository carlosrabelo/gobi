# Gobi

## First milestone

Interactive `gobi` REPL and program executor runs, parsing DBF files, executing commands via dot prompt, evaluating expressions, and running script programs. NDX index and full TUI screens are later milestones.

## Foundation

- [x] Project layout setup (cmd, internal, pkg structure)
- [x] Root Makefile with test, lint, format and build targets
- [x] Configuration struct for global environment variables
- [x] Central execution context manager

## Expression Engine

- [x] Token definitions (literals, fields, variables, operators)
- [x] Lexer scanning strings and numbers
- [x] Lexer scanning logical constants (.T., .F.)
- [x] Lexer scanning operators and parenthesis
- [x] AST structures for binary, unary, and function nodes
- [x] Pratt parser or recursive descent expression parser
- [x] Evaluator for literal types (string, float64, bool)
- [x] Evaluator field lookup in active DBF record
- [x] Evaluator memory variable lookup
- [x] Short-circuit logic for .AND. and .OR. operators
- [x] Built-in function: EOF()
- [x] Built-in function: RECNO()
- [x] Built-in function: DELETED()
- [x] Built-in function: FOUND()
- [x] Built-in function: TRIM()
- [x] Built-in function: UPPER() and LOWER()
- [x] Built-in function: LEN() and SUBSTR()
- [x] Built-in function: VAL() and STR()

## REPL and Command Shell

References: [gobi/pkg/docs/language_spec.md](gobi/pkg/docs/language_spec.md)

- [x] Basic console line reader with '.' prompt
- [x] Command history log file and keyboard scrolling
- [x] Command parser splitting verb and string arguments
- [x] Parser support for conditional FOR clause
- [x] Parser support for conditional WHILE clause
- [x] Parser support for TO output redirect path
- [x] Command multiplexer matching verb to execution handler

## DBF File Layout

References: [docs/dbf_spec.md](file:///home/carlos/Sources/gobi/docs/dbf_spec.md)

- [x] Read header signature (0x02 or 0x82)
- [x] Read record count (2 bytes) and record length (2 bytes)
- [x] Read field count and iterate 16-byte field descriptors
- [x] Map field descriptors into memory structs
- [x] Header terminator validation (0x0D)
- [x] Raw record reader with deletion flag check (0x20 vs 0x2A)
- [x] Field decoder for Character (C) - space-stripped strings
- [x] Field decoder for Numeric (N) - parsing ASCII into float64
- [x] Field decoder for Logical (L) - parsing T/F/Y/N to bool
- [x] Raw record writer with field padding
- [x] End-Of-File (EOF) marker logic (0x1A)
- [x] In-place record update by record number offset
- [x] Append new record helper updating header count
- [x] Safe file flush and sync on close

## Database Commands

- [x] `USE` command opening target DBF file
- [x] `SELECT` switching between Primary and Secondary areas
- [x] `CLOSE DATABASES` and `CLOSE INDEX`
- [x] `DISPLAY STRUCTURE` and `LIST STRUCTURE`
- [x] `GOTO` absolute record position
- [x] `GO TOP` and `GO BOTTOM`
- [x] `SKIP` moving cursor relative to active database size
- [x] `LIST` showing fields with FOR/WHILE filtering
- [x] `DISPLAY` paginating records list to stdout
- [x] `APPEND` line prompt adding records interactively
- [x] `REPLACE` field value evaluations
- [x] `DELETE` marking active record for deletion
- [x] `RECALL` restoring deleted record marker
- [x] `PACK` rewriting DBF to drop deleted records
- [x] `ZAP` truncating active DBF data area
- [x] `CREATE` interactively defining schema and creating new DBF
- [x] `EDIT` interactive full-screen form record editor
- [x] `MODIFY STRUCTURE` changing schema of active DBF
- [x] `COPY TO` exporting records/schema to a new DBF
- [x] `APPEND FROM` importing records from another DBF or text file
- [x] `UPDATE FROM` modifying active DBF records using secondary area data
- [x] `JOIN` combining active and secondary DBFs into a new table
- [x] `TOTAL ON` summarizing numeric fields to a target DBF
- [x] `LOCATE` and `CONTINUE` sequential search matching condition
- [x] `COUNT` records matching conditional filter
- [x] `SUM` totaling numeric field values matching filter
- [x] `AVERAGE` calculating mean of numeric fields matching filter
- [x] `?` evaluate and display expression value with newline
- [x] `??` evaluate and display expression value without newline

## Memory Variables

- [x] Global symbol registry mapping names to variable values
- [x] `STORE` assignment parser and executor
- [x] `DISPLAY MEMORY` and `LIST MEMORY` tabular printer
- [x] `SAVE TO` binary serializer saving variables to .mem file
- [x] `RESTORE FROM` deserializer loading variables
- [x] `RELEASE` cleaning symbols from registry

## Programming & Scripting

- [x] `DO` command loader checking file existence
- [x] PRG line-by-line script reader and parser
- [x] Script instruction pointer controller
- [x] `IF ... ELSE ... ENDIF` parser and branch executor
- [x] `DO WHILE ... ENDDO` loop manager
- [x] `LOOP` and `EXIT` jump resolution
- [x] Nesting call stack for parent/child scripts execution
- [x] `RETURN` pops caller state from call stack
- [x] `CANCEL` clears script execution stack and halts to REPL

## Indexing (.ndx)

References: [docs/ndx_spec.md](file:///home/carlos/Sources/gobi/docs/ndx_spec.md)

- [x] NDX page 0 header structure serialization
- [x] B-Tree in-memory representations for nodes and entries
- [x] Disk page manager allocating 512-byte blocks
- [x] Internal node split on page overflow
- [x] Leaf node entry insertion and record mapping
- [x] Exact key search traversing tree pages
- [x] Prefix key search mapping to first matches
- [x] Leaf node entry deletion on key removals
- [x] `INDEX ON` command scanning DBF and constructing tree
- [x] Multi-index synchronization during table `APPEND`
- [x] Multi-index synchronization during table `REPLACE`
- [x] `REINDEX` command re-building active trees
- [x] `FIND` and `SEEK` commands setting DBF cursor
- [x] `SORT ON` sorting DBF records physically

## Screen and TUI (VT100)

- [x] ANSI escape wrapper for cursor positioning and colors
- [x] Terminal raw-mode keyboard capture adapter
- [x] Double-buffered terminal frame writer
- [x] `CLEAR` command clearing screen buffer
- [x] `@ SAY` printing expression string at coordinates
- [x] `@ GET` registering interactive input fields
- [x] `READ` loop handling user editing, Tab, and field validation
- [x] Interactive `APPEND` screen generation
- [x] Interactive `EDIT` record screen generation
- [x] `BROWSE` table view spreadsheet matrix
- [x] `BROWSE` cell cursor movement using arrow keys
- [x] `BROWSE` inline cell edit and record deletion

## System Environment & CLI

- [x] `SET TALK` flag controller
- [x] `SET INTENSITY` video invert controller
- [x] `SET BELL` sound alarm controller
- [x] `SET DEFAULT` data directory mapping
- [x] `DIR` command listing DBFs with sizes and records count
- [x] `RENAME` and `ERASE` file hooks
- [x] Entry point arguments (`-e` inline command, file run)
- [x] `HELP` command documentation parser

## Second milestone

## Console & Program Input

- [x] `ACCEPT ['prompt'] TO <var>` string input into memory variable
- [x] `INPUT ['prompt'] TO <var>` evaluated expression input
- [x] `WAIT [TO <var>]` single-key pause storing the pressed key
- [x] `TEXT ... ENDTEXT` literal text block output in scripts
- [x] `REMARK` echoing its text to output (currently treated as silent comment)
- [x] `NOTE` silent comment lines in scripts (alias of `*`)

## Flow Control

- [x] `DO CASE / CASE <expr> / OTHERWISE / ENDCASE` branch structure

## Record Operations & Scopes

- [x] Scope clauses `ALL` and `NEXT <n>` on record commands (LIST, DISPLAY, DELETE, RECALL, REPLACE, COUNT, SUM, LOCATE)
- [x] `APPEND BLANK` adding an empty record without prompting
- [x] `INSERT [BEFORE] [BLANK]` inserting a record at the cursor position
- [x] `CHANGE [<scope>] FIELD <list> [FOR <expr>]` line-mode field editor

## Command Semantics Alignment

- [x] `ERASE` clearing the screen (dBase II semantics)
- [x] `DELETE FILE <filename>` removing files (replaces current ERASE behavior)
- [x] `CLEAR` closing databases and releasing memory variables (dBase II semantics)
- [x] `CLEAR GETS` releasing pending @ GET registrations
- [x] `DISPLAY FILES` / `LIST FILES [LIKE <pattern>]` directory listing (DIR as alias)
- [ ] `DISPLAY STATUS` / `LIST STATUS` showing open files, indexes, and SET values

## Index Integration

- [ ] `USE <file> INDEX <ndx1>, ...` binding existing index files
- [ ] `SET INDEX TO [<ndx list>]` rebinding indexes on the active table
- [ ] Index-ordered navigation: GO TOP/BOTTOM, SKIP, LIST following the active index order

## SET Options

- [ ] `SET EXACT ON/OFF` exact string comparison mode
- [ ] `SET DELETED ON/OFF` hiding records marked for deletion
- [ ] `SET ECHO ON/OFF` echoing script commands during execution
- [ ] `SET CARRY ON/OFF` carrying previous record values into APPEND
- [ ] `SET CONFIRM ON/OFF` requiring field entry confirmation
- [ ] `SET COLON ON/OFF` field delimiters in full-screen forms
- [ ] `SET HEADING ON/OFF` column headers on LIST/DISPLAY
- [ ] `SET ESCAPE ON/OFF` ESC interrupting command execution
- [ ] `SET CONSOLE ON/OFF` suppressing screen output
- [ ] `SET ALTERNATE TO <file>` and `SET ALTERNATE ON/OFF` output capture
- [ ] `SET SCREEN ON/OFF` full-screen mode toggle reconciled with AUTO/DEFAULT extension
- [ ] `SET STEP ON/OFF` single-step script execution
- [ ] No-op stubs with warning for printer/CP-M options (PRINT, EJECT, MARGIN, RAW, DEBUG, LINKAGE, DATE TO, COLOR TO, FORMAT TO, F<n>)

## Expression Engine Compatibility

- [ ] Macro substitution `&<memvar>` in command lines and expressions
- [ ] `CHR(<n>)` ASCII character function
- [ ] `RANK(<str>)` ASCII code function
- [ ] `DATE()` system date function
- [ ] `FILE(<name>)` file existence function
- [ ] `INT(<n>)` integer truncation function
- [ ] `TYPE(<expr>)` and `TEST(<expr>)` expression inspection functions
- [ ] `@(<substr>, <str>)` AT position function
- [ ] Shorthand forms: `#` (record number), `!` (upper), `$(s,start,len)` (substring), `*` (deleted flag)
- [ ] `<c1> $ <c2>` substring containment operator

## Reporting & Printer

- [ ] `REPORT [FORM <frm>] [<scope>] [TO PRINT]` report generator with .frm files
- [ ] `EJECT` and `TO PRINT` output routing
- [ ] `MODIFY COMMAND` simple PRG text editor

## Testing & Quality

- [ ] Binary-level compatibility checks against real dBase II .dbf/.ndx/.mem fixture files
- [ ] End-to-end regression tests running complete PRG programs
