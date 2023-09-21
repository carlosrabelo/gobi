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
- [ ] Command history log file and keyboard scrolling
- [ ] Command parser splitting verb and string arguments
- [ ] Parser support for conditional FOR clause
- [ ] Parser support for conditional WHILE clause
- [ ] Parser support for TO output redirect path
- [ ] Command multiplexer matching verb to execution handler

## DBF File Layout

References: [docs/dbf_spec.md](file:///home/carlos/Sources/gobi/docs/dbf_spec.md)

- [ ] Read header signature (0x02 or 0x82)
- [ ] Read record count (2 bytes) and record length (2 bytes)
- [ ] Read field count and iterate 16-byte field descriptors
- [ ] Map field descriptors into memory structs
- [ ] Header terminator validation (0x0D)
- [ ] Raw record reader with deletion flag check (0x20 vs 0x2A)
- [ ] Field decoder for Character (C) - space-stripped strings
- [ ] Field decoder for Numeric (N) - parsing ASCII into float64
- [ ] Field decoder for Logical (L) - parsing T/F/Y/N to bool
- [ ] Raw record writer with field padding
- [ ] End-Of-File (EOF) marker logic (0x1A)
- [ ] In-place record update by record number offset
- [ ] Append new record helper updating header count
- [ ] Safe file flush and sync on close

## Database Commands

- [ ] `USE` command opening target DBF file
- [ ] `SELECT` switching between Primary and Secondary areas
- [ ] `CLOSE DATABASES` and `CLOSE INDEX`
- [ ] `DISPLAY STRUCTURE` and `LIST STRUCTURE`
- [ ] `GOTO` absolute record position
- [ ] `GO TOP` and `GO BOTTOM`
- [ ] `SKIP` moving cursor relative to active database size
- [ ] `LIST` showing fields with FOR/WHILE filtering
- [ ] `DISPLAY` paginating records list to stdout
- [ ] `APPEND` line prompt adding records interactively
- [ ] `REPLACE` field value evaluations
- [ ] `DELETE` marking active record for deletion
- [ ] `RECALL` restoring deleted record marker
- [ ] `PACK` rewriting DBF to drop deleted records
- [ ] `ZAP` truncating active DBF data area
- [ ] `CREATE` interactively defining schema and creating new DBF
- [ ] `EDIT` interactive full-screen form record editor
- [ ] `MODIFY STRUCTURE` changing schema of active DBF
- [ ] `COPY TO` exporting records/schema to a new DBF
- [ ] `APPEND FROM` importing records from another DBF or text file
- [ ] `UPDATE FROM` modifying active DBF records using secondary area data
- [ ] `JOIN` combining active and secondary DBFs into a new table
- [ ] `TOTAL ON` summarizing numeric fields to a target DBF
- [ ] `LOCATE` and `CONTINUE` sequential search matching condition
- [ ] `COUNT` records matching conditional filter
- [ ] `SUM` totaling numeric field values matching filter
- [ ] `AVERAGE` calculating mean of numeric fields matching filter
- [ ] `?` evaluate and display expression value with newline
- [ ] `??` evaluate and display expression value without newline

## Memory Variables

- [ ] Global symbol registry mapping names to variable values
- [ ] `STORE` assignment parser and executor
- [ ] `DISPLAY MEMORY` and `LIST MEMORY` tabular printer
- [ ] `SAVE TO` binary serializer saving variables to .mem file
- [ ] `RESTORE FROM` deserializer loading variables
- [ ] `RELEASE` cleaning symbols from registry

## Programming & Scripting

- [ ] `DO` command loader checking file existence
- [ ] PRG line-by-line script reader and parser
- [ ] Script instruction pointer controller
- [ ] `IF ... ELSE ... ENDIF` parser and branch executor
- [ ] `DO WHILE ... ENDDO` loop manager
- [ ] `LOOP` and `EXIT` jump resolution
- [ ] Nesting call stack for parent/child scripts execution
- [ ] `RETURN` pops caller state from call stack
- [ ] `CANCEL` clears script execution stack and halts to REPL

## Indexing (.ndx)

References: [docs/ndx_spec.md](file:///home/carlos/Sources/gobi/docs/ndx_spec.md)

- [ ] NDX page 0 header structure serialization
- [ ] B-Tree in-memory representations for nodes and entries
- [ ] Disk page manager allocating 512-byte blocks
- [ ] Internal node split on page overflow
- [ ] Leaf node entry insertion and record mapping
- [ ] Exact key search traversing tree pages
- [ ] Prefix key search mapping to first matches
- [ ] Leaf node entry deletion on key removals
- [ ] `INDEX ON` command scanning DBF and constructing tree
- [ ] Multi-index synchronization during table `APPEND`
- [ ] Multi-index synchronization during table `REPLACE`
- [ ] `REINDEX` command re-building active trees
- [ ] `FIND` and `SEEK` commands setting DBF cursor
- [ ] `SORT ON` sorting DBF records physically

## Screen and TUI (VT100)

- [ ] ANSI escape wrapper for cursor positioning and colors
- [ ] Terminal raw-mode keyboard capture adapter
- [ ] Double-buffered terminal frame writer
- [ ] `CLEAR` command clearing screen buffer
- [ ] `@ SAY` printing expression string at coordinates
- [ ] `@ GET` registering interactive input fields
- [ ] `READ` loop handling user editing, Tab, and field validation
- [ ] Interactive `APPEND` screen generation
- [ ] Interactive `EDIT` record screen generation
- [ ] `BROWSE` table view spreadsheet matrix
- [ ] `BROWSE` cell cursor movement using arrow keys
- [ ] `BROWSE` inline cell edit and record deletion

## System Environment & CLI

- [ ] `SET TALK` flag controller
- [ ] `SET INTENSITY` video invert controller
- [ ] `SET BELL` sound alarm controller
- [ ] `SET DEFAULT` data directory mapping
- [ ] `DIR` command listing DBFs with sizes and records count
- [ ] `RENAME` and `ERASE` file hooks
- [ ] Entry point arguments (`-e` inline command, file run)
- [ ] `HELP` command documentation parser

## Second milestone

## Console & Program Input

- [ ] `ACCEPT ['prompt'] TO <var>` string input into memory variable
- [ ] `INPUT ['prompt'] TO <var>` evaluated expression input
- [ ] `WAIT [TO <var>]` single-key pause storing the pressed key
- [ ] `TEXT ... ENDTEXT` literal text block output in scripts
- [ ] `REMARK` echoing its text to output (currently treated as silent comment)
- [ ] `NOTE` silent comment lines in scripts (alias of `*`)

## Flow Control

- [ ] `DO CASE / CASE <expr> / OTHERWISE / ENDCASE` branch structure

## Record Operations & Scopes

- [ ] Scope clauses `ALL` and `NEXT <n>` on record commands (LIST, DISPLAY, DELETE, RECALL, REPLACE, COUNT, SUM, LOCATE)
- [ ] `APPEND BLANK` adding an empty record without prompting
- [ ] `INSERT [BEFORE] [BLANK]` inserting a record at the cursor position
- [ ] `CHANGE [<scope>] FIELD <list> [FOR <expr>]` line-mode field editor

## Command Semantics Alignment

- [ ] `ERASE` clearing the screen (dBase II semantics)
- [ ] `DELETE FILE <filename>` removing files (replaces current ERASE behavior)
- [ ] `CLEAR` closing databases and releasing memory variables (dBase II semantics)
- [ ] `CLEAR GETS` releasing pending @ GET registrations
- [ ] `DISPLAY FILES` / `LIST FILES [LIKE <pattern>]` directory listing (DIR as alias)
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
