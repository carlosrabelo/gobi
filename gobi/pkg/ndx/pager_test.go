package ndx

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

type ndxFile struct {
	data []byte
	pos  int
}

func (f *ndxFile) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *ndxFile) Write(p []byte) (int, error) {
	end := f.pos + len(p)
	for end > len(f.data) {
		f.data = append(f.data, 0)
	}
	copy(f.data[f.pos:], p)
	f.pos = end
	return len(p), nil
}

func (f *ndxFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.pos = int(offset)
	case io.SeekCurrent:
		f.pos += int(offset)
	case io.SeekEnd:
		f.pos = len(f.data) + int(offset)
	}
	if f.pos < 0 {
		f.pos = 0
	}
	return int64(f.pos), nil
}

func newTestHeader() *Header {
	return &Header{
		RootPageID:     0,
		PageCount:      1,
		KeyLength:      10,
		MaxKeysPerPage: 42,
		KeyType:        KeyTypeCharacter,
		Expression:     "NAME",
	}
}

func TestPageOffset(t *testing.T) {
	if PageOffset(0) != 0 {
		t.Fatalf("page 0 offset = %d, want 0", PageOffset(0))
	}
	if PageOffset(3) != 3*PageSize {
		t.Fatalf("page 3 offset = %d, want %d", PageOffset(3), 3*PageSize)
	}
}

func TestCreatePageManagerInitializesHeaderPage(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if len(file.data) != PageSize {
		t.Fatalf("expected %d-byte file, got %d", PageSize, len(file.data))
	}
	if pm.Header().PageCount != 1 {
		t.Fatalf("page count = %d, want 1", pm.Header().PageCount)
	}
}

func TestAllocatePageExtendsFile(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}

	page1, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	if page1 != 1 {
		t.Fatalf("first page id = %d, want 1", page1)
	}
	if pm.Header().PageCount != 2 {
		t.Fatalf("page count = %d, want 2", pm.Header().PageCount)
	}
	if len(file.data) != 2*PageSize {
		t.Fatalf("file size = %d, want %d", len(file.data), 2*PageSize)
	}

	page2, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("second AllocatePage: %v", err)
	}
	if page2 != 2 {
		t.Fatalf("second page id = %d, want 2", page2)
	}
	if len(file.data) != 3*PageSize {
		t.Fatalf("file size = %d, want %d", len(file.data), 3*PageSize)
	}

	got, err := pm.ReadPage(page1)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	for i, b := range got {
		if b != 0 {
			t.Fatalf("expected zeroed page at byte %d, got 0x%02X", i, b)
		}
	}
}

func TestOpenPageManagerReadsExistingFile(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if _, err := pm.AllocatePage(); err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}

	opened, err := OpenPageManager(file)
	if err != nil {
		t.Fatalf("OpenPageManager: %v", err)
	}
	if opened.Header().PageCount != 2 {
		t.Fatalf("opened page count = %d, want 2", opened.Header().PageCount)
	}
}

func TestWritePageAndReadNodeRoundTrip(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}

	node := &Node{
		PageID: pageID,
		Kind:   NodeKindLeaf,
		Leaf: []LeafEntry{
			{RecordNumber: 3, Key: Key("Alice")},
		},
	}
	if err := pm.WriteNode(node); err != nil {
		t.Fatalf("WriteNode: %v", err)
	}

	got, err := pm.ReadNode(pageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	if len(got.Leaf) != 1 || got.Leaf[0].RecordNumber != 3 {
		t.Fatalf("unexpected node: %#v", got)
	}
}

func TestSyncHeaderPersistsRootPage(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pm.Header().RootPageID = 7
	if err := pm.SyncHeader(); err != nil {
		t.Fatalf("SyncHeader: %v", err)
	}

	opened, err := OpenPageManager(file)
	if err != nil {
		t.Fatalf("OpenPageManager: %v", err)
	}
	if opened.Header().RootPageID != 7 {
		t.Fatalf("root page = %d, want 7", opened.Header().RootPageID)
	}
}

func TestReadPageOutOfRange(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	_, err = pm.ReadPage(1)
	if err == nil {
		t.Fatal("expected out of range error")
	}
}

func TestWritePageRejectsHeaderPage(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	err = pm.WritePage(0, make([]byte, PageSize))
	if err == nil {
		t.Fatal("expected page 0 write error")
	}
}

