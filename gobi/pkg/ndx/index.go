package ndx

import (
	"fmt"
	"io"
)

// Index is an open NDX on a read-writer.
type Index struct {
	rw  io.ReadWriteSeeker
	hdr Header
}

// Create writes a new empty index (root 0) with the given key attributes.
func Create(rw io.ReadWriteSeeker, keyLen int, typ KeyType, unique bool, expr string) (*Index, error) {
	h := Header{
		Root:      0,
		PageCount: 1,
		KeyLen:    uint16(keyLen),
		Type:      typ,
		Unique:    unique,
		Expr:      expr,
	}
	idx := &Index{rw: rw, hdr: h}
	if err := idx.writeHeader(); err != nil {
		return nil, err
	}
	return idx, nil
}

// Open reads page 0 and returns an Index.
func Open(rw io.ReadWriteSeeker) (*Index, error) {
	if _, err := rw.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	h, err := ReadHeader(rw)
	if err != nil {
		return nil, err
	}
	return &Index{rw: rw, hdr: h}, nil
}

// Header returns a copy of page 0.
func (idx *Index) Header() Header { return idx.hdr }

func (idx *Index) writeHeader() error {
	buf, err := EncodeHeader(idx.hdr)
	if err != nil {
		return err
	}
	// EncodeHeader fills EntrySize/MaxKeys — keep hdr in sync.
	decoded, err := DecodeHeader(buf)
	if err != nil {
		return err
	}
	idx.hdr = decoded
	if _, err := idx.rw.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = idx.rw.Write(buf)
	return err
}

// Search finds an exact key and returns its record number.
func (idx *Index) Search(key []byte) (recNo uint32, ok bool, err error) {
	if err := idx.checkKey(key); err != nil {
		return 0, false, err
	}
	if idx.hdr.Root == 0 {
		return 0, false, nil
	}
	pageNum := idx.hdr.Root
	for pageNum != 0 {
		p, err := idx.readPage(pageNum)
		if err != nil {
			return 0, false, err
		}
		i, exact := idx.findPos(p, key)
		if exact {
			return p.Entries[i].RecNo, true, nil
		}
		var child uint32
		if i < len(p.Entries) {
			child = p.Entries[i].Child
		} else {
			child = p.Right
		}
		if child == 0 {
			return 0, false, nil
		}
		pageNum = child
	}
	return 0, false, nil
}

func (idx *Index) checkKey(key []byte) error {
	if len(key) != int(idx.hdr.KeyLen) {
		return fmt.Errorf("key length %d, want %d", len(key), idx.hdr.KeyLen)
	}
	return nil
}

func (idx *Index) findPos(p Page, key []byte) (int, bool) {
	for i, e := range p.Entries {
		c := CompareKeys(key, e.Key, idx.hdr.Type)
		if c == 0 {
			return i, true
		}
		if c < 0 {
			return i, false
		}
	}
	return len(p.Entries), false
}

// Insert adds key/recNo. Unique indexes ignore duplicate keys.
func (idx *Index) Insert(key []byte, recNo uint32) error {
	if err := idx.checkKey(key); err != nil {
		return err
	}
	if idx.hdr.Root == 0 {
		num, err := idx.allocPage()
		if err != nil {
			return err
		}
		p := Page{
			Num:     num,
			Entries: []Entry{{Child: 0, RecNo: recNo, Key: append([]byte(nil), key...)}},
		}
		if err := idx.writePage(p); err != nil {
			return err
		}
		idx.hdr.Root = num
		return idx.writeHeader()
	}
	split, err := idx.insert(idx.hdr.Root, key, recNo)
	if err != nil {
		return err
	}
	if split == nil {
		return nil
	}
	num, err := idx.allocPage()
	if err != nil {
		return err
	}
	root := Page{
		Num: num,
		Entries: []Entry{{
			Child: idx.hdr.Root,
			RecNo: split.RecNo,
			Key:   append([]byte(nil), split.Key...),
		}},
		Right: split.Right,
	}
	if err := idx.writePage(root); err != nil {
		return err
	}
	idx.hdr.Root = num
	return idx.writeHeader()
}

type promote struct {
	Key   []byte
	RecNo uint32
	Right uint32 // new right page
}

