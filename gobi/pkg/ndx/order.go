package ndx

import "fmt"

// OrderedRecordNumbers walks the B-Tree in key order and returns every
// record number, providing the traversal sequence for index-ordered
// navigation (GO TOP/BOTTOM, SKIP, LIST).
func (pm *PageManager) OrderedRecordNumbers() ([]uint16, error) {
	if pm == nil || pm.header == nil {
		return nil, fmt.Errorf("ndx: nil page manager")
	}
	if pm.header.RootPageID == 0 {
		return nil, nil
	}
	var records []uint16
	if err := pm.appendOrderedRecords(pm.header.RootPageID, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (pm *PageManager) appendOrderedRecords(pageID uint16, out *[]uint16) error {
	page, err := pm.ReadPage(pageID)
	if err != nil {
		return err
	}

	switch pageNodeKind(pm.header, page[:]) {
	case NodeKindLeaf:
		node, err := ParseNodePage(pm.header, NodeKindLeaf, page[:])
		if err != nil {
			return err
		}
		for _, entry := range node.Leaf {
			*out = append(*out, entry.RecordNumber)
		}
		return nil
	case NodeKindInternal:
		node, err := ParseNodePage(pm.header, NodeKindInternal, page[:])
		if err != nil {
			return err
		}
		for _, entry := range node.Internal {
			if err := pm.appendOrderedRecords(entry.ChildPageID, out); err != nil {
				return err
			}
		}
		if node.RightChild != 0 {
			return pm.appendOrderedRecords(node.RightChild, out)
		}
		return nil
	default:
		return fmt.Errorf("ndx: invalid page kind at page %d", pageID)
	}
}
