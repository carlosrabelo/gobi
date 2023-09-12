package expr

// Area holds the open-table facts that cursor built-ins need.
// With no table, HasTable is false, counts are 0, and paths are empty.
type Area struct {
	HasTable bool
	RecNo    int // 1-based; 0 when no table; RecCount+1 when past last
	RecCount int
	RecSize  int
	Deleted  bool
	Found    bool
	BOF      bool
	Fields   []string
	DBF      string
	NDX      [7]string
}
