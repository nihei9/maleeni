package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type SyntaxError struct {
	message string
}

func (err *SyntaxError) Error() string {
	return fmt.Sprintf("Syntax Error: %v", err.message)
}

func raiseSyntaxError(format string, a ...interface{}) {
	panic(&SyntaxError{
		message: fmt.Sprintf(format, a...),
	})
}

type symbolTable struct {
	symPos2Byte map[symbolPosition]byteRange
	endPos2ID   map[symbolPosition]int
}

func genSymbolTable(root astNode) *symbolTable {
	symTab := &symbolTable{
		symPos2Byte: map[symbolPosition]byteRange{},
		endPos2ID:   map[symbolPosition]int{},
	}
	return genSymTab(symTab, root)
}

func genSymTab(symTab *symbolTable, node astNode) *symbolTable {
	if node == nil {
		return symTab
	}

	switch n := node.(type) {
	case *symbolNode:
		symTab.symPos2Byte[n.pos] = byteRange{
			from: n.from,
			to:   n.to,
		}
	case *endMarkerNode:
		symTab.endPos2ID[n.pos] = n.id
	default:
		left, right := node.children()
		genSymTab(symTab, left)
		genSymTab(symTab, right)
	}
	return symTab
}

func parse(regexps map[int][]byte) (astNode, *symbolTable, error) {
	if len(regexps) == 0 {
		return nil, nil, fmt.Errorf("parse() needs at least one token entry")
	}
	var root astNode
	for id, re := range regexps {
		if len(re) == 0 {
			return nil, nil, fmt.Errorf("regular expression must be a non-empty byte sequence")
		}
		p := newParser(id, bytes.NewReader(re))
		n, err := p.parse()
		if err != nil {
			return nil, nil, err
		}
		if root == nil {
			root = n
		} else {
			root = newAltNode(root, n)
		}
	}
	positionSymbols(root, 1)

	return root, genSymbolTable(root), nil
}

type parser struct {
	id        int
	lex       *lexer
	peekedTok *token
	lastTok   *token
}

func newParser(id int, src io.Reader) *parser {
	return &parser{
		id:        id,
		lex:       newLexer(src),
		peekedTok: nil,
		lastTok:   nil,
	}
}

func (p *parser) parse() (astNode, error) {
	return p.parseRegexp()
}

func (p *parser) parseRegexp() (ast astNode, retErr error) {
	defer func() {
		err := recover()
		if err != nil {
			retErr = err.(error)
			var synErr SyntaxError
			if !errors.Is(retErr, &synErr) {
				panic(err)
			}
			return
		}
	}()

	alt := p.parseAlt()
	p.expect(tokenKindEOF)
	return newConcatNode(alt, newEndMarkerNode(p.id, symbolPositionNil)), nil
}

func (p *parser) parseAlt() astNode {
	left := p.parseConcat()
	for {
		if !p.consume(tokenKindAlt) {
			break
		}
		right := p.parseConcat()
		left = newAltNode(left, right)
	}
	return left
}

func (p *parser) parseConcat() astNode {
	left := p.parseRepeat()
	for {
		right := p.parseRepeat()
		if right == nil {
			break
		}
		left = newConcatNode(left, right)
	}
	return left
}

func (p *parser) parseRepeat() astNode {
	group := p.parseGroup()
	if p.consume(tokenKindRepeat) {
		return newRepeatNode(group)
	}
	if p.consume(tokenKindRepeatOneOrMore) {
		return newRepeatOneOrMoreNode(group)
	}
	if p.consume(tokenKindOption) {
		return newOptionNode(group)
	}
	return group
}

func (p *parser) parseGroup() astNode {
	if p.consume(tokenKindGroupOpen) {
		defer p.expect(tokenKindGroupClose)
		return p.parseAlt()
	}
	return p.parseSingleChar()
}

func (p *parser) parseSingleChar() astNode {
	if p.consume(tokenKindAnyChar) {
		return genAnyCharAST(p.lastTok)
	}
	if p.consume(tokenKindBExpOpen) {
		defer p.expect(tokenKindBExpClose)
		left := p.parseBExpElem()
		if left == nil {
			raiseSyntaxError("bracket expression must include at least one character")
		}
		for {
			right := p.parseBExpElem()
			if right == nil {
				break
			}
			left = newAltNode(left, right)
		}
		return left
	}
	return p.parseNormalChar()
}

