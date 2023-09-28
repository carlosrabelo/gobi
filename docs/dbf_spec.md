# dBase II DBF File Format Specification

Gobi uses the classic dBase II database file format (`.dbf`). This format differs significantly from dBase III/IV, notably in the size of the header structures and limits.

## Header Structure

The header contains metadata about the structure of the database and is terminated by a Carriage Return (`0x0D`).

| Offset (Bytes) | Size (Bytes) | Type | Purpose |
|---|---|---|---|
| 0 | 1 | Byte | Signature / File Type: `0x02` (normal DBF), `0x82` (with Memo `.dbt` file) |
| 1 | 2 | uint16 | Record count (little-endian, max 65,535 records) |
| 3 | 3 | Bytes | Last update date: Year (YY), Month (MM), Day (DD) relative to 1900 |
| 6 | 2 | uint16 | Length of each record in bytes (little-endian, max 1,000 bytes) |
| 8 | Variable | - | Array of Field Descriptors (each is 16 bytes) |
| Variable | 1 | Byte | Header Terminator: `0x0D` |

## Field Descriptor Structure

Each field is defined by a 16-byte block inside the header. The maximum number of fields in dBase II is **32**.

| Offset (Bytes) | Size (Bytes) | Type | Purpose |
|---|---|---|---|
| 0 | 10 | ASCII | Field Name (padded with null bytes `0x00`) |
| 10 | 1 | ASCII | Field Type: `'C'`, `'N'`, or `'L'` |
| 11 | 1 | Byte | Field Length (max 254 bytes) |
| 12 | 2 | uint16 | Field RAM Address (used by retro interpreter, set to `0x0000` or parsed/ignored) |
| 14 | 1 | Byte | Decimal Count (for numeric fields only) |
| 15 | 1 | Byte | Unused / Reserved (set to `0x00`) |

## Data Records

Records immediately follow the header terminator `0x0D`.

- All fields are stored physically in plain text ASCII representation.
- Every record starts with a **Deletion Flag** (1 byte):
  - `' '` (`0x20`): Active record.
  - `'*'` (`0x2A`): Deleted record (logical deletion).
- Field values are packed sequentially according to their designated `Field Length`. No separators exist between fields.
- The End-of-File (EOF) marker is designated by Ctrl+Z (`0x1A`) at the end of the file, although physical file truncation should also be honored.

### Field Type Specifications

1. **Character (`C`)**: Text padded with space characters (`0x20`) on the right.
2. **Numeric (`N`)**: ASCII numbers, right-aligned, padded with spaces (`0x20`) on the left. Decimals are represented with a literal dot `.`.
3. **Logical (`L`)**: Single byte representation. 
   - True: `'T'`, `'t'`, `'Y'`, `'y'`
   - False: `'F'`, `'f'`, `'N'`, `'n'`
   - Empty/Uninitialized: `'?'`
