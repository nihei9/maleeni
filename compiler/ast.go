package compiler

import (
	"fmt"
	"io"
)

type astNode interface {
	fmt.Stringer
	children() (astNode, astNode)
	nullable() bool
	first() symbolPositionSet
	last() symbolPositionSet
}

var (
	_ astNode = &symbolNode{}
	_ astNode = &endMarkerNode{}
	_ astNode = &concatNode{}
	_ astNode = &altNode{}
	_ astNode = &optionNode{}
)

type symbolNode struct {
	byteRange
	pos       symbolPosition
	firstMemo symbolPositionSet
	lastMemo  symbolPositionSet
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
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.add(n.pos)
	}
	return n.firstMemo
}

func (n *symbolNode) last() symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.add(n.pos)
	}
	return n.lastMemo
}

type endMarkerNode struct {
	id        int
	pos       symbolPosition
	firstMemo symbolPositionSet
	lastMemo  symbolPositionSet
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
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.add(n.pos)
	}
	return n.firstMemo
}

func (n *endMarkerNode) last() symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.add(n.pos)
	}
	return n.lastMemo
}

type concatNode struct {
	left      astNode
	right     astNode
	firstMemo symbolPositionSet
	lastMemo  symbolPositionSet
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
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
		if n.left.nullable() {
			n.firstMemo.merge(n.right.first())
		}
	}
	return n.firstMemo
}

func (n *concatNode) last() symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.right.last())
		if n.right.nullable() {
			n.lastMemo.merge(n.left.last())
		}
	}
	return n.lastMemo
}

type altNode struct {
	left      astNode
	right     astNode
	firstMemo symbolPositionSet
	lastMemo  symbolPositionSet
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
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
		n.firstMemo.merge(n.right.first())
	}
	return n.firstMemo
}

func (n *altNode) last() symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.left.last())
		n.lastMemo.merge(n.right.last())
	}
	return n.lastMemo
}

type repeatNode struct {
	left      astNode
	firstMemo symbolPositionSet
	lastMemo  symbolPositionSet
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
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
	}
	return n.firstMemo
}

func (n *repeatNode) last() symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.left.last())
	}
	return n.lastMemo
}

func newRepeatOneOrMoreNode(left astNode) *concatNode {
	return newConcatNode(
		left,
		&repeatNode{
			left: copyAST(left),
		})
}

type optionNode struct {
	left      astNode
	firstMemo symbolPositionSet
	lastMemo  symbolPositionSet
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
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
	}
	return n.firstMemo
}

func (n *optionNode) last() symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.left.last())
	}
	return n.lastMemo
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
	node.first()
	node.last()
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