func (p *parser) parseBExpElem() astNode {
	left := p.parseNormalChar()
	if !p.consume(tokenKindCharRange) {
		return left
	}
	right := p.parseNormalChar()
	if right == nil {
		raiseSyntaxError("invalid range expression")
	}
	return genRangeAST(left, right)
}

func (p *parser) parseNormalChar() astNode {
	if !p.consume(tokenKindChar) {
		return nil
	}

	b := []byte(string(p.lastTok.char))
	switch len(b) {
	case 1:
		return newSymbolNode(p.lastTok, b[0], symbolPositionNil)
	case 2:
		return newConcatNode(
			newSymbolNode(p.lastTok, b[0], symbolPositionNil),
			newSymbolNode(p.lastTok, b[1], symbolPositionNil),
		)
	case 3:
		return newConcatNode(
			newConcatNode(
				newSymbolNode(p.lastTok, b[0], symbolPositionNil),
				newSymbolNode(p.lastTok, b[1], symbolPositionNil),
			),
			newSymbolNode(p.lastTok, b[2], symbolPositionNil),
		)
	default: // is equivalent to case 4
		return newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(p.lastTok, b[0], symbolPositionNil),
					newSymbolNode(p.lastTok, b[1], symbolPositionNil),
				),
				newSymbolNode(p.lastTok, b[2], symbolPositionNil),
			),
			newSymbolNode(p.lastTok, b[3], symbolPositionNil),
		)
	}
}

// Refelences:
// * https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf#G7404
//   * Table 3-6.  UTF-8 Bit Distribution
//   * Table 3-7.  Well-Formed UTF-8 Byte Sequences
func genAnyCharAST(tok *token) astNode {
	return newAltNode(
		newAltNode(
			newAltNode(
				newAltNode(
					newAltNode(
						newAltNode(
							newAltNode(
								newAltNode(
									// 1 byte character <00..7F>
									newRangeSymbolNode(tok, 0x00, 0x7f, symbolPositionNil),
									// 2 bytes character <C2..DF 80..BF>
									newConcatNode(
										newRangeSymbolNode(tok, 0xc2, 0xdf, symbolPositionNil),
										newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
									),
								),
								// 3 bytes character <E0 A0..BF 80..BF>
								newConcatNode(
									newConcatNode(
										newSymbolNode(tok, 0xe0, symbolPositionNil),
										newRangeSymbolNode(tok, 0xa0, 0xbf, symbolPositionNil),
									),
									newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
								),
							),
							// 3 bytes character <E1..EC 80..BF 80..BF>
							newConcatNode(
								newConcatNode(
									newRangeSymbolNode(tok, 0xe1, 0xec, symbolPositionNil),
									newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
								),
								newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
							),
						),
						// 3 bytes character <ED 80..9F 80..BF>
						newConcatNode(
							newConcatNode(
								newSymbolNode(tok, 0xed, symbolPositionNil),
								newRangeSymbolNode(tok, 0x80, 0x9f, symbolPositionNil),
							),
							newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
						),
					),
					// 3 bytes character <EE..EF 80..BF 80..BF>
					newConcatNode(
						newConcatNode(
							newRangeSymbolNode(tok, 0xee, 0xef, symbolPositionNil),
							newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
						),
						newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
					),
				),
				// 4 bytes character <F0 90..BF 80..BF 80..BF>
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(tok, 0xf0, symbolPositionNil),
							newRangeSymbolNode(tok, 0x90, 0xbf, symbolPositionNil),
						),
						newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
					),
					newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
				),
			),
			// 4 bytes character <F1..F3 80..BF 80..BF 80..BF>
			newConcatNode(
				newConcatNode(
					newConcatNode(
						newRangeSymbolNode(tok, 0xf1, 0xf3, symbolPositionNil),
						newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
					),
					newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
				),
				newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
			),
		),
		// 4 bytes character <F4 80..8F 80..BF 80..BF>
		newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(tok, 0xf4, symbolPositionNil),
					newRangeSymbolNode(tok, 0x80, 0x8f, symbolPositionNil),
				),
				newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
			),
			newRangeSymbolNode(tok, 0x80, 0xbf, symbolPositionNil),
		),
	)
}

