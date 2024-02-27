# dBase II Language & Syntax Specification

Gobi implements the subset of the dBase II programming language. All commands are case-insensitive and can be run interactively or inside `.prg` script files.

## Command Syntaxes

### Database Operations
- `CREATE <filename>`: Prompts interatively for structure definitions.
- `USE [<filename>] [INDEX <index1>, <index2>, ...]`: Opens a table and optionally binds indexes. If no arguments are passed, closes the active database.
- `SELECT <PRIMARY/SECONDARY>`: Switches work area. dBase II supports two concurrent work areas (Primary and Secondary).
- `CLOSE [DATABASES / INDEX / ALL]`: Closes files in current or all work areas.
- `CLEAR`: Closes all databases and releases all memory variables (dBase II semantics).
- `DELETE FILE <filename>`: Removes a file from disk.
- `DISPLAY STRUCTURE` / `LIST STRUCTURE`: Prints schema details.
- `DISPLAY FILES [LIKE <pattern>]` / `LIST FILES [LIKE <pattern>]`: Directory listing; without a pattern shows database files with record counts. `DIR` is an alias.
- `DISPLAY STATUS` / `LIST STATUS`: Shows the selected work area, open databases and indexes per area, and current SET values.

### Record Navigation
- `GO TO <n>` / `<n>`: Moves cursor to record index `n` (1-indexed).
- `GO TOP`: Moves to first record (or first active/indexed key).
- `GO BOTTOM`: Moves to last record.
- `SKIP [<n>]`: Moves current record cursor relatively by `n` (default is 1).

### Data Manipulation
- `LIST [ALL / NEXT <n>] [<fields>] [FOR <expr>] [WHILE <expr>]`: Print matching records.
- `DISPLAY [ALL / NEXT <n>] [<fields>] [FOR <expr>] [WHILE <expr>]`: Paged display of matching records.
- `APPEND`: In simple mode, prompts field by field. In full screen mode, starts form-based data entry.
- `APPEND BLANK`: Adds an empty record without prompting and makes it the current record.
- `INSERT [BEFORE] [BLANK]`: Insert a record after the current one (or before it with BEFORE), shifting the following records; BLANK skips the field prompts.
- `CHANGE [ALL / NEXT <n>] FIELD <list> [FOR <expr>]`: Line-mode field editor; shows each listed field and prompts CHANGE? for the substring to replace and TO? for the replacement.
- `REPLACE [ALL / NEXT <n>] <field> WITH <expr> [FOR <expr>] [WHILE <expr>]`: Update records matching the criteria; without scope or clauses only the current record changes.
- `DELETE [ALL / NEXT <n>] [FOR <expr>]`: Mark records with `*` (logical deletion).
- `RECALL [ALL / NEXT <n>] [FOR <expr>]`: Unmark logically deleted records.
- `PACK`: Re-writes `.dbf` removing all logically deleted records physically.
- `ZAP`: Instantly truncates the active database.
- `COUNT [ALL / NEXT <n>] [FOR <expr>] [WHILE <expr>] [TO <var>]`: Count matching records.
- `SUM [ALL / NEXT <n>] <expr list> [FOR <expr>] [WHILE <expr>] [TO <var list>]`: Total numeric expressions over matching records.
- `LOCATE [ALL / NEXT <n>] FOR <expr> [WHILE <expr>]`: Find the first matching record; resume with `CONTINUE`.

### Variables & Memory
- `STORE <expr> TO <var>` / `<var> = <expr>`: Set variable in memory.
- `DISPLAY MEMORY` / `LIST MEMORY`: Print current variables.
- `SAVE TO <file>.mem`: Serialize active variables to disk.
- `RESTORE FROM <file>.mem`: Restore variables from disk.
- `RELEASE [ALL / <var>]`: Clean memory variables.

### Console Input
- `ACCEPT ['<prompt>'] TO <var>`: Prompts the user and stores the typed line as a character memory variable.
- `INPUT ['<prompt>'] TO <var>`: Prompts the user, evaluates the typed line as an expression, and stores the resulting value.
- `WAIT [TO <var>]`: Displays WAITING and pauses until a single key is pressed, optionally storing the key as a character variable.

### Program Flow Control
- `DO <filename.prg>`: Executes a program script.
- `IF <expr>` ... `[ELSE]` ... `ENDIF`: Conditional branching.
- `DO WHILE <expr>` ... `ENDDO`: Loop. Supports `LOOP` to restart cycle and `EXIT` to break.
- `DO CASE` / `CASE <expr>` / `OTHERWISE` / `ENDCASE`: Multi-branch selection; runs the first true CASE, or OTHERWISE when none matches.
- `TEXT` ... `ENDTEXT`: Outputs the enclosed lines verbatim from a command file.
- `REMARK <text>`: Echoes its text to the output. Use `*` for silent comment lines.
- `NOTE <text>`: Silent comment line, alias of `*`.
- `RETURN`: Returns from script to caller.
- `CANCEL`: Aborts execution of all scripts and returns to REPL.

### Environment Settings
- `SET TALK [ON / OFF]`: Echo command execution results and record counts.
- `SET INTENSITY [ON / OFF]`: Reverse-video highlighting on TUI input fields.
- `SET BELL [ON / OFF]`: Bell sound on validation errors.
- `SET EXACT [ON / OFF]`: With OFF (default), string comparisons stop at the end of the right operand, so `'Smith' = 'Sm'` is true; with ON the whole strings must match.
- `SET DELETED [ON / OFF]`: With ON, records marked for deletion are hidden from LIST, DISPLAY, COUNT, LOCATE, and record navigation (GO TOP/BOTTOM, SKIP).
- `SET DEFAULT TO <path>`: Default directory for databases and scripts.
- `SET INDEX TO [<ndx1>, <ndx2>, ...]`: Rebinds index files on the active table; the first becomes the controlling index, driving GO TOP/BOTTOM, SKIP, and LIST order. Without a list, closes all bound indexes.
- `SET SCREEN [AUTO / DEFAULT]`: Adapt screen geometry to the real terminal size (AUTO) or pin the classic dBase II 80x24 (DEFAULT). Gobi extension.

### Full-Screen Interactive (TUI)
- `ERASE`: Clears screen and homes cursor (dBase II semantics).
- `@ <row>, <col> SAY <expr>`: Print value at specific coordinates.
- `@ <row>, <col> GET <var/field>`: Register input buffer at coordinates.
- `CLEAR GETS`: Releases pending @ GET registrations without touching the screen.
- `READ`: Enter full-screen interactive form edit mode for all registered `GET` fields.
- `EDIT <n>`: Edit record `n` in interactive form.
- `BROWSE`: Opens a full-screen interactive grid sheet for the active database.

---

## Built-In Functions

- `EOF()`: Returns logical `.T.` if cursor is past last record.
- `BOF()`: Returns logical `.T.` if cursor is before first record.
- `DELETED()`: Returns logical `.T.` if current record is marked deleted.
- `RECNO()`: Returns active record integer index.
- `FOUND()`: Returns logical `.T.` if last `FIND` or `SEEK` was successful.
- `TRIM(<str>)`: Strips trailing whitespace.
- `UPPER(<str>)` / `LOWER(<str>)`: Case conversion.
- `LEN(<str>)`: String length.
- `SUBSTR(<str>, <start>, <len>)`: Substring extraction (1-indexed).
- `VAL(<str>)`: Convert string to numeric.
- `CHR(<n>)`: Returns the ASCII character for code `n` (0-255).
- `STR(<val>, <width>, <decimals>)`: Convert numeric to formatted string.
