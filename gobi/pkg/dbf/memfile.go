package dbf

import (
	"fmt"
	"io"
)

// MemFile is an in-memory ReadWriteSeeker for tests.
type MemFile struct {
	buf []byte
	pos int64
}

func (m *MemFile) Read(p []byte) (int, error) {
	if m.pos >= int64(len(m.buf)) {
		return 0, io.EOF
	}
	n := copy(p, m.buf[m.pos:])
	m.pos += int64(n)
	return n, nil
}

func (m *MemFile) Write(p []byte) (int, error) {
	end := int(m.pos) + len(p)
	if end > len(m.buf) {
		nb := make([]byte, end)
		copy(nb, m.buf)
		m.buf = nb
	}
	copy(m.buf[m.pos:], p)
	m.pos += int64(len(p))
	return len(p), nil
}

func (m *MemFile) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = m.pos + offset
	case io.SeekEnd:
		abs = int64(len(m.buf)) + offset
	default:
		return 0, fmt.Errorf("bad whence")
	}
	if abs < 0 {
		return 0, fmt.Errorf("negative seek")
	}
	m.pos = abs
	return abs, nil
}

// Bytes returns the full buffer.
func (m *MemFile) Bytes() []byte { return m.buf }

// Sync is a no-op for MemFile.
func (m *MemFile) Sync() error { return nil }