func genRangeAST(fromNode, toNode astNode) astNode {
	from := genByteSeq(fromNode)
	to := genByteSeq(toNode)
	if !isValidOrder(from, to) {
		raiseSyntaxError("range expression with invalid order: [%s-%s] ([%v-%v])", string(from), string(to), from, to)
	}
	switch len(from) {
	case 1:
		switch len(to) {
		case 1:
			return gen1ByteCharRangeAST(from, to)
		case 2:
			return newAltNode(
				gen1ByteCharRangeAST(from, []byte{0x7f}),
				gen2ByteCharRangeAST([]byte{0xc2, 0x80}, to),
			)
		case 3:
			return newAltNode(
				newAltNode(
					gen1ByteCharRangeAST(from, []byte{0x7f}),
					gen2ByteCharRangeAST([]byte{0xc2, 0x80}, []byte{0xdf, 0xbf}),
				),
				gen3ByteCharRangeAST([]byte{0xe0, 0xa0, 0x80}, to),
			)
		case 4:
			return newAltNode(
				newAltNode(
					newAltNode(
						gen1ByteCharRangeAST(from, []byte{0x7f}),
						gen2ByteCharRangeAST([]byte{0xc2, 0x80}, []byte{0xdf, 0xbf}),
					),
					gen3ByteCharRangeAST([]byte{0xe0, 0xa0, 0x80}, []byte{0xef, 0xbf, 0xbf}),
				),
				gen4ByteCharRangeAST([]byte{0xf0, 0x90, 0x80}, to),
			)
		}
	case 2:
		switch len(to) {
		case 2:
			return gen2ByteCharRangeAST(from, to)
		case 3:
			return newAltNode(
				gen2ByteCharRangeAST(from, []byte{0xdf, 0xbf}),
				gen3ByteCharRangeAST([]byte{0xc2, 0x80}, to),
			)
		case 4:
			return newAltNode(
				newAltNode(
					gen2ByteCharRangeAST(from, []byte{0xdf, 0xbf}),
					gen3ByteCharRangeAST([]byte{0xc2, 0x80}, []byte{0xef, 0xbf, 0xbf}),
				),
				gen4ByteCharRangeAST([]byte{0xf0, 0x90, 0x80}, to),
			)
		}
	case 3:
		switch len(to) {
		case 3:
			return gen3ByteCharRangeAST(from, to)
		case 4:
			return newAltNode(
				gen3ByteCharRangeAST(from, []byte{0xef, 0xbf, 0xbf}),
				gen4ByteCharRangeAST([]byte{0xf0, 0x90, 0x80}, to),
			)
		}
	case 4:
		return gen4ByteCharRangeAST(from, to)
	}
	panic(fmt.Sprintf("invalid range; from: %v, to: %v", from, to))
}

func genByteSeq(node astNode) []byte {
	switch n := node.(type) {
	case *symbolNode:
		return []byte{n.from}
	case *concatNode:
		seq := genByteSeq(n.left)
		seq = append(seq, genByteSeq(n.right)...)
		return seq
	}
	panic(fmt.Sprintf("genByteSeq() cannot handle %T: %v", node, node))
}

func isValidOrder(from, to []byte) bool {
	if len(from) > len(to) {
		return false
	}
	if len(from) < len(to) {
		return true
	}
	for i, f := range from {
		t := to[i]
		if f > t {
			return false
		}
		if f < t {
			return true
		}
	}
	return true
}

func gen1ByteCharRangeAST(from, to []byte) astNode {
	return newRangeSymbolNode(nil, from[0], to[0], symbolPositionNil)
}

func gen2ByteCharRangeAST(from, to []byte) astNode {
	from0 := from[0]
	from1 := from[1]
	to0 := to[0]
	to1 := to[1]
	switch {
	case from0 == to0 && from1 == to1:
		return newConcatNode(
			newSymbolNode(nil, from0, symbolPositionNil),
			newSymbolNode(nil, from1, symbolPositionNil),
		)
	case from0 == to0:
		return newConcatNode(
			newSymbolNode(nil, from0, symbolPositionNil),
			newRangeSymbolNode(nil, from1, to1, symbolPositionNil),
		)
	default:
		alt1 := newConcatNode(
			newSymbolNode(nil, from0, symbolPositionNil),
			newRangeSymbolNode(nil, from1, 0xbf, symbolPositionNil),
		)
		alt2 := newConcatNode(
			newRangeSymbolNode(nil, from0+1, to0, symbolPositionNil),
			newRangeSymbolNode(nil, 0x80, to1, symbolPositionNil),
		)
		return newAltNode(alt1, alt2)
	}
}

type byteBoundsEntry struct {
	min byte
	max byte
}