func TestWriteNodeRejectsHeaderPage(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	err = pm.WriteNode(&Node{PageID: 0, Kind: NodeKindLeaf})
	if err == nil {
		t.Fatal("expected page 0 node write error")
	}
}

func TestCreatePageManagerNilInputs(t *testing.T) {
	if _, err := CreatePageManager(nil, newTestHeader()); err == nil {
		t.Fatal("expected nil file error")
	}
	if _, err := CreatePageManager(&ndxFile{}, nil); err == nil {
		t.Fatal("expected nil header error")
	}
}

func TestOpenPageManagerNilFile(t *testing.T) {
	if _, err := OpenPageManager(nil); err == nil {
		t.Fatal("expected nil file error")
	}
}

func TestWritePageWrongSize(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	err = pm.WritePage(pageID, []byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected wrong page size error")
	}
}

func TestAllocatePageUpdatesHeaderOnDisk(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if _, err := pm.AllocatePage(); err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	if binary.LittleEndian.Uint16(file.data[2:4]) != 2 {
		t.Fatalf("on-disk page count = %d, want 2", binary.LittleEndian.Uint16(file.data[2:4]))
	}
}

func TestReadPageShortFile(t *testing.T) {
	file := &ndxFile{data: make([]byte, PageSize)}
	if _, err := CreatePageManager(file, newTestHeader()); err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pm := &PageManager{file: file, header: &Header{PageCount: 2}}
	_, err := pm.ReadPage(1)
	if err == nil {
		t.Fatal("expected short read error")
	}
}

func TestSyncHeaderWriteError(t *testing.T) {
	pm := &PageManager{
		file:   failingSeeker{err: errors.New("seek failed")},
		header: newTestHeader(),
	}
	if err := pm.SyncHeader(); err == nil {
		t.Fatal("expected sync header error")
	}
}

type failingSeeker struct {
	err error
}

func (f failingSeeker) Read([]byte) (int, error)  { return 0, f.err }
func (f failingSeeker) Write([]byte) (int, error) { return 0, f.err }
func (f failingSeeker) Seek(int64, int) (int64, error) {
	return 0, f.err
}

func TestCreatePageManagerDefaultsPageCount(t *testing.T) {
	h := newTestHeader()
	h.PageCount = 0
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if pm.Header().PageCount != 1 {
		t.Fatalf("page count = %d, want 1", pm.Header().PageCount)
	}
}

func TestWriteNodeNilNode(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.WriteNode(nil); err == nil {
		t.Fatal("expected nil node error")
	}
}

func TestOpenPageManagerShortHeader(t *testing.T) {
	file := &ndxFile{data: make([]byte, PageSize-1)}
	_, err := OpenPageManager(file)
	if err == nil {
		t.Fatal("expected short header error")
	}
}

func TestValidatePageIDNilHeader(t *testing.T) {
	pm := &PageManager{file: &ndxFile{}}
	_, err := pm.ReadPage(0)
	if err == nil {
		t.Fatal("expected nil header error")
	}
}

func TestAllocatePageWriteError(t *testing.T) {
	pm := &PageManager{
		file:   failingWriteSeeker{writeErr: errors.New("write failed")},
		header: newTestHeader(),
	}
	_, err := pm.AllocatePage()
	if err == nil {
		t.Fatal("expected allocate write error")
	}
}

func TestAllocatePageSeekError(t *testing.T) {
	pm := &PageManager{
		file:   failingWriteSeeker{seekErr: errors.New("seek failed")},
		header: newTestHeader(),
	}
	_, err := pm.AllocatePage()
	if err == nil {
		t.Fatal("expected allocate seek error")
	}
}

func TestWritePageSeekError(t *testing.T) {
	pm := &PageManager{
		file:   failingWriteSeeker{seekErr: errors.New("seek failed")},
		header: &Header{PageCount: 2},
	}
	err := pm.WritePage(1, make([]byte, PageSize))
	if err == nil {
		t.Fatal("expected write page seek error")
	}
}

func TestWritePageWriteError(t *testing.T) {
	pm := &PageManager{
		file:   failingWriteSeeker{writeErr: errors.New("write failed")},
		header: &Header{PageCount: 2},
	}
	err := pm.WritePage(1, make([]byte, PageSize))
	if err == nil {
		t.Fatal("expected write page write error")
	}
}