func (idx *Index) insert(pageNum uint32, key []byte, recNo uint32) (*promote, error) {
	p, err := idx.readPage(pageNum)
	if err != nil {
		return nil, err
	}
	i, exact := idx.findPos(p, key)
	if exact && idx.hdr.Unique {
		return nil, nil
	}
	var child uint32
	if i < len(p.Entries) {
		child = p.Entries[i].Child
	} else {
		child = p.Right
	}
	if child != 0 {
		pr, err := idx.insert(child, key, recNo)
		if err != nil {
			return nil, err
		}
		if pr == nil {
			return nil, nil
		}
		return idx.insertSep(p, i, pr)
	}
	// Leaf insert.
	e := Entry{Child: 0, RecNo: recNo, Key: append([]byte(nil), key...)}
	p.Entries = append(p.Entries, Entry{})
	copy(p.Entries[i+1:], p.Entries[i:])
	p.Entries[i] = e
	if len(p.Entries) <= int(idx.hdr.MaxKeys) {
		return nil, idx.writePage(p)
	}
	return idx.splitPage(p)
}

func (idx *Index) insertSep(p Page, i int, pr *promote) (*promote, error) {
	e := Entry{
		Child: 0, // set below
		RecNo: pr.RecNo,
		Key:   append([]byte(nil), pr.Key...),
	}
	if i < len(p.Entries) {
		e.Child = p.Entries[i].Child // left page (already split in place)
	} else {
		e.Child = p.Right
	}
	p.Entries = append(p.Entries, Entry{})
	copy(p.Entries[i+1:], p.Entries[i:])
	p.Entries[i] = e
	if i+1 < len(p.Entries) {
		p.Entries[i+1].Child = pr.Right
	} else {
		p.Right = pr.Right
	}
	if len(p.Entries) <= int(idx.hdr.MaxKeys) {
		return nil, idx.writePage(p)
	}
	return idx.splitPage(p)
}

func (idx *Index) splitPage(p Page) (*promote, error) {
	mid := len(p.Entries) / 2
	median := p.Entries[mid]
	rightNum, err := idx.allocPage()
	if err != nil {
		return nil, err
	}
	right := Page{
		Num:     rightNum,
		Entries: append([]Entry(nil), p.Entries[mid+1:]...),
		Right:   p.Right,
	}
	left := Page{
		Num:     p.Num,
		Entries: append([]Entry(nil), p.Entries[:mid]...),
		Right:   median.Child,
	}
	if err := idx.writePage(left); err != nil {
		return nil, err
	}
	if err := idx.writePage(right); err != nil {
		return nil, err
	}
	return &promote{
		Key:   append([]byte(nil), median.Key...),
		RecNo: median.RecNo,
		Right: rightNum,
	}, nil
}

// Delete removes the entry for key and recNo.
func (idx *Index) Delete(key []byte, recNo uint32) error {
	if err := idx.checkKey(key); err != nil {
		return err
	}
	if idx.hdr.Root == 0 {
		return nil
	}
	empty, err := idx.delete(idx.hdr.Root, key, recNo)
	if err != nil {
		return err
	}
	if empty {
		idx.hdr.Root = 0
		return idx.writeHeader()
	}
	return nil
}

func (idx *Index) delete(pageNum uint32, key []byte, recNo uint32) (bool, error) {
	p, err := idx.readPage(pageNum)
	if err != nil {
		return false, err
	}
	i, exact := idx.findPos(p, key)
	if exact && p.Entries[i].RecNo == recNo {
		left := p.Entries[i].Child
		p.Entries = append(p.Entries[:i], p.Entries[i+1:]...)
		if left != 0 && i < len(p.Entries) {
			// Prefer keeping structure simple: drop left subtree pointer into next.
			if p.Entries[i].Child == 0 {
				p.Entries[i].Child = left
			}
		} else if left != 0 && i >= len(p.Entries) {
			if p.Right == 0 {
				p.Right = left
			}
		}
		if len(p.Entries) == 0 && p.Right == 0 {
			return true, idx.writePage(p)
		}
		return false, idx.writePage(p)
	}
	var child uint32
	if i < len(p.Entries) {
		child = p.Entries[i].Child
	} else {
		child = p.Right
	}
	if child == 0 {
		return false, nil
	}
	subEmpty, err := idx.delete(child, key, recNo)
	if err != nil {
		return false, err
	}
	if !subEmpty {
		return false, nil
	}
	if i < len(p.Entries) {
		p.Entries[i].Child = 0
	} else {
		p.Right = 0
	}
	return len(p.Entries) == 0 && p.Right == 0, idx.writePage(p)
}