var (
	bounds3 = [][]byteBoundsEntry{
		nil,
		{{min: 0xe0, max: 0xe0}, {min: 0xa0, max: 0xbf}, {min: 0x80, max: 0xbf}},
		{{min: 0xe1, max: 0xec}, {min: 0x80, max: 0xbf}, {min: 0x80, max: 0xbf}},
		{{min: 0xed, max: 0xed}, {min: 0x80, max: 0x9f}, {min: 0x80, max: 0xbf}},
		{{min: 0xee, max: 0xef}, {min: 0x80, max: 0xbf}, {min: 0x80, max: 0xbf}},
	}

	bounds4 = [][]byteBoundsEntry{
		nil,
		{{min: 0xf0, max: 0xf0}, {min: 0x90, max: 0xbf}, {min: 0x80, max: 0xbf}, {min: 0x80, max: 0xbf}},
		{{min: 0xf1, max: 0xf3}, {min: 0x80, max: 0xbf}, {min: 0x80, max: 0xbf}, {min: 0x80, max: 0xbf}},
		{{min: 0xf4, max: 0xf4}, {min: 0x80, max: 0x8f}, {min: 0x80, max: 0xbf}, {min: 0x80, max: 0xbf}},
	}
)

func get3ByteCharRangeNum(seq []byte) int {
	head := seq[0]
	switch {
	case head == 0xe0:
		return 1
	case head >= 0xe1 && head <= 0xec:
		return 2
	case head == 0xed:
		return 3
	case head >= 0xee && head <= 0xef:
		return 4
	}
	return 0
}

func get4ByteCharRangeNum(seq []byte) int {
	head := seq[0]
	switch {
	case head == 0xf0:
		return 1
	case head >= 0xf1 && head <= 0xf3:
		return 2
	case head == 0xf4:
		return 3
	}
	return 0
}

func gen3ByteCharRangeAST(from, to []byte) astNode {
	from0 := from[0]
	from1 := from[1]
	from2 := from[2]
	to0 := to[0]
	to1 := to[1]
	to2 := to[2]
	switch {
	case from0 == to0 && from1 == to1 && from2 == to2:
		return newConcatNode(
			newConcatNode(
				newSymbolNode(nil, from0, symbolPositionNil),
				newSymbolNode(nil, from1, symbolPositionNil),
			),
			newSymbolNode(nil, from2, symbolPositionNil),
		)
	case from0 == to0 && from1 == to1:
		return newConcatNode(
			newConcatNode(
				newSymbolNode(nil, from0, symbolPositionNil),
				newSymbolNode(nil, from1, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from2, to2, symbolPositionNil),
		)
	case from0 == to0:
		rangeNum := get3ByteCharRangeNum(from)
		bounds := bounds3[rangeNum]
		var alt astNode
		alt = newConcatNode(
			newConcatNode(
				newSymbolNode(nil, from0, symbolPositionNil),
				newSymbolNode(nil, from1, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from2, bounds[2].max, symbolPositionNil),
		)
		if from1+1 < to1 {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, from0, symbolPositionNil),
						newRangeSymbolNode(nil, from1+1, to1-1, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
				),
			)
		}
		alt = newAltNode(
			alt,
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, from0, symbolPositionNil),
					newSymbolNode(nil, to1, symbolPositionNil),
				),
				newRangeSymbolNode(nil, bounds[2].min, to2, symbolPositionNil),
			),
		)
		return alt
	default:
		fromRangeNum := get3ByteCharRangeNum(from)
		toRangeNum := get3ByteCharRangeNum(to)
		bounds := bounds3[fromRangeNum]
		var alt astNode
		alt = newConcatNode(
			newConcatNode(
				newSymbolNode(nil, from0, symbolPositionNil),
				newSymbolNode(nil, from1, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from2, bounds[2].max, symbolPositionNil),
		)
		if from1 < bounds[1].max {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, from0, symbolPositionNil),
						newRangeSymbolNode(nil, from1+1, bounds[1].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
				),
			)
		}
		if fromRangeNum == toRangeNum {
			if from0+1 < to0 {
				alt = newAltNode(
					alt,
					newConcatNode(
						newConcatNode(
							newRangeSymbolNode(nil, from0+1, to0-1, symbolPositionNil),
							newRangeSymbolNode(nil, bounds[1].min, bounds[1].max, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
				)
			}
			if to1 > bounds[1].min {
				alt = newAltNode(
					alt,
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, to0, symbolPositionNil),
							newRangeSymbolNode(nil, bounds[1].min, to1-1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
				)
			}
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, to0, symbolPositionNil),
						newSymbolNode(nil, to1, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[2].min, to2, symbolPositionNil),
				),
			)
			return alt
		}
		for rangeNum := fromRangeNum + 1; rangeNum < toRangeNum; rangeNum++ {
			bounds := bounds3[rangeNum]
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newRangeSymbolNode(nil, bounds[0].min, bounds[0].max, symbolPositionNil),
						newRangeSymbolNode(nil, bounds[1].min, bounds[1].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
				),
			)
		}
		bounds = bounds3[toRangeNum]
		if to0 > bounds[0].min {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newRangeSymbolNode(nil, bounds[0].min, to0-1, symbolPositionNil),
						newRangeSymbolNode(nil, bounds[1].min, bounds[1].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
				),
			)
		}
		if to1 > bounds[1].min {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, to0, symbolPositionNil),
						newRangeSymbolNode(nil, bounds[1].min, to1-1, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
				),
			)
		}
		alt = newAltNode(
			alt,
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, to0, symbolPositionNil),
					newSymbolNode(nil, to1, symbolPositionNil),
				),
				newRangeSymbolNode(nil, bounds[2].min, to2, symbolPositionNil),
			),
		)
		return alt
	}
}

