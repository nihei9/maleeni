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

func raiseSyntaxError(message string) {
	panic(&SyntaxError{
		message: message,
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
	if !p.consume(tokenKindRepeat) {
		return group
	}
	return newRepeatNode(group)
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
		left := p.parseNormalChar()
		if left == nil {
			raiseSyntaxError("bracket expression must include at least one character")
		}
		for {
			right := p.parseNormalChar()
			if right == nil {
				break
			}
			left = newAltNode(left, right)
		}
		return left
	}
	return p.parseNormalChar()
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
