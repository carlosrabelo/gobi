package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	// MemoBlockSize is the dBase III Plus memo block length.
	MemoBlockSize = 512
	memoEnd0      = 0x1A
	memoEnd1      = 0x1A
)

// MemoFile is an open .dbt memo store.
type MemoFile struct {
	rw       io.ReadWriteSeeker
	NextFree uint32
}

// CreateMemo writes block 0 with next-free = 1.
func CreateMemo(rw io.ReadWriteSeeker) (*MemoFile, error) {
	m := &MemoFile{rw: rw, NextFree: 1}
	if err := m.writeHeader(); err != nil {
		return nil, err
	}
	return m, nil
}

// OpenMemo reads block 0.
func OpenMemo(rw io.ReadWriteSeeker) (*MemoFile, error) {
	if _, err := rw.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var hdr [MemoBlockSize]byte
	if _, err := io.ReadFull(rw, hdr[:]); err != nil {
		return nil, fmt.Errorf("short memo header: %w", err)
	}
	next := binary.LittleEndian.Uint32(hdr[0:4])
	if next == 0 {
		next = 1
	}
	return &MemoFile{rw: rw, NextFree: next}, nil
}

func (m *MemoFile) writeHeader() error {
	var hdr [MemoBlockSize]byte
	binary.LittleEndian.PutUint32(hdr[0:4], m.NextFree)
	if _, err := m.rw.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := m.rw.Write(hdr[:])
	return err
}

// ReadText reads memo text starting at block. Block 0 or empty digits → "".
func (m *MemoFile) ReadText(block uint32) (string, error) {
	if m == nil || block == 0 {
		return "", nil
	}
	var b strings.Builder
	pendingEnd := false
	for {
		off := int64(block) * MemoBlockSize
		size, err := m.size()
		if err != nil {
			return "", err
		}
		if off >= size {
			return "", fmt.Errorf("memo block %d past end of file", block)
		}
		if _, err := m.rw.Seek(off, io.SeekStart); err != nil {
			return "", err
		}
		var buf [MemoBlockSize]byte
		n, err := io.ReadFull(m.rw, buf[:])
		if err != nil && err != io.ErrUnexpectedEOF {
			return "", fmt.Errorf("memo block %d past end of file", block)
		}
		chunk := buf[:n]
		for i := 0; i < len(chunk); i++ {
			if pendingEnd {
				if chunk[i] == memoEnd1 {
					return b.String(), nil
				}
				b.WriteByte(memoEnd0)
				pendingEnd = false
			}
			if chunk[i] == memoEnd0 {
				pendingEnd = true
				continue
			}
			b.WriteByte(chunk[i])
		}
		block++
	}
}

func (m *MemoFile) size() (int64, error) {
	cur, err := m.rw.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	end, err := m.rw.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	_, err = m.rw.Seek(cur, io.SeekStart)
	return end, err
}

// AllocWrite stores text in newly allocated blocks and returns the starting block.
func (m *MemoFile) AllocWrite(text string) (uint32, error) {
	if m == nil {
		return 0, fmt.Errorf("no memo file")
	}
	payload := append([]byte(text), memoEnd0, memoEnd1)
	start := m.NextFree
	block := start
	for len(payload) > 0 {
		var buf [MemoBlockSize]byte
		n := copy(buf[:], payload)
		payload = payload[n:]
		off := int64(block) * MemoBlockSize
		if _, err := m.rw.Seek(off, io.SeekStart); err != nil {
			return 0, err
		}
		if _, err := m.rw.Write(buf[:]); err != nil {
			return 0, err
		}
		block++
	}
	m.NextFree = block
	if err := m.writeHeader(); err != nil {
		return 0, err
	}
	return start, nil
}

// ParseBlockDigits parses the 10-byte memo field (leading spaces). Empty → 0.
func ParseBlockDigits(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("bad memo block %q", s)
	}
	return uint32(n), nil
}

// FormatBlockDigits formats a block number into 10 digits with leading spaces.
func FormatBlockDigits(block uint32) string {
	if block == 0 {
		return strings.Repeat(" ", 10)
	}
	s := strconv.FormatUint(uint64(block), 10)
	if len(s) > 10 {
		s = s[len(s)-10:]
	}
	return strings.Repeat(" ", 10-len(s)) + s
}
