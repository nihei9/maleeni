package compiler

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type symbolPosition uint16

const (
	symbolPositionNil = symbolPosition(0x0000) // 0000 0000 0000 0000

	symbolPositionMin = uint16(0x0001) // 0000 0000 0000 0001
	symbolPositionMax = uint16(0x7fff) // 0111 1111 1111 1111

	symbolPositionMaskSymbol  = uint16(0x0000) // 0000 0000 0000 0000
	symbolPositionMaskEndMark = uint16(0x8000) // 1000 0000 0000 0000

	symbolPositionMaskValue = uint16(0x7fff) // 0111 1111 1111 1111
)

func newSymbolPosition(n uint16, endMark bool) (symbolPosition, error) {
	if n < symbolPositionMin || n > symbolPositionMax {
		return symbolPositionNil, fmt.Errorf("symbol position must be within %v to %v; n: %v, endMark: %v", symbolPositionMin, symbolPositionMax, n, endMark)
	}
	if endMark {
		return symbolPosition(n | symbolPositionMaskEndMark), nil
	}
	return symbolPosition(n | symbolPositionMaskSymbol), nil
}

func (p symbolPosition) String() string {
	if p.isEndMark() {
		return fmt.Sprintf("end#%v", uint16(p)&symbolPositionMaskValue)
	}
	return fmt.Sprintf("sym#%v", uint16(p)&symbolPositionMaskValue)
}

func (p symbolPosition) isEndMark() bool {
	if uint16(p)&symbolPositionMaskEndMark > 1 {
		return true
	}
	return false
}

func (p symbolPosition) describe() (uint16, bool) {
	v := uint16(p) & symbolPositionMaskValue
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

type astNode interface {
	fmt.Stringer
	children() (astNode, astNode)
	nullable() bool
	first() symbolPositionSet
	last() symbolPositionSet
}

type symbolNode struct {
	byteRange
	pos symbolPosition
}

func newSymbolNode(value byte) *symbolNode {
	return &symbolNode{
		byteRange: byteRange{
			from: value,
			to:   value,
		},
		pos: symbolPositionNil,
	}
}

func newRangeSymbolNode(from, to byte) *symbolNode {
	return &symbolNode{
		byteRange: byteRange{
			from: from,
			to:   to,
		},
		pos: symbolPositionNil,
	}
}

func (n *symbolNode) String() string {
	return fmt.Sprintf("{type: symbol, value: %v - %v, pos: %v}", n.from, n.to, n.pos)
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

func newEndMarkerNode(id int) *endMarkerNode {
	return &endMarkerNode{
		id:  id,
		pos: symbolPositionNil,
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
			left: copyAST(left),
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

func copyAST(src astNode) astNode {
	switch n := src.(type) {
	case *symbolNode:
		return newRangeSymbolNode(n.from, n.to)
	case *endMarkerNode:
		return newEndMarkerNode(n.id)
	case *concatNode:
		return newConcatNode(copyAST(n.left), copyAST(n.right))
	case *altNode:
		return newAltNode(copyAST(n.left), copyAST(n.right))
	case *repeatNode:
		return newRepeatNode(copyAST(n.left))
	case *optionNode:
		return newOptionNode(copyAST(n.left))
	}
	panic(fmt.Errorf("copyAST cannot handle %T type; AST: %v", src, src))
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

func positionSymbols(node astNode, n uint16) (uint16, error) {
	if node == nil {
		return n, nil
	}

	l, r := node.children()
	p := n
	p, err := positionSymbols(l, p)
	if err != nil {
		return p, err
	}
	p, err = positionSymbols(r, p)
	if err != nil {
		return p, err
	}
	switch n := node.(type) {
	case *symbolNode:
		n.pos, err = newSymbolPosition(p, false)
		if err != nil {
			return p, err
		}
		p++
	case *endMarkerNode:
		n.pos, err = newSymbolPosition(p, true)
		if err != nil {
			return p, err
		}
		p++
	}
	return p, nil
}

func printAST(w io.Writer, ast astNode, ruledLine string, childRuledLinePrefix string, withAttrs bool) {
	if ast == nil {
		return
	}
	fmt.Fprintf(w, ruledLine)
	fmt.Fprintf(w, "node: %v", ast)
	if withAttrs {
		fmt.Fprintf(w, ", nullable: %v, first: %v, last: %v", ast.nullable(), ast.first(), ast.last())
	}
	fmt.Fprintf(w, "\n")
	left, right := ast.children()
	children := []astNode{}
	if left != nil {
		children = append(children, left)
	}
	if right != nil {
		children = append(children, right)
	}
	num := len(children)
	for i, child := range children {
		line := "└─ "
		if num > 1 {
			if i == 0 {
				line = "├─ "
			} else if i < num-1 {
				line = "│  "
			}
		}
		prefix := "│  "
		if i >= num-1 {
			prefix = "    "
		}
		printAST(w, child, childRuledLinePrefix+line, childRuledLinePrefix+prefix, withAttrs)
	}
}
