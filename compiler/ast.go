package compiler

import (
	"fmt"
	"sort"
	"strings"
)

type symbolPosition uint8

const (
	symbolPositionNil = symbolPosition(0)

	symbolPositionMaskSymbol  = uint8(0x00) // 0000 0000
	symbolPositionMaskEndMark = uint8(0x80) // 1000 0000

	symbolPositionMaskValue = uint8(0x7f) // 0111 1111
)

func newSymbolPosition(n uint8, endMark bool) symbolPosition {
	if endMark {
		return symbolPosition(n | symbolPositionMaskEndMark)
	}
	return symbolPosition(n | symbolPositionMaskSymbol)
}

func (p symbolPosition) String() string {
	if p.isEndMark() {
		return fmt.Sprintf("end#%v", uint8(p)&symbolPositionMaskValue)
	}
	return fmt.Sprintf("sym#%v", uint8(p)&symbolPositionMaskValue)
}

func (p symbolPosition) isEndMark() bool {
	if uint8(p)&symbolPositionMaskEndMark > 1 {
		return true
	}
	return false
}

func (p symbolPosition) describe() (uint8, bool) {
	v := uint8(p) & symbolPositionMaskValue
	if p.isEndMark() {
		return v, true
	}
	return v, false
}

type symbolPositionSet map[symbolPosition]struct{}

func newSymbolPositionSet() symbolPositionSet {
	return map[symbolPosition]struct{}{}
}

func (s symbolPositionSet) String() string {
	if len(s) <= 0 {
		return "{}"
	}
	ps := s.sort()
	var b strings.Builder
	fmt.Fprintf(&b, "{")
	for i, p := range ps {
		if i <= 0 {
			fmt.Fprintf(&b, "%v", p)
			continue
		}
		fmt.Fprintf(&b, ", %v", p)
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func (s symbolPositionSet) add(pos symbolPosition) symbolPositionSet {
	s[pos] = struct{}{}
	return s
}

func (s symbolPositionSet) merge(t symbolPositionSet) symbolPositionSet {
	for p := range t {
		s.add(p)
	}
	return s
}

func (s symbolPositionSet) intersect(set symbolPositionSet) symbolPositionSet {
	in := newSymbolPositionSet()
	for p1 := range s {
		for p2 := range set {
			if p1 != p2 {
				continue
			}
			in.add(p1)
		}
	}
	return in
}

func (s symbolPositionSet) hash() string {
	if len(s) <= 0 {
		return ""
	}
	sorted := s.sort()
	var b strings.Builder
	fmt.Fprintf(&b, "%v", sorted[0])
	for _, p := range sorted[1:] {
		fmt.Fprintf(&b, ":%v", p)
	}
	return b.String()
}

func (s symbolPositionSet) sort() []symbolPosition {
	sorted := []symbolPosition{}
	for p := range s {
		sorted = append(sorted, p)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}

type byteRange struct {
	from byte
	to   byte
}

func (r byteRange) String() string {
	return fmt.Sprintf("%v - %v", r.from, r.to)
}

type astNode interface {
	fmt.Stringer
	children() (astNode, astNode)
	nullable() bool
	first() symbolPositionSet
	last() symbolPositionSet
}

type symbolNode struct {
	byteRange
	token *token
	pos   symbolPosition
}

func newSymbolNode(tok *token, value byte, pos symbolPosition) *symbolNode {
	return &symbolNode{
		byteRange: byteRange{
			from: value,
			to:   value,
		},
		token: tok,
		pos:   pos,
	}
}

func newRangeSymbolNode(tok *token, from, to byte, pos symbolPosition) *symbolNode {
	return &symbolNode{
		byteRange: byteRange{
			from: from,
			to:   to,
		},
		token: tok,
		pos:   pos,
	}
}

func (n *symbolNode) String() string {
	return fmt.Sprintf("{type: symbol, value: %v - %v, token char: %v, pos: %v}", n.from, n.to, string(n.token.char), n.pos)
}

func (n *symbolNode) children() (astNode, astNode) {
	return nil, nil
}

func (n *symbolNode) nullable() bool {
	return false
}

func (n *symbolNode) first() symbolPositionSet {
	s := newSymbolPositionSet()
	s.add(n.pos)
	return s
}

func (n *symbolNode) last() symbolPositionSet {
	s := newSymbolPositionSet()
	s.add(n.pos)
	return s
}

type endMarkerNode struct {
	id  int
	pos symbolPosition
}

func newEndMarkerNode(id int, pos symbolPosition) *endMarkerNode {
	return &endMarkerNode{
		id:  id,
		pos: pos,
	}
}

func (n *endMarkerNode) String() string {
	return fmt.Sprintf("{type: end, pos: %v}", n.pos)
}

func (n *endMarkerNode) children() (astNode, astNode) {
	return nil, nil
}

func (n *endMarkerNode) nullable() bool {
	return false
}

func (n *endMarkerNode) first() symbolPositionSet {
	s := newSymbolPositionSet()
	s.add(n.pos)
	return s
}

func (n *endMarkerNode) last() symbolPositionSet {
	s := newSymbolPositionSet()
	s.add(n.pos)
	return s
}

type concatNode struct {
	left  astNode
	right astNode
}

func newConcatNode(left, right astNode) *concatNode {
	return &concatNode{
		left:  left,
		right: right,
	}
}

func (n *concatNode) String() string {
	return fmt.Sprintf("{type: concat}")
}

func (n *concatNode) children() (astNode, astNode) {
	return n.left, n.right
}

func (n *concatNode) nullable() bool {
	return n.left.nullable() && n.right.nullable()
}

func (n *concatNode) first() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.first())
	if n.left.nullable() {
		s.merge(n.right.first())
	}
	return s
}

func (n *concatNode) last() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.right.last())
	if n.right.nullable() {
		s.merge(n.left.last())
	}
	return s
}

