package compiler

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type ParseErrors struct {
	Errors []*ParseError
}

func (e *ParseErrors) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%v", e.Errors[0])
	for _, err := range e.Errors[1:] {
		fmt.Fprintf(&b, "\n%v", err)
	}
	return b.String()
}

type ParseError struct {
	ID      int
	Pattern []byte
	Cause   error
	Details string
}

func (e *ParseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "#%v %v: %v", e.ID, string(e.Pattern), e.Cause)
	if e.Details != "" {
		fmt.Fprintf(&b, ": %v", e.Details)
	}
	return b.String()
}

func raiseSyntaxError(synErr *SyntaxError) {
	panic(synErr)
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
	symPos := symbolPositionMin
	var root astNode
	var perrs []*ParseError
	for id, pattern := range regexps {
		p := newParser(id, symPos, bytes.NewReader(pattern))
		ast, err := p.parse()
		if err != nil {
			perr := &ParseError{
				ID:      id,
				Pattern: pattern,
				Cause:   err,
				Details: p.errMsgDetails,
			}
			if p.errMsgDetails != "" {
				perr.Details = p.errMsgDetails
			}
			perrs = append(perrs, perr)
			continue
		}
		if root == nil {
			root = ast
		} else {
			root = newAltNode(root, ast)
		}
		symPos = p.symPos
	}
	if len(perrs) > 0 {
		return nil, nil, &ParseErrors{
			Errors: perrs,
		}
	}

	return root, genSymbolTable(root), nil
}

type parser struct {
	id            int
	lex           *lexer
	peekedTok     *token
	lastTok       *token
	errMsgDetails string
	symPos        uint16
}

func newParser(id int, initialSymPos uint16, src io.Reader) *parser {
	return &parser{
		id:            id,
		lex:           newLexer(src),
		peekedTok:     nil,
		lastTok:       nil,
		errMsgDetails: "",
		symPos:        initialSymPos,
	}
}

func (p *parser) parse() (ast astNode, retErr error) {
	defer func() {
		err := recover()
		if err != nil {
			var ok bool
			retErr, ok = err.(error)
			if !ok {
				retErr = fmt.Errorf("%v", err)
			}
			return
		}
	}()

	ast, err := p.parseRegexp()
	if err != nil {
		return nil, err
	}
	p.symPos, err = positionSymbols(ast, p.symPos)
	if err != nil {
		return nil, err
	}
	return ast, nil
}

func (p *parser) parseRegexp() (astNode, error) {
	alt := p.parseAlt()
	if alt == nil {
		if p.consume(tokenKindGroupClose) {
			raiseSyntaxError(synErrGroupNoInitiator)
		}
		raiseSyntaxError(synErrNullPattern)
	}
	if p.consume(tokenKindGroupClose) {
		raiseSyntaxError(synErrGroupNoInitiator)
	}
	p.expect(tokenKindEOF)
	return newConcatNode(alt, newEndMarkerNode(p.id)), nil
}

func (p *parser) parseAlt() astNode {
	left := p.parseConcat()
	if left == nil {
		if p.consume(tokenKindAlt) {
			raiseSyntaxError(synErrAltLackOfOperand)
		}
		return nil
	}
	for {
		if !p.consume(tokenKindAlt) {
			break
		}
		right := p.parseConcat()
		if right == nil {
			raiseSyntaxError(synErrAltLackOfOperand)
		}
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
	if group == nil {
		if p.consume(tokenKindRepeat) {
			p.errMsgDetails = "* needs an operand"
			raiseSyntaxError(synErrRepNoTarget)
		}
		if p.consume(tokenKindRepeatOneOrMore) {
			p.errMsgDetails = "+ needs an operand"
			raiseSyntaxError(synErrRepNoTarget)
		}
		if p.consume(tokenKindOption) {
			p.errMsgDetails = "? needs an operand"
			raiseSyntaxError(synErrRepNoTarget)
		}
		return nil
	}
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
		alt := p.parseAlt()
		if alt == nil {
			if p.consume(tokenKindEOF) {
				raiseSyntaxError(synErrGroupUnclosed)
			}
			raiseSyntaxError(synErrGroupNoElem)
		}
		if p.consume(tokenKindEOF) {
			raiseSyntaxError(synErrGroupUnclosed)
		}
		if !p.consume(tokenKindGroupClose) {
			raiseSyntaxError(synErrGroupInvalidForm)
		}
		return alt
	}
	return p.parseSingleChar()
}

