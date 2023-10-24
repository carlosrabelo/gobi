package ndx

import (
	"io"
)

// memFile is an in-memory io.ReadWriteSeeker for tests.
type memFile struct {
	buf []byte
	pos int64
}

func (m *memFile) Read(p []byte) (int, error) {
	if m.pos >= int64(len(m.buf)) {
		return 0, io.EOF
	}
	n := copy(p, m.buf[m.pos:])
	m.pos += int64(n)
	return n, nil
}

func (m *memFile) Write(p []byte) (int, error) {
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

func (m *memFile) Seek(offset int64, whence int) (int64, error) {
	var n int64
	switch whence {
	case io.SeekStart:
		n = offset
	case io.SeekCurrent:
		n = m.pos + offset
	case io.SeekEnd:
		n = int64(len(m.buf)) + offset
	default:
		return 0, io.ErrUnexpectedEOF
	}
	if n < 0 {
		return 0, io.ErrUnexpectedEOF
	}
	m.pos = n
	return n, nil
}
