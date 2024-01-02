package ndx

import (
	"fmt"
	"io"
)

// PageManager manages fixed 512-byte pages in an NDX file on disk.
type PageManager struct {
	file   io.ReadWriteSeeker
	header *Header
}

// PageOffset returns the byte offset of pageID in an NDX file.
func PageOffset(pageID uint16) int64 {
	return int64(pageID) * PageSize
}

// OpenPageManager opens an existing NDX file and reads page 0 metadata.
func OpenPageManager(file io.ReadWriteSeeker) (*PageManager, error) {
	if file == nil {
		return nil, fmt.Errorf("ndx: nil file")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("ndx: seeking to header: %w", err)
	}
	header, err := ReadHeader(file)
	if err != nil {
		return nil, err
	}
	return &PageManager{file: file, header: header}, nil
}

// CreatePageManager initializes a new NDX file with header page 0.
func CreatePageManager(file io.ReadWriteSeeker, h *Header) (*PageManager, error) {
	if file == nil {
		return nil, fmt.Errorf("ndx: nil file")
	}
	if h == nil {
		return nil, fmt.Errorf("ndx: nil header")
	}
	if err := validateHeader(h); err != nil {
		return nil, err
	}

	header := *h
	if header.PageCount == 0 {
		header.PageCount = 1
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("ndx: seeking to header: %w", err)
	}
	if err := WriteHeader(file, &header); err != nil {
		return nil, err
	}
	return &PageManager{file: file, header: &header}, nil
}

// Header returns the in-memory page 0 metadata.
func (pm *PageManager) Header() *Header {
	return pm.header
}

// SyncHeader writes the current header to page 0.
func (pm *PageManager) SyncHeader() error {
	if _, err := pm.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("ndx: seeking to header: %w", err)
	}
	return WriteHeader(pm.file, pm.header)
}

// ReadPage reads a 512-byte page by ID.
func (pm *PageManager) ReadPage(pageID uint16) ([PageSize]byte, error) {
	var page [PageSize]byte
	if err := pm.validatePageID(pageID); err != nil {
		return page, err
	}
	if _, err := pm.file.Seek(PageOffset(pageID), io.SeekStart); err != nil {
		return page, fmt.Errorf("ndx: seeking to page %d: %w", pageID, err)
	}
	if _, err := io.ReadFull(pm.file, page[:]); err != nil {
		return page, fmt.Errorf("ndx: reading page %d: %w", pageID, err)
	}
	return page, nil
}

// WritePage writes a 512-byte page by ID.
func (pm *PageManager) WritePage(pageID uint16, page []byte) error {
	if pageID == 0 {
		return fmt.Errorf("ndx: page 0 is managed by SyncHeader")
	}
	if len(page) != PageSize {
		return fmt.Errorf("ndx: page must be %d bytes", PageSize)
	}
	if err := pm.validatePageID(pageID); err != nil {
		return err
	}
	if _, err := pm.file.Seek(PageOffset(pageID), io.SeekStart); err != nil {
		return fmt.Errorf("ndx: seeking to page %d: %w", pageID, err)
	}
	if _, err := pm.file.Write(page); err != nil {
		return fmt.Errorf("ndx: writing page %d: %w", pageID, err)
	}
	return nil
}

// ReadNode reads and decodes a B-Tree node page.
func (pm *PageManager) ReadNode(pageID uint16, kind NodeKind) (*Node, error) {
	page, err := pm.ReadPage(pageID)
	if err != nil {
		return nil, err
	}
	node, err := ParseNodePage(pm.header, kind, page[:])
	if err != nil {
		return nil, err
	}
	node.PageID = pageID
	return node, nil
}

// WriteNode encodes and writes a B-Tree node page.
func (pm *PageManager) WriteNode(node *Node) error {
	if node == nil {
		return fmt.Errorf("ndx: nil node")
	}
	if node.PageID == 0 {
		return fmt.Errorf("ndx: page 0 is reserved for the header")
	}
	page, err := MarshalNodePage(pm.header, node)
	if err != nil {
		return err
	}
	return pm.WritePage(node.PageID, page[:])
}

// AllocatePage appends a zeroed 512-byte page and returns its ID.
func (pm *PageManager) AllocatePage() (uint16, error) {
	if pm.header.PageCount == 0 {
		pm.header.PageCount = 1
	}
	if pm.header.PageCount == 65535 {
		return 0, fmt.Errorf("ndx: page count overflow")
	}
	pageID := pm.header.PageCount

	var zero [PageSize]byte
	if _, err := pm.file.Seek(PageOffset(pageID), io.SeekStart); err != nil {
		return 0, fmt.Errorf("ndx: seeking to new page %d: %w", pageID, err)
	}
	if _, err := pm.file.Write(zero[:]); err != nil {
		return 0, fmt.Errorf("ndx: writing new page %d: %w", pageID, err)
	}

	pm.header.PageCount++
	if err := pm.SyncHeader(); err != nil {
		return 0, err
	}
	return pageID, nil
}

func (pm *PageManager) validatePageID(pageID uint16) error {
	if pm.header == nil {
		return fmt.Errorf("ndx: nil header")
	}
	if pageID >= pm.header.PageCount {
		return fmt.Errorf("ndx: page %d out of range (page count %d)", pageID, pm.header.PageCount)
	}
	return nil
}
