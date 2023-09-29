package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	SignatureStd  byte = 0x02
	SignatureMemo byte = 0x82
)

var validSignatures = map[byte]bool{
	SignatureStd:  true,
	SignatureMemo: true,
}

type Header struct {
	Signature   byte
	RecordCount uint16
	RecordLen   uint16
}

func readHeader(r io.Reader) (*Header, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, fmt.Errorf("dbf: reading header: %w", err)
	}

	sig := buf[0]
	if !validSignatures[sig] {
		return nil, fmt.Errorf("dbf: invalid signature 0x%02X", sig)
	}

	rc := binary.LittleEndian.Uint16(buf[1:3])
	rl := binary.LittleEndian.Uint16(buf[6:8])

	return &Header{
		Signature:   sig,
		RecordCount: rc,
		RecordLen:   rl,
	}, nil
}