func (p *parser) parseSingleChar() astNode {
	if p.consume(tokenKindAnyChar) {
		return genAnyCharAST()
	}
	if p.consume(tokenKindBExpOpen) {
		left := p.parseBExpElem()
		if left == nil {
			if p.consume(tokenKindEOF) {
				raiseSyntaxError(synErrBExpUnclosed)
			}
			raiseSyntaxError(synErrBExpNoElem)
		}
		for {
			right := p.parseBExpElem()
			if right == nil {
				break
			}
			left = newAltNode(left, right)
		}
		if p.consume(tokenKindEOF) {
			raiseSyntaxError(synErrBExpUnclosed)
		}
		p.expect(tokenKindBExpClose)
		return left
	}
	if p.consume(tokenKindInverseBExpOpen) {
		elem := p.parseBExpElem()
		if elem == nil {
			if p.consume(tokenKindEOF) {
				raiseSyntaxError(synErrBExpUnclosed)
			}
			raiseSyntaxError(synErrBExpNoElem)
		}
		inverse := exclude(elem, genAnyCharAST())
		if inverse == nil {
			panic(fmt.Errorf("a pattern that isn't matching any symbols"))
		}
		for {
			elem := p.parseBExpElem()
			if elem == nil {
				break
			}
			inverse = exclude(elem, inverse)
			if inverse == nil {
				panic(fmt.Errorf("a pattern that isn't matching any symbols"))
			}
		}
		if p.consume(tokenKindEOF) {
			raiseSyntaxError(synErrBExpUnclosed)
		}
		p.expect(tokenKindBExpClose)
		return inverse
	}
	c := p.parseNormalChar()
	if c == nil {
		if p.consume(tokenKindBExpClose) {
			raiseSyntaxError(synErrBExpInvalidForm)
		}
		return nil
	}
	return c
}

func (p *parser) parseBExpElem() astNode {
	left := p.parseNormalChar()
	if left == nil {
		return nil
	}
	if !p.consume(tokenKindCharRange) {
		return left
	}
	right := p.parseNormalChar()
	if right == nil {
		panic(fmt.Errorf("invalid range expression"))
	}
	from := genByteSeq(left)
	to := genByteSeq(right)
	if !isValidOrder(from, to) {
		p.errMsgDetails = fmt.Sprintf("[%s-%s] ([%v-%v])", string(from), string(to), from, to)
		raiseSyntaxError(synErrRangeInvalidOrder)
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
		return newSymbolNode(b[0])
	case 2:
		return genConcatNode(
			newSymbolNode(b[0]),
			newSymbolNode(b[1]),
		)
	case 3:
		return genConcatNode(
			newSymbolNode(b[0]),
			newSymbolNode(b[1]),
			newSymbolNode(b[2]),
		)
	default: // is equivalent to case 4
		return genConcatNode(
			newSymbolNode(b[0]),
			newSymbolNode(b[1]),
			newSymbolNode(b[2]),
			newSymbolNode(b[3]),
		)
	}
}

func exclude(symbol, base astNode) astNode {
	switch base.(type) {
	case *altNode:
		left, right := base.children()
		return genAltNode(
			exclude(symbol, left),
			exclude(symbol, right),
		)
	case *concatNode:
		baseSeq := genByteRangeSeq(base)
		symSeq := genByteRangeSeq(symbol)
		excluded := excludeByteRangeSequence(symSeq, baseSeq)
		if len(excluded) <= 0 {
			return nil
		}
		return convertByteRangeSeqsToAST(excluded)
	case *symbolNode:
		baseSeq := genByteRangeSeq(base)
		symSeq := genByteRangeSeq(symbol)
		excluded := excludeByteRangeSequence(symSeq, baseSeq)
		if len(excluded) <= 0 {
			return nil
		}
		return convertByteRangeSeqsToAST(excluded)
	}
	return nil
}

