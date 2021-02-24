package compiler

func newSymbolNodeWithPos(v byte, pos symbolPosition) *symbolNode {
	n := newSymbolNode(v)
	n.pos = pos
	return n
}

func newEndMarkerNodeWithPos(id int, pos symbolPosition) *endMarkerNode {
	n := newEndMarkerNode(id)
	n.pos = pos
	return n
}
