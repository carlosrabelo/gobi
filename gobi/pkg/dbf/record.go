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

func (tbl *Table) ReadRecordAt(r io.ReadSeeker, recNo int) (*Record, error) {
	off, err := tbl.RecordOffset(recNo)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(off, io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seeking to record %d: %w", recNo, err)
	}
	return tbl.ReadRecord(r)
}

func (tbl *Table) WriteRecordAt(w io.WriteSeeker, recNo int, rec *Record) error {
	off, err := tbl.RecordOffset(recNo)
	if err != nil {
		return err
	}
	if len(rec.Data) != int(tbl.Header.RecordLen) {
		return fmt.Errorf("dbf: record length %d does not match table record length %d", len(rec.Data), tbl.Header.RecordLen)
	}
	if _, err := w.Seek(off, io.SeekStart); err != nil {
		return fmt.Errorf("dbf: seeking to record %d: %w", recNo, err)
	}
	_, err = w.Write(rec.Data)
	return err
}
