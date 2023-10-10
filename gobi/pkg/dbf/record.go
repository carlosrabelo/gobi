package dbf

import (
	"fmt"
	"io"
)

const (
	deletionActive  byte = 0x20
	deletionDeleted byte = 0x2A
	eofMarker       byte = 0x1A
)

type Record struct {
	Deleted bool
	Data    []byte
}

func (r *Record) FieldData(tbl *Table, index int) []byte {
	if index < 0 || index >= len(tbl.Fields) {
		return nil
	}
	start := tbl.Offset[index] + 1
	end := tbl.Offset[index+1] + 1
	if end > len(r.Data) {
		return nil
	}
	return r.Data[start:end]
}

func (tbl *Table) ReadRecord(r io.Reader) (*Record, error) {
	buf := make([]byte, tbl.Header.RecordLen)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		if err == io.ErrUnexpectedEOF {
			if n > 0 && buf[0] == eofMarker {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("dbf: truncated record: %w", err)
		}
		return nil, err
	}

	if buf[0] == eofMarker {
		return nil, io.EOF
	}

	if buf[0] != deletionActive && buf[0] != deletionDeleted {
		return nil, fmt.Errorf("dbf: invalid deletion flag 0x%02X", buf[0])
	}

	return &Record{
		Deleted: buf[0] == deletionDeleted,
		Data:    buf,
	}, nil
}

func (tbl *Table) ReadAllRecords(r io.Reader) ([]*Record, error) {
	var records []*Record
	for {
		rec, err := tbl.ReadRecord(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return records, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func WriteEOF(w io.Writer) error {
	_, err := w.Write([]byte{eofMarker})
	return err
}