func gen4ByteCharRangeAST(from, to []byte) astNode {
	from0 := from[0]
	from1 := from[1]
	from2 := from[2]
	from3 := from[3]
	to0 := to[0]
	to1 := to[1]
	to2 := to[2]
	to3 := to[3]
	switch {
	case from0 == to0 && from1 == to1 && from2 == to2 && from3 == to3:
		return newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, from0, symbolPositionNil),
					newSymbolNode(nil, from1, symbolPositionNil),
				),
				newSymbolNode(nil, from2, symbolPositionNil),
			),
			newSymbolNode(nil, from3, symbolPositionNil),
		)
	case from0 == to0 && from1 == to1 && from2 == to2:
		return newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, from0, symbolPositionNil),
					newSymbolNode(nil, from1, symbolPositionNil),
				),
				newSymbolNode(nil, from2, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from3, to3, symbolPositionNil),
		)
	case from0 == to0 && from1 == to1:
		rangeNum := get4ByteCharRangeNum(from)
		bounds := bounds4[rangeNum]
		var alt astNode
		alt = newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, from0, symbolPositionNil),
					newSymbolNode(nil, from1, symbolPositionNil),
				),
				newSymbolNode(nil, from2, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from3, bounds[3].max, symbolPositionNil),
		)
		if from2+1 < to2 {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, from0, symbolPositionNil),
							newSymbolNode(nil, from1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, from2+1, to2-1, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		alt = newAltNode(
			alt,
			newConcatNode(
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, from0, symbolPositionNil),
						newSymbolNode(nil, from1, symbolPositionNil),
					),
					newSymbolNode(nil, to2, symbolPositionNil),
				),
				newRangeSymbolNode(nil, bounds[3].min, to3, symbolPositionNil),
			),
		)
		return alt
	case from0 == to0:
		rangeNum := get4ByteCharRangeNum(from)
		bounds := bounds4[rangeNum]
		var alt astNode
		alt = newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, from0, symbolPositionNil),
					newSymbolNode(nil, from1, symbolPositionNil),
				),
				newSymbolNode(nil, from2, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from3, bounds[3].max, symbolPositionNil),
		)
		if from2 < bounds[2].max {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, from0, symbolPositionNil),
							newSymbolNode(nil, from1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, from2+1, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		if from1+1 < to1 {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, from0, symbolPositionNil),
							newRangeSymbolNode(nil, from1+1, to1-1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		if to2 > bounds[2].min {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, from0, symbolPositionNil),
							newSymbolNode(nil, to1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, to2-1, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		alt = newAltNode(
			alt,
			newConcatNode(
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, from0, symbolPositionNil),
						newSymbolNode(nil, to1, symbolPositionNil),
					),
					newSymbolNode(nil, to2, symbolPositionNil),
				),
				newRangeSymbolNode(nil, bounds[3].min, to3, symbolPositionNil),
			),
		)
		return alt
	default:
		fromRangeNum := get4ByteCharRangeNum(from)
		toRangeNum := get4ByteCharRangeNum(to)
		bounds := bounds4[fromRangeNum]
		var alt astNode
		alt = newConcatNode(
			newConcatNode(
				newConcatNode(
					newSymbolNode(nil, from0, symbolPositionNil),
					newSymbolNode(nil, from1, symbolPositionNil),
				),
				newSymbolNode(nil, from2, symbolPositionNil),
			),
			newRangeSymbolNode(nil, from3, bounds[3].max, symbolPositionNil),
		)
		if from2 < bounds[2].max {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, from0, symbolPositionNil),
							newSymbolNode(nil, from1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, from2+1, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		if from1 < bounds[1].max {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, from0, symbolPositionNil),
							newRangeSymbolNode(nil, from1+1, bounds[1].max, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		if fromRangeNum == toRangeNum {
			if from0+1 < to0 {
				alt = newAltNode(
					alt,
					newConcatNode(
						newConcatNode(
							newConcatNode(
								newRangeSymbolNode(nil, from0+1, to0-1, symbolPositionNil),
								newRangeSymbolNode(nil, bounds[1].min, bounds[1].max, symbolPositionNil),
							),
							newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
					),
				)
			}
			if to1 > bounds[1].min {
				alt = newAltNode(
					alt,
					newConcatNode(
						newConcatNode(
							newConcatNode(
								newSymbolNode(nil, to0, symbolPositionNil),
								newRangeSymbolNode(nil, bounds[1].min, to1-1, symbolPositionNil),
							),
							newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
					),
				)
			}
			if to2 > bounds[2].min {
				alt = newAltNode(
					alt,
					newConcatNode(
						newConcatNode(
							newConcatNode(
								newSymbolNode(nil, to0, symbolPositionNil),
								newSymbolNode(nil, to1, symbolPositionNil),
							),
							newRangeSymbolNode(nil, bounds[2].min, to2-1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
					),
				)
			}
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, to0, symbolPositionNil),
							newSymbolNode(nil, to1, symbolPositionNil),
						),
						newSymbolNode(nil, to2, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, to3, symbolPositionNil),
				),
			)
			return alt
		}
		for rangeNum := fromRangeNum + 1; rangeNum < toRangeNum; rangeNum++ {
			bounds := bounds4[rangeNum]
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newRangeSymbolNode(nil, bounds[0].min, bounds[0].max, symbolPositionNil),
							newRangeSymbolNode(nil, bounds[1].min, bounds[1].max, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		bounds = bounds4[toRangeNum]
		if to0 > bounds[0].min {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newRangeSymbolNode(nil, bounds[0].min, to0-1, symbolPositionNil),
							newRangeSymbolNode(nil, bounds[1].min, bounds[1].max, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		if to1 > bounds[1].min {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, to0, symbolPositionNil),
							newRangeSymbolNode(nil, bounds[1].min, to1-1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, bounds[2].max, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		if to2 > bounds[2].min {
			alt = newAltNode(
				alt,
				newConcatNode(
					newConcatNode(
						newConcatNode(
							newSymbolNode(nil, to0, symbolPositionNil),
							newSymbolNode(nil, to1, symbolPositionNil),
						),
						newRangeSymbolNode(nil, bounds[2].min, to2-1, symbolPositionNil),
					),
					newRangeSymbolNode(nil, bounds[3].min, bounds[3].max, symbolPositionNil),
				),
			)
		}
		alt = newAltNode(
			alt,
			newConcatNode(
				newConcatNode(
					newConcatNode(
						newSymbolNode(nil, to0, symbolPositionNil),
						newSymbolNode(nil, to1, symbolPositionNil),
					),
					newSymbolNode(nil, to2, symbolPositionNil),
				),
				newRangeSymbolNode(nil, bounds[3].min, to3, symbolPositionNil),
			),
		)
		return alt
	}
}

func (p *parser) expect(expected tokenKind) {
	if !p.consume(expected) {
		tok := p.peekedTok
		errMsg := fmt.Sprintf("unexpected token; expected: %v, actual: %v", expected, tok.kind)
		raiseSyntaxError(errMsg)
	}
}

func (p *parser) consume(expected tokenKind) bool {
	var tok *token
	var err error
	if p.peekedTok != nil {
		tok = p.peekedTok
		p.peekedTok = nil
	} else {
		tok, err = p.lex.next()
		if err != nil {
			panic(err)
		}
	}
	p.lastTok = tok
	if tok.kind == expected {
		return true
	}
	p.peekedTok = tok
	p.lastTok = nil

	return false
}