func convertByteRangeSeqsToAST(seqs [][]byteRange) astNode {
	concats := []astNode{}
	for _, seq := range seqs {
		syms := []astNode{}
		for _, elem := range seq {
			syms = append(syms, newRangeSymbolNode(elem.from, elem.to))
		}
		concats = append(concats, genConcatNode(syms...))
	}
	return genAltNode(concats...)
}

// Refelences:
// * https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf#G7404
//   * Table 3-6.  UTF-8 Bit Distribution
//   * Table 3-7.  Well-Formed UTF-8 Byte Sequences
func genAnyCharAST() astNode {
	return genAltNode(
		// 1 byte character <00..7F>
		newRangeSymbolNode(0x00, 0x7f),
		// 2 bytes character <C2..DF 80..BF>
		genConcatNode(
			newRangeSymbolNode(0xc2, 0xdf),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 3 bytes character <E0 A0..BF 80..BF>
		genConcatNode(
			newSymbolNode(0xe0),
			newRangeSymbolNode(0xa0, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 3 bytes character <E1..EC 80..BF 80..BF>
		genConcatNode(
			newRangeSymbolNode(0xe1, 0xec),
			newRangeSymbolNode(0x80, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 3 bytes character <ED 80..9F 80..BF>
		genConcatNode(
			newSymbolNode(0xed),
			newRangeSymbolNode(0x80, 0x9f),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 3 bytes character <EE..EF 80..BF 80..BF>
		genConcatNode(
			newRangeSymbolNode(0xee, 0xef),
			newRangeSymbolNode(0x80, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 4 bytes character <F0 90..BF 80..BF 80..BF>
		genConcatNode(
			newSymbolNode(0xf0),
			newRangeSymbolNode(0x90, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 4 bytes character <F1..F3 80..BF 80..BF 80..BF>
		genConcatNode(
			newRangeSymbolNode(0xf1, 0xf3),
			newRangeSymbolNode(0x80, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
		),
		// 4 bytes character <F4 80..8F 80..BF 80..BF>
		genConcatNode(
			newSymbolNode(0xf4),
			newRangeSymbolNode(0x80, 0x8f),
			newRangeSymbolNode(0x80, 0xbf),
			newRangeSymbolNode(0x80, 0xbf),
		),
	)
}

func genRangeAST(fromNode, toNode astNode) astNode {
	from := genByteSeq(fromNode)
	to := genByteSeq(toNode)
	switch len(from) {
	case 1:
		switch len(to) {
		case 1:
			return gen1ByteCharRangeAST(from, to)
		case 2:
			return genAltNode(
				gen1ByteCharRangeAST(from, []byte{0x7f}),
				gen2ByteCharRangeAST([]byte{0xc2, 0x80}, to),
			)
		case 3:
			return genAltNode(
				gen1ByteCharRangeAST(from, []byte{0x7f}),
				gen2ByteCharRangeAST([]byte{0xc2, 0x80}, []byte{0xdf, 0xbf}),
				gen3ByteCharRangeAST([]byte{0xe0, 0xa0, 0x80}, to),
			)
		case 4:
			return genAltNode(
				gen1ByteCharRangeAST(from, []byte{0x7f}),
				gen2ByteCharRangeAST([]byte{0xc2, 0x80}, []byte{0xdf, 0xbf}),
				gen3ByteCharRangeAST([]byte{0xe0, 0xa0, 0x80}, []byte{0xef, 0xbf, 0xbf}),
				gen4ByteCharRangeAST([]byte{0xf0, 0x90, 0x80}, to),
			)
		}
	case 2:
		switch len(to) {
		case 2:
			return gen2ByteCharRangeAST(from, to)
		case 3:
			return genAltNode(
				gen2ByteCharRangeAST(from, []byte{0xdf, 0xbf}),
				gen3ByteCharRangeAST([]byte{0xc2, 0x80}, to),
			)
		case 4:
			return genAltNode(
				gen2ByteCharRangeAST(from, []byte{0xdf, 0xbf}),
				gen3ByteCharRangeAST([]byte{0xc2, 0x80}, []byte{0xef, 0xbf, 0xbf}),
				gen4ByteCharRangeAST([]byte{0xf0, 0x90, 0x80}, to),
			)
		}
	case 3:
		switch len(to) {
		case 3:
			return gen3ByteCharRangeAST(from, to)
		case 4:
			return genAltNode(
				gen3ByteCharRangeAST(from, []byte{0xef, 0xbf, 0xbf}),
				gen4ByteCharRangeAST([]byte{0xf0, 0x90, 0x80}, to),
			)
		}
	case 4:
		return gen4ByteCharRangeAST(from, to)
	}
	panic(fmt.Errorf("invalid range; from: %v, to: %v", from, to))
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
	panic(fmt.Errorf("genByteSeq() cannot handle %T: %v", node, node))
}

func genByteRangeSeq(node astNode) []byteRange {
	switch n := node.(type) {
	case *symbolNode:
		return []byteRange{{from: n.from, to: n.to}}
	case *concatNode:
		seq := genByteRangeSeq(n.left)
		seq = append(seq, genByteRangeSeq(n.right)...)
		return seq
	}
	panic(fmt.Errorf("genByteRangeSeq() cannot handle %T: %v", node, node))
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

type byteBoundsEntry struct {
	min byte
	max byte
}

var (
	bounds1 = [][]byteBoundsEntry{
		nil,
		{{min: 0x00, max: 0x7f}},
	}

	bounds2 = [][]byteBoundsEntry{
		nil,
		{{min: 0xc2, max: 0xdf}, {min: 0x80, max: 0xbf}},
	}

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

func gen1ByteCharRangeAST(from, to []byte) astNode {
	return newRangeSymbolNode(from[0], to[0])
}

func gen2ByteCharRangeAST(from, to []byte) astNode {
	from0 := from[0]
	from1 := from[1]
	to0 := to[0]
	to1 := to[1]
	switch {
	case from0 == to0 && from1 == to1:
		return genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
		)
	case from0 == to0:
		return genConcatNode(
			newSymbolNode(from0),
			newRangeSymbolNode(from1, to1),
		)
	default:
		alt1 := genConcatNode(
			newSymbolNode(from0),
			newRangeSymbolNode(from1, 0xbf),
		)
		alt2 := genConcatNode(
			newRangeSymbolNode(from0+1, to0),
			newRangeSymbolNode(0x80, to1),
		)
		return newAltNode(alt1, alt2)
	}
}

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
		return genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newSymbolNode(from2),
		)
	case from0 == to0 && from1 == to1:
		return genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newRangeSymbolNode(from2, to2),
		)
	case from0 == to0:
		rangeNum := get3ByteCharRangeNum(from)
		bounds := bounds3[rangeNum]
		var alt astNode
		alt = genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newRangeSymbolNode(from2, bounds[2].max),
		)
		if from1+1 < to1 {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newRangeSymbolNode(from1+1, to1-1),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
				),
			)
		}
		alt = genAltNode(
			alt,
			genConcatNode(
				newSymbolNode(from0),
				newSymbolNode(to1),
				newRangeSymbolNode(bounds[2].min, to2),
			),
		)
		return alt
	default:
		fromRangeNum := get3ByteCharRangeNum(from)
		toRangeNum := get3ByteCharRangeNum(to)
		bounds := bounds3[fromRangeNum]
		var alt astNode
		alt = genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newRangeSymbolNode(from2, bounds[2].max),
		)
		if from1 < bounds[1].max {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newRangeSymbolNode(from1+1, bounds[1].max),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
				),
			)
		}
		if fromRangeNum == toRangeNum {
			if from0+1 < to0 {
				alt = genAltNode(
					alt,
					genConcatNode(
						newRangeSymbolNode(from0+1, to0-1),
						newRangeSymbolNode(bounds[1].min, bounds[1].max),
						newRangeSymbolNode(bounds[2].min, bounds[2].max),
					),
				)
			}
			if to1 > bounds[1].min {
				alt = genAltNode(
					alt,
					genConcatNode(
						newSymbolNode(to0),
						newRangeSymbolNode(bounds[1].min, to1-1),
						newRangeSymbolNode(bounds[2].min, bounds[2].max),
					),
				)
			}
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(to0),
					newSymbolNode(to1),
					newRangeSymbolNode(bounds[2].min, to2),
				),
			)
			return alt
		}
		for rangeNum := fromRangeNum + 1; rangeNum < toRangeNum; rangeNum++ {
			bounds := bounds3[rangeNum]
			alt = genAltNode(
				alt,
				genConcatNode(
					newRangeSymbolNode(bounds[0].min, bounds[0].max),
					newRangeSymbolNode(bounds[1].min, bounds[1].max),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
				),
			)
		}
		bounds = bounds3[toRangeNum]
		if to0 > bounds[0].min {
			alt = genAltNode(
				alt,
				genConcatNode(
					newRangeSymbolNode(bounds[0].min, to0-1),
					newRangeSymbolNode(bounds[1].min, bounds[1].max),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
				),
			)
		}
		if to1 > bounds[1].min {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(to0),
					newRangeSymbolNode(bounds[1].min, to1-1),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
				),
			)
		}
		alt = genAltNode(
			alt,
			genConcatNode(
				newSymbolNode(to0),
				newSymbolNode(to1),
				newRangeSymbolNode(bounds[2].min, to2),
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
		return genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newSymbolNode(from2),
			newSymbolNode(from3),
		)
	case from0 == to0 && from1 == to1 && from2 == to2:
		return genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newSymbolNode(from2),
			newRangeSymbolNode(from3, to3),
		)
	case from0 == to0 && from1 == to1:
		rangeNum := get4ByteCharRangeNum(from)
		bounds := bounds4[rangeNum]
		var alt astNode
		alt = genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newSymbolNode(from2),
			newRangeSymbolNode(from3, bounds[3].max),
		)
		if from2+1 < to2 {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newSymbolNode(from1),
					newRangeSymbolNode(from2+1, to2-1),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		alt = genAltNode(
			alt,
			genConcatNode(
				newSymbolNode(from0),
				newSymbolNode(from1),
				newSymbolNode(to2),
				newRangeSymbolNode(bounds[3].min, to3),
			),
		)
		return alt
	case from0 == to0:
		rangeNum := get4ByteCharRangeNum(from)
		bounds := bounds4[rangeNum]
		var alt astNode
		alt = genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newSymbolNode(from2),
			newRangeSymbolNode(from3, bounds[3].max),
		)
		if from2 < bounds[2].max {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newSymbolNode(from1),
					newRangeSymbolNode(from2+1, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		if from1+1 < to1 {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newRangeSymbolNode(from1+1, to1-1),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		if to2 > bounds[2].min {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newSymbolNode(to1),
					newRangeSymbolNode(bounds[2].min, to2-1),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		alt = genAltNode(
			alt,
			genConcatNode(
				newSymbolNode(from0),
				newSymbolNode(to1),
				newSymbolNode(to2),
				newRangeSymbolNode(bounds[3].min, to3),
			),
		)
		return alt
	default:
		fromRangeNum := get4ByteCharRangeNum(from)
		toRangeNum := get4ByteCharRangeNum(to)
		bounds := bounds4[fromRangeNum]
		var alt astNode
		alt = genConcatNode(
			newSymbolNode(from0),
			newSymbolNode(from1),
			newSymbolNode(from2),
			newRangeSymbolNode(from3, bounds[3].max),
		)
		if from2 < bounds[2].max {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newSymbolNode(from1),
					newRangeSymbolNode(from2+1, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		if from1 < bounds[1].max {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(from0),
					newRangeSymbolNode(from1+1, bounds[1].max),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		if fromRangeNum == toRangeNum {
			if from0+1 < to0 {
				alt = genAltNode(
					alt,
					genConcatNode(
						newRangeSymbolNode(from0+1, to0-1),
						newRangeSymbolNode(bounds[1].min, bounds[1].max),
						newRangeSymbolNode(bounds[2].min, bounds[2].max),
						newRangeSymbolNode(bounds[3].min, bounds[3].max),
					),
				)
			}
			if to1 > bounds[1].min {
				alt = genAltNode(
					alt,
					genConcatNode(
						newSymbolNode(to0),
						newRangeSymbolNode(bounds[1].min, to1-1),
						newRangeSymbolNode(bounds[2].min, bounds[2].max),
						newRangeSymbolNode(bounds[3].min, bounds[3].max),
					),
				)
			}
			if to2 > bounds[2].min {
				alt = genAltNode(
					alt,
					genConcatNode(
						newSymbolNode(to0),
						newSymbolNode(to1),
						newRangeSymbolNode(bounds[2].min, to2-1),
						newRangeSymbolNode(bounds[3].min, bounds[3].max),
					),
				)
			}
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(to0),
					newSymbolNode(to1),
					newSymbolNode(to2),
					newRangeSymbolNode(bounds[3].min, to3),
				),
			)
			return alt
		}
		for rangeNum := fromRangeNum + 1; rangeNum < toRangeNum; rangeNum++ {
			bounds := bounds4[rangeNum]
			alt = genAltNode(
				alt,
				genConcatNode(
					newRangeSymbolNode(bounds[0].min, bounds[0].max),
					newRangeSymbolNode(bounds[1].min, bounds[1].max),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		bounds = bounds4[toRangeNum]
		if to0 > bounds[0].min {
			alt = genAltNode(
				alt,
				genConcatNode(
					newRangeSymbolNode(bounds[0].min, to0-1),
					newRangeSymbolNode(bounds[1].min, bounds[1].max),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		if to1 > bounds[1].min {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(to0),
					newRangeSymbolNode(bounds[1].min, to1-1),
					newRangeSymbolNode(bounds[2].min, bounds[2].max),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		if to2 > bounds[2].min {
			alt = genAltNode(
				alt,
				genConcatNode(
					newSymbolNode(to0),
					newSymbolNode(to1),
					newRangeSymbolNode(bounds[2].min, to2-1),
					newRangeSymbolNode(bounds[3].min, bounds[3].max),
				),
			)
		}
		alt = genAltNode(
			alt,
			genConcatNode(
				newSymbolNode(to0),
				newSymbolNode(to1),
				newSymbolNode(to2),
				newRangeSymbolNode(bounds[3].min, to3),
			),
		)
		return alt
	}
}

func genConcatNode(cs ...astNode) astNode {
	if len(cs) <= 0 {
		return nil
	}
	if len(cs) == 1 {
		return cs[0]
	}
	concat := newConcatNode(cs[0], cs[1])
	for _, c := range cs[2:] {
		concat = newConcatNode(concat, c)
	}
	return concat
}

func genAltNode(cs ...astNode) astNode {
	nonNilNodes := []astNode{}
	for _, c := range cs {
		if c == nil {
			continue
		}
		nonNilNodes = append(nonNilNodes, c)
	}
	if len(nonNilNodes) <= 0 {
		return nil
	}
	if len(nonNilNodes) == 1 {
		return nonNilNodes[0]
	}
	alt := newAltNode(nonNilNodes[0], nonNilNodes[1])
	for _, c := range nonNilNodes[2:] {
		alt = newAltNode(alt, c)
	}
	return alt
}

func (p *parser) expect(expected tokenKind) {
	if !p.consume(expected) {
		tok := p.peekedTok
		p.errMsgDetails = fmt.Sprintf("unexpected token; expected: %v, actual: %v", expected, tok.kind)
		raiseSyntaxError(synErrUnexpectedToken)
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
			p.errMsgDetails = p.lex.errMsgDetails
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