type altNode struct {
	left  astNode
	right astNode
}

func newAltNode(left, right astNode) *altNode {
	return &altNode{
		left:  left,
		right: right,
	}
}

func (n *altNode) String() string {
	return fmt.Sprintf("{type: alt}")
}

func (n *altNode) children() (astNode, astNode) {
	return n.left, n.right
}

func (n *altNode) nullable() bool {
	return n.left.nullable() || n.right.nullable()
}

func (n *altNode) first() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.first())
	s.merge(n.right.first())
	return s
}

func (n *altNode) last() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.last())
	s.merge(n.right.last())
	return s
}

type repeatNode struct {
	left astNode
}

func newRepeatNode(left astNode) *repeatNode {
	return &repeatNode{
		left: left,
	}
}

func (n *repeatNode) String() string {
	return fmt.Sprintf("{type: repeat}")
}

func (n *repeatNode) children() (astNode, astNode) {
	return n.left, nil
}

func (n *repeatNode) nullable() bool {
	return true
}

func (n *repeatNode) first() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.first())
	return s
}

func (n *repeatNode) last() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.last())
	return s
}

func newRepeatOneOrMoreNode(left astNode) *concatNode {
	return newConcatNode(
		left,
		&repeatNode{
			left: left,
		})
}

type optionNode struct {
	left astNode
}

func newOptionNode(left astNode) *optionNode {
	return &optionNode{
		left: left,
	}
}

func (n *optionNode) String() string {
	return fmt.Sprintf("{type: option}")
}

func (n *optionNode) children() (astNode, astNode) {
	return n.left, nil
}

func (n *optionNode) nullable() bool {
	return true
}

func (n *optionNode) first() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.first())
	return s
}

func (n *optionNode) last() symbolPositionSet {
	s := newSymbolPositionSet()
	s.merge(n.left.last())
	return s
}

type followTable map[symbolPosition]symbolPositionSet

func genFollowTable(root astNode) followTable {
	follow := followTable{}
	calcFollow(follow, root)
	return follow
}

func calcFollow(follow followTable, ast astNode) {
	if ast == nil {
		return
	}
	left, right := ast.children()
	calcFollow(follow, left)
	calcFollow(follow, right)
	switch n := ast.(type) {
	case *concatNode:
		l, r := n.children()
		for _, p := range l.last().sort() {
			if _, ok := follow[p]; !ok {
				follow[p] = newSymbolPositionSet()
			}
			follow[p].merge(r.first())
		}
	case *repeatNode:
		for _, p := range n.last().sort() {
			if _, ok := follow[p]; !ok {
				follow[p] = newSymbolPositionSet()
			}
			follow[p].merge(n.first())
		}
	}
}

func positionSymbols(node astNode, n uint8) uint8 {
	if node == nil {
		return n
	}

	l, r := node.children()
	p := n
	p = positionSymbols(l, p)
	p = positionSymbols(r, p)
	switch n := node.(type) {
	case *symbolNode:
		n.pos = newSymbolPosition(p, false)
		p++
	case *endMarkerNode:
		n.pos = newSymbolPosition(p, true)
		p++
	}
	return p
}
