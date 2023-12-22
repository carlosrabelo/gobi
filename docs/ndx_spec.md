# dBase II NDX File Format Specification

Gobi uses disk-backed B-Trees for indexing, matching the classic dBase II `.ndx` file structure. Index files are built in 512-byte pages to ensure fast random access on raw storage devices.

## Index Header Structure (Page 0)

The first page (Page 0, 512 bytes) contains the metadata for the index.

| Offset (Bytes) | Size (Bytes) | Type | Purpose |
|---|---|---|---|
| 0 | 2 | uint16 | Root page ID (0 if empty/no records) |
| 2 | 2 | uint16 | Total number of pages allocated |
| 4 | 2 | uint16 | Key length (calculated from index expression, max 100 bytes) |
| 6 | 2 | uint16 | Maximum keys per page |
| 8 | 2 | uint16 | Key type (0 = Character, 1 = Numeric) |
| 10 | 400 | ASCII | Index expression string (null-terminated, e.g., `"NOME + STR(IDADE,3,0)"`) |
| 410 | 102 | - | Unused padding (zeros `0x00`) to fill the 512-byte page |

## Index Page Node Structure

All other pages (Pages 1+) represent B-Tree nodes. Each page is 512 bytes long.

- **Leaf Nodes**: Contain keys and physical record numbers in the `.dbf` file.
- **Internal Nodes**: Contain keys and child page IDs in the `.ndx` file.

### Node Page Layout

| Offset (Bytes) | Size (Bytes) | Type | Purpose |
|---|---|---|---|
| 0 | 2 | int16 | Count of active keys in this page |
| 2 | Variable | - | Key array and pointer references |

#### Entry format for Internal Nodes:
Each entry contains:
- `ChildPageID` (2 bytes, uint16) pointing to the page of the subtree containing keys smaller or equal to the key.
- `KeyData` (Size matches `Key Length` bytes) representing the separator value.

After all entries, a final `ChildPageID` (2 bytes, uint16) points to the subtree containing keys larger than the last key.

#### Entry format for Leaf Nodes:
Each entry contains:
- `RecordNumber` (2 bytes, uint16) pointing to the record in the `.dbf` file.
- `KeyData` (Size matches `Key Length` bytes) representing the literal indexed key value.

## Insertion and Balance Rules
- Nodes must split when a new key is added that exceeds the page capacity (`Maximum keys per page`).
- When a split occurs, the median key is pushed up to the parent node.
- Merging or redistributing occurs when a node falls below 50% capacity (underflow) during deletion.