func TestReadPageSeekError(t *testing.T) {
	pm := &PageManager{
		file:   failingWriteSeeker{seekErr: errors.New("seek failed")},
		header: &Header{PageCount: 2},
	}
	_, err := pm.ReadPage(1)
	if err == nil {
		t.Fatal("expected read page seek error")
	}
}

func TestReadNodeParseError(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	page, err := pm.ReadPage(pageID)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	binary.LittleEndian.PutUint16(page[0:2], 0xFFFF)
	if err := pm.WritePage(pageID, page[:]); err != nil {
		t.Fatalf("WritePage: %v", err)
	}
	_, err = pm.ReadNode(pageID, NodeKindLeaf)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestPageManagerWriteNodeMarshalError(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	err = pm.WriteNode(&Node{PageID: pageID, Kind: NodeKind(99)})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestCreatePageManagerInvalidHeader(t *testing.T) {
	h := newTestHeader()
	h.KeyType = 9
	_, err := CreatePageManager(&ndxFile{}, h)
	if err == nil {
		t.Fatal("expected invalid header error")
	}
}

func TestOpenPageManagerSeekError(t *testing.T) {
	_, err := OpenPageManager(failingWriteSeeker{seekErr: errors.New("seek failed")})
	if err == nil {
		t.Fatal("expected open seek error")
	}
}

func TestWritePageOutOfRange(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	err = pm.WritePage(9, make([]byte, PageSize))
	if err == nil {
		t.Fatal("expected out of range error")
	}
}

func TestReadNodeReadPageError(t *testing.T) {
	pm := &PageManager{
		file:   &ndxFile{data: make([]byte, PageSize)},
		header: &Header{PageCount: 2},
	}
	_, err := pm.ReadNode(1, NodeKindLeaf)
	if err == nil {
		t.Fatal("expected read node error")
	}
}

func TestAllocatePageSyncHeaderError(t *testing.T) {
	inner := &ndxFile{}
	pm := &PageManager{
		file:   &seekFailAtZero{inner: inner},
		header: newTestHeader(),
	}
	_, err := pm.AllocatePage()
	if err == nil {
		t.Fatal("expected sync header error")
	}
}

func TestAllocatePageNormalizesZeroPageCount(t *testing.T) {
	file := &ndxFile{}
	pm := &PageManager{file: file, header: newTestHeader()}
	pm.header.PageCount = 0
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	if pageID != 1 {
		t.Fatalf("page id = %d, want 1", pageID)
	}
}

func TestCreatePageManagerSeekError(t *testing.T) {
	_, err := CreatePageManager(failingWriteSeeker{seekErr: errors.New("seek failed")}, newTestHeader())
	if err == nil {
		t.Fatal("expected create seek error")
	}
}

func TestCreatePageManagerWriteHeaderError(t *testing.T) {
	_, err := CreatePageManager(failingWriteSeeker{writeErr: errors.New("write failed")}, newTestHeader())
	if err == nil {
		t.Fatal("expected write header error")
	}
}

type seekFailAtZero struct {
	inner     *ndxFile
	wrotePage bool
}

func (s *seekFailAtZero) Read(p []byte) (int, error)  { return s.inner.Read(p) }
func (s *seekFailAtZero) Write(p []byte) (int, error) { return s.inner.Write(p) }
func (s *seekFailAtZero) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && s.wrotePage {
		return 0, errors.New("sync seek failed")
	}
	pos, err := s.inner.Seek(offset, whence)
	if err == nil && offset == int64(PageSize) {
		s.wrotePage = true
	}
	return pos, err
}

func TestAllocatePagePageCountOverflow(t *testing.T) {
	pm := &PageManager{
		file:   &ndxFile{},
		header: newTestHeader(),
	}
	pm.header.PageCount = 65535
	_, err := pm.AllocatePage()
	if err == nil {
		t.Fatal("expected page count overflow error")
	}
}

type failingWriteSeeker struct {
	seekErr  error
	writeErr error
}

func (f failingWriteSeeker) Read([]byte) (int, error)  { return 0, io.EOF }
func (f failingWriteSeeker) Write([]byte) (int, error) { return 0, f.writeErr }
func (f failingWriteSeeker) Seek(int64, int) (int64, error) {
	if f.seekErr != nil {
		return 0, f.seekErr
	}
	return 0, nil
}
