package compiler

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/nihei9/maleeni/spec"
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
	ID      spec.LexModeKindID
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
	endPos2ID   map[symbolPosition]spec.LexModeKindID
}

func genSymbolTable(root astNode) *symbolTable {
	symTab := &symbolTable{
		symPos2Byte: map[symbolPosition]byteRange{},
		endPos2ID:   map[symbolPosition]spec.LexModeKindID{},
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

type patternEntry struct {
	id      spec.LexModeKindID
	pattern []byte
}

func parse(pats []*patternEntry, fragments map[string][]byte) (astNode, *symbolTable, error) {
	if len(pats) == 0 {
		return nil, nil, fmt.Errorf("parse() needs at least one token entry")
	}

	fragmentASTs, err := parseFragments(fragments)
	if err != nil {
		return nil, nil, err
	}
	if fragmentASTs == nil {
		fragmentASTs = map[string]astNode{}
	}

	root, err := parseRegexp(pats, fragmentASTs)
	if err != nil {
		return nil, nil, err
	}

	return root, genSymbolTable(root), nil
}

type incompleteFragment struct {
	kind string
	ast  astNode
}

func parseFragments(fragments map[string][]byte) (map[string]astNode, error) {
	if len(fragments) == 0 {
		return nil, nil
	}
	fragmentASTs := map[string]astNode{}
	incompleteFragments := []*incompleteFragment{}
	var perrs []*ParseError
	for kind, pattern := range fragments {
		p := newParser(bytes.NewReader(pattern))
		ast, err := p.parse()
		if err != nil {
			perrs = append(perrs, &ParseError{
				Pattern: pattern,
				Cause:   err,
				Details: p.errMsgDetails,
			})
			continue
		}
		if p.incomplete {
			incompleteFragments = append(incompleteFragments, &incompleteFragment{
				kind: kind,
				ast:  ast,
			})
		} else {
			fragmentASTs[kind] = ast
		}
	}
	for len(incompleteFragments) > 0 {
		lastIncompCount := len(incompleteFragments)
		remainingFragments := []*incompleteFragment{}
		for _, e := range incompleteFragments {
			remains := applyFragments(e.ast, fragmentASTs)
			if len(remains) > 0 {
				remainingFragments = append(remainingFragments, e)
			} else {
				fragmentASTs[e.kind] = e.ast
			}
		}
		incompleteFragments = remainingFragments
		if len(incompleteFragments) == lastIncompCount {
			for _, e := range incompleteFragments {
				perrs = append(perrs, &ParseError{
					Cause: fmt.Errorf("%v has an undefined fragment or a cycle", e.kind),
				})
			}
			break
		}
	}
	if len(perrs) > 0 {
		return nil, &ParseErrors{
			Errors: perrs,
		}
	}

	return fragmentASTs, nil
}

func parseRegexp(pats []*patternEntry, fragmentASTs map[string]astNode) (astNode, error) {
	symPos := symbolPositionMin
	var root astNode
	var perrs []*ParseError

	for _, pat := range pats {
		if pat.id == spec.LexModeKindIDNil {
			continue
		}

		p := newParser(bytes.NewReader(pat.pattern))
		ast, err := p.parse()
		if err != nil {
			perrs = append(perrs, &ParseError{
				ID:      pat.id,
				Pattern: pat.pattern,
				Cause:   err,
				Details: p.errMsgDetails,
			})
			continue
		}
		remains := applyFragments(ast, fragmentASTs)
		if len(remains) > 0 {
			perrs = append(perrs, &ParseError{
				ID:      pat.id,
				Pattern: pat.pattern,
				Cause:   fmt.Errorf("undefined fragment: %+v", remains),
			})
			continue
		}
		ast = newConcatNode(ast, newEndMarkerNode(pat.id))
		symPos, err = positionSymbols(ast, symPos)
		if err != nil {
			perrs = append(perrs, &ParseError{
				ID:      pat.id,
				Pattern: pat.pattern,
				Cause:   err,
				Details: p.errMsgDetails,
			})
			continue
		}
		root = genAltNode(root, ast)
	}
	if len(perrs) > 0 {
		return nil, &ParseErrors{
			Errors: perrs,
		}
	}

	return root, nil
}

func applyFragments(ast astNode, fragments map[string]astNode) []string {
	if ast == nil {
		return nil
	}
	n, ok := ast.(*fragmentNode)
	if !ok {
		var remains []string
		left, right := ast.children()
		r := applyFragments(left, fragments)
		if len(r) > 0 {
			remains = append(remains, r...)
		}
		r = applyFragments(right, fragments)
		if len(r) > 0 {
			remains = append(remains, r...)
		}
		return remains
	}
	f, ok := fragments[n.symbol]
	if !ok {
		return []string{n.symbol}
	}
	n.left = copyAST(f)
	return nil
}

type parser struct {
	lex           *lexer
	peekedTok     *token
	lastTok       *token
	incomplete    bool
	errMsgDetails string
}

func newParser(src io.Reader) *parser {
	return &parser{
		lex: newLexer(src),
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
	return alt, nil
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
	if p.consume(tokenKindCodePointLeader) {
		return p.parseCodePoint()
	}
	if p.consume(tokenKindCharPropLeader) {
		return p.parseCharProp()
	}
	if p.consume(tokenKindFragmentLeader) {
		return p.parseFragment()
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
	if p.consume(tokenKindCodePointLeader) {
		return p.parseCodePoint()
	}
	if p.consume(tokenKindCharPropLeader) {
		return p.parseCharProp()
	}
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

func (p *parser) parseCodePoint() astNode {
	if !p.consume(tokenKindLBrace) {
		raiseSyntaxError(synErrCPExpInvalidForm)
	}
	if !p.consume(tokenKindCodePoint) {
		raiseSyntaxError(synErrCPExpInvalidForm)
	}

	var cp []byte
	{
		// Although hex.DecodeString method can handle only a hex string that has even length,
		// `codePoint` always has even length by the lexical specification.
		b, err := hex.DecodeString(p.lastTok.codePoint)
		if err != nil {
			panic(fmt.Errorf("failed to decode a code point (%v) into a byte slice: %v", p.lastTok.codePoint, err))
		}
		// `b` must be 4 bytes to convert it into a 32-bit integer.
		l := len(b)
		for i := 0; i < 4-l; i++ {
			b = append([]byte{0}, b...)
		}
		n := binary.BigEndian.Uint32(b)
		if n < 0x0000 || n > 0x10FFFF {
			raiseSyntaxError(synErrCPExpOutOfRange)
		}

		cp = []byte(string(rune(n)))
	}

	var concat astNode
	{
		concat = newSymbolNode(cp[0])
		for _, b := range cp[1:] {
			concat = genConcatNode(
				concat,
				newSymbolNode(b),
			)
		}
	}

	if !p.consume(tokenKindRBrace) {
		raiseSyntaxError(synErrCPExpInvalidForm)
	}

	return concat
}

func (p *parser) parseCharProp() astNode {
	if !p.consume(tokenKindLBrace) {
		raiseSyntaxError(synErrCharPropExpInvalidForm)
	}
	var sym1, sym2 string
	if !p.consume(tokenKindCharPropSymbol) {
		raiseSyntaxError(synErrCharPropExpInvalidForm)
	}
	sym1 = p.lastTok.propSymbol
	if p.consume(tokenKindEqual) {
		if !p.consume(tokenKindCharPropSymbol) {
			raiseSyntaxError(synErrCharPropExpInvalidForm)
		}
		sym2 = p.lastTok.propSymbol
	}

	var alt astNode
	var propName, propVal string
	if sym2 != "" {
		propName = sym1
		propVal = sym2
	} else {
		propName = "gc"
		propVal = sym1
	}
	pat, err := normalizeCharacterProperty(propName, propVal)
	if err != nil {
		p.errMsgDetails = fmt.Sprintf("%v", err)
		raiseSyntaxError(synErrCharPropUnsupported)
	}
	if pat != "" {
		p := newParser(bytes.NewReader([]byte(pat)))
		ast, err := p.parse()
		if err != nil {
			panic(err)
		}
		alt = ast
	} else {
		cpRanges, inverse, err := findCodePointRanges(propName, propVal)
		if err != nil {
			p.errMsgDetails = fmt.Sprintf("%v", err)
			raiseSyntaxError(synErrCharPropUnsupported)
		}
		if inverse {
			r := cpRanges[0]
			from := genNormalCharAST(r.From)
			to := genNormalCharAST(r.To)
			alt = exclude(genRangeAST(from, to), genAnyCharAST())
			if alt == nil {
				panic(fmt.Errorf("a pattern that isn't matching any symbols"))
			}
			for _, r := range cpRanges[1:] {
				from := genNormalCharAST(r.From)
				to := genNormalCharAST(r.To)
				alt = exclude(genRangeAST(from, to), alt)
				if alt == nil {
					panic(fmt.Errorf("a pattern that isn't matching any symbols"))
				}
			}
		} else {
			for _, r := range cpRanges {
				from := genNormalCharAST(r.From)
				to := genNormalCharAST(r.To)
				alt = genAltNode(
					alt,
					genRangeAST(from, to),
				)
			}
		}
	}

	if !p.consume(tokenKindRBrace) {
		raiseSyntaxError(synErrCharPropExpInvalidForm)
	}

	return alt
}

func (p *parser) parseFragment() astNode {
	if !p.consume(tokenKindLBrace) {
		raiseSyntaxError(synErrFragmentExpInvalidForm)
	}
	if !p.consume(tokenKindFragmentSymbol) {
		raiseSyntaxError(synErrFragmentExpInvalidForm)
	}
	sym := p.lastTok.fragmentSymbol

	if !p.consume(tokenKindRBrace) {
		raiseSyntaxError(synErrFragmentExpInvalidForm)
	}

	p.incomplete = true

	return newFragmentNode(sym, nil)
}

func (p *parser) parseNormalChar() astNode {
	if !p.consume(tokenKindChar) {
		return nil
	}
	return genNormalCharAST(p.lastTok.char)
}

func genNormalCharAST(c rune) astNode {
	b := []byte(string(c))
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
