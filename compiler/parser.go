package compiler

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/nihei9/maleeni/spec"
	"github.com/nihei9/maleeni/ucd"
	"github.com/nihei9/maleeni/utf8"
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

	// If and only if isContributoryPropertyExposed is true, the parser interprets contributory properties that
	// appear in property expressions.
	//
	// The contributory properties are not exposed, and users cannot use those properties because the parser
	// follows [UAX #44 5.13 Property APIs]. For instance, \p{Other_Alphabetic} is invalid.
	//
	// isContributoryPropertyExposed is set to true when the parser is generated recursively. The parser needs to
	// interpret derived properties internally because the derived properties consist of other properties that
	// may contain the contributory properties.
	//
	// [UAX #44 5.13 Property APIs] says:
	// > The following subtypes of Unicode character properties should generally not be exposed in APIs,
	// > except in limited circumstances. They may not be useful, particularly in public API collections,
	// > and may instead prove misleading to the users of such API collections.
	// >   * Contributory properties are not recommended for public APIs.
	// > ...
	// https://unicode.org/reports/tr44/#Property_APIs
	isContributoryPropertyExposed bool
}

func newParser(src io.Reader) *parser {
	return &parser{
		lex:                           newLexer(src),
		isContributoryPropertyExposed: false,
	}
}

func (p *parser) exposeContributoryProperty() {
	p.isContributoryPropertyExposed = true
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
		propName = ""
		propVal = sym1
	}
	if !p.isContributoryPropertyExposed && ucd.IsContributoryProperty(propName) {
		p.errMsgDetails = propName
		raiseSyntaxError(synErrCharPropUnsupported)
	}
	pat, err := ucd.NormalizeCharacterProperty(propName, propVal)
	if err != nil {
		p.errMsgDetails = fmt.Sprintf("%v", err)
		raiseSyntaxError(synErrCharPropUnsupported)
	}
	if pat != "" {
		p := newParser(bytes.NewReader([]byte(pat)))
		p.exposeContributoryProperty()
		ast, err := p.parse()
		if err != nil {
			panic(err)
		}
		alt = ast
	} else {
		cpRanges, inverse, err := ucd.FindCodePointRanges(propName, propVal)
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
	if alt, ok := symbol.(*altNode); ok {
		return exclude(alt.right, exclude(alt.left, base))
	}

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

func genAnyCharAST() astNode {
	return convertCharBlocksToAST(utf8.AllCharBlocks())
}

func genRangeAST(fromNode, toNode astNode) astNode {
	from := genByteSeq(fromNode)
	to := genByteSeq(toNode)
	blks, err := utf8.GenCharBlocks(from, to)
	if err != nil {
		panic(err)
	}
	return convertCharBlocksToAST(blks)
}

func convertCharBlocksToAST(blks []*utf8.CharBlock) astNode {
	var alt astNode
	for _, blk := range blks {
		r := make([]astNode, len(blk.From))
		for i := 0; i < len(blk.From); i++ {
			r[i] = newRangeSymbolNode(blk.From[i], blk.To[i])
		}
		alt = genAltNode(alt, genConcatNode(r...))
	}
	return alt
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
