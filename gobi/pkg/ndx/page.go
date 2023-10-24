package ndx

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Page is one 512-byte index page (not page 0).
type Page struct {
	Num     uint32
	Entries []Entry
	// Right is the child for keys greater than the last entry (0 if none).
	Right uint32
}

func (idx *Index) readPage(num uint32) (Page, error) {
	if num == 0 || num >= idx.hdr.PageCount {
		return Page{}, fmt.Errorf("invalid page %d", num)
	}
	buf := make([]byte, PageSize)
	if _, err := idx.rw.Seek(int64(num)*PageSize, io.SeekStart); err != nil {
		return Page{}, err
	}
	if _, err := io.ReadFull(idx.rw, buf); err != nil {
		return Page{}, err
	}
	count := binary.LittleEndian.Uint32(buf[0:4])
	if count > uint32(idx.hdr.MaxKeys) {
		return Page{}, fmt.Errorf("page count %d above capacity %d", count, idx.hdr.MaxKeys)
	}
	es := int(idx.hdr.EntrySize)
	p := Page{Num: num, Entries: make([]Entry, 0, count)}
	off := 4
	for i := uint32(0); i < count; i++ {
		e, err := DecodeEntry(buf[off:off+es], int(idx.hdr.KeyLen))
		if err != nil {
			return Page{}, err
		}
		p.Entries = append(p.Entries, e)
		off += es
	}
	// Rightmost child lives in the next entry slot's Child field when unused,
	// or after the last entry at Child position when page is full (bytes at off).
	if off+4 <= PageSize {
		p.Right = binary.LittleEndian.Uint32(buf[off : off+4])
	}
	return p, nil
}

func (idx *Index) writePage(p Page) error {
	if len(p.Entries) > int(idx.hdr.MaxKeys) {
		return fmt.Errorf("page over capacity")
	}
	buf := make([]byte, PageSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(p.Entries)))
	es := int(idx.hdr.EntrySize)
	off := 4
	for _, e := range p.Entries {
		if err := EncodeEntry(buf[off:off+es], e, int(idx.hdr.KeyLen)); err != nil {
			return err
		}
		off += es
	}
	if off+4 <= PageSize {
		binary.LittleEndian.PutUint32(buf[off:off+4], p.Right)
	}
	if _, err := idx.rw.Seek(int64(p.Num)*PageSize, io.SeekStart); err != nil {
		return err
	}
	_, err := idx.rw.Write(buf)
	return err
}

func (idx *Index) allocPage() (uint32, error) {
	num := idx.hdr.PageCount
	idx.hdr.PageCount++
	// Extend file with a zero page.
	buf := make([]byte, PageSize)
	if _, err := idx.rw.Seek(int64(num)*PageSize, io.SeekStart); err != nil {
		return 0, err
	}
	if _, err := idx.rw.Write(buf); err != nil {
		return 0, err
	}
	if err := idx.writeHeader(); err != nil {
		return 0, err
	}
	return num, nil
}
