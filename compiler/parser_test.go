package compiler

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/nihei9/maleeni/spec"
	"github.com/nihei9/maleeni/ucd"
)

func symPos(n uint16) symbolPosition {
	pos, err := newSymbolPosition(n, false)
	if err != nil {
		panic(err)
	}
	return pos
}

func endPos(n uint16) symbolPosition {
	pos, err := newSymbolPosition(n, true)
	if err != nil {
		panic(err)
	}
	return pos
}

func TestParse(t *testing.T) {
	tests := []struct {
		pattern     string
		fragments   map[string]string
		ast         astNode
		syntaxError *SyntaxError

		// When an AST is large, as patterns containing a character property expression,
		// this test only checks that the pattern is parsable.
		// The check of the validity of such AST is performed by checking that it can be matched correctly using the driver.
		skipTestAST bool
	}{
		{
			pattern: "a",
			ast: genConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "abc",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('b'), symPos(2)),
					newSymbolNodeWithPos(byte('c'), symPos(3)),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "a?",
			ast: genConcatNode(
				newOptionNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "[abc]?",
			ast: genConcatNode(
				newOptionNode(
					genAltNode(
						newSymbolNodeWithPos(byte('a'), symPos(1)),
						newSymbolNodeWithPos(byte('b'), symPos(2)),
						newSymbolNodeWithPos(byte('c'), symPos(3)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "\\u{3042}?",
			ast: genConcatNode(
				newOptionNode(
					genConcatNode(
						newSymbolNodeWithPos(0xE3, symPos(1)),
						newSymbolNodeWithPos(0x81, symPos(2)),
						newSymbolNodeWithPos(0x82, symPos(3)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern:     "\\p{Letter}?",
			skipTestAST: true,
		},
		{
			pattern: "\\f{a2c}?",
			fragments: map[string]string{
				"a2c": "abc",
			},
			ast: genConcatNode(
				newOptionNode(
					newFragmentNode("a2c",
						genConcatNode(
							newSymbolNodeWithPos(byte('a'), symPos(1)),
							newSymbolNodeWithPos(byte('b'), symPos(2)),
							newSymbolNodeWithPos(byte('c'), symPos(3)),
						),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "(a)?",
			ast: genConcatNode(
				newOptionNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "((a?)?)?",
			ast: genConcatNode(
				newOptionNode(
					newOptionNode(
						newOptionNode(
							newSymbolNodeWithPos(byte('a'), symPos(1)),
						),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "(abc)?",
			ast: genConcatNode(
				newOptionNode(
					genConcatNode(
						newSymbolNodeWithPos(byte('a'), symPos(1)),
						newSymbolNodeWithPos(byte('b'), symPos(2)),
						newSymbolNodeWithPos(byte('c'), symPos(3)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "(a|b)?",
			ast: genConcatNode(
				newOptionNode(
					genAltNode(
						newSymbolNodeWithPos(byte('a'), symPos(1)),
						newSymbolNodeWithPos(byte('b'), symPos(2)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern:     "?",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "(?)",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "a|?",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "?|b",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "a??",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern: "a*",
			ast: genConcatNode(
				newRepeatNode(
					newSymbolNodeWithPos(byte('a'), 1),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "[abc]*",
			ast: genConcatNode(
				newRepeatNode(
					genAltNode(
						newSymbolNodeWithPos(byte('a'), 1),
						newSymbolNodeWithPos(byte('b'), 2),
						newSymbolNodeWithPos(byte('c'), 3),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "\\u{3042}*",
			ast: genConcatNode(
				newRepeatNode(
					genConcatNode(
						newSymbolNodeWithPos(0xE3, symPos(1)),
						newSymbolNodeWithPos(0x81, symPos(2)),
						newSymbolNodeWithPos(0x82, symPos(3)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern:     "\\p{Letter}*",
			skipTestAST: true,
		},
		{
			pattern: "\\f{a2c}*",
			fragments: map[string]string{
				"a2c": "abc",
			},
			ast: genConcatNode(
				newRepeatNode(
					newFragmentNode("a2c",
						genConcatNode(
							newSymbolNodeWithPos(byte('a'), symPos(1)),
							newSymbolNodeWithPos(byte('b'), symPos(2)),
							newSymbolNodeWithPos(byte('c'), symPos(3)),
						),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "((a*)*)*",
			ast: genConcatNode(
				newRepeatNode(
					newRepeatNode(
						newRepeatNode(
							newSymbolNodeWithPos(byte('a'), 1),
						),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "(abc)*",
			ast: genConcatNode(
				newRepeatNode(
					genConcatNode(
						newSymbolNodeWithPos(byte('a'), 1),
						newSymbolNodeWithPos(byte('b'), 2),
						newSymbolNodeWithPos(byte('c'), 3),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "(a|b)*",
			ast: genConcatNode(
				newRepeatNode(
					genAltNode(
						newSymbolNodeWithPos(byte('a'), 1),
						newSymbolNodeWithPos(byte('b'), 2),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern:     "*",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "(*)",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "a|*",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "*|b",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "a**",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern: "a+",
			ast: genConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newRepeatNode(
					newSymbolNodeWithPos(byte('a'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern: "[abc]+",
			ast: genConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('b'), symPos(2)),
					newSymbolNodeWithPos(byte('c'), symPos(3)),
				),
				newRepeatNode(
					genAltNode(
						newSymbolNodeWithPos(byte('a'), symPos(4)),
						newSymbolNodeWithPos(byte('b'), symPos(5)),
						newSymbolNodeWithPos(byte('c'), symPos(6)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(7)),
			),
		},
		{
			pattern: "\\u{3042}+",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(0xE3, symPos(1)),
					newSymbolNodeWithPos(0x81, symPos(2)),
					newSymbolNodeWithPos(0x82, symPos(3)),
				),
				newRepeatNode(
					genConcatNode(
						newSymbolNodeWithPos(0xE3, symPos(4)),
						newSymbolNodeWithPos(0x81, symPos(5)),
						newSymbolNodeWithPos(0x82, symPos(6)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(7)),
			),
		},
		{
			pattern:     "\\p{Letter}+",
			skipTestAST: true,
		},
		{
			pattern: "\\f{a2c}+",
			fragments: map[string]string{
				"a2c": "abc",
			},
			ast: genConcatNode(
				newFragmentNode("a2c",
					genConcatNode(
						newSymbolNodeWithPos(byte('a'), symPos(1)),
						newSymbolNodeWithPos(byte('b'), symPos(2)),
						newSymbolNodeWithPos(byte('c'), symPos(3)),
					),
				),
				newRepeatNode(
					newFragmentNode("a2c",
						genConcatNode(
							newSymbolNodeWithPos(byte('a'), symPos(4)),
							newSymbolNodeWithPos(byte('b'), symPos(5)),
							newSymbolNodeWithPos(byte('c'), symPos(6)),
						),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(7)),
			),
		},
		{
			pattern: "((a+)+)+",
			ast: genConcatNode(
				genConcatNode(
					genConcatNode(
						genConcatNode(
							newSymbolNodeWithPos(byte('a'), symPos(1)),
							newRepeatNode(
								newSymbolNodeWithPos(byte('a'), symPos(2)),
							),
						),
						newRepeatNode(
							genConcatNode(
								newSymbolNodeWithPos(byte('a'), symPos(3)),
								newRepeatNode(
									newSymbolNodeWithPos(byte('a'), symPos(4)),
								),
							),
						),
					),
					newRepeatNode(
						genConcatNode(
							genConcatNode(
								newSymbolNodeWithPos(byte('a'), symPos(5)),
								newRepeatNode(
									newSymbolNodeWithPos(byte('a'), symPos(6)),
								),
							),
							newRepeatNode(
								genConcatNode(
									newSymbolNodeWithPos(byte('a'), symPos(7)),
									newRepeatNode(
										newSymbolNodeWithPos(byte('a'), symPos(8)),
									),
								),
							),
						),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(9)),
			),
		},
		{
			pattern: "(abc)+",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('b'), symPos(2)),
					newSymbolNodeWithPos(byte('c'), symPos(3)),
				),
				newRepeatNode(
					genConcatNode(
						newSymbolNodeWithPos(byte('a'), symPos(4)),
						newSymbolNodeWithPos(byte('b'), symPos(5)),
						newSymbolNodeWithPos(byte('c'), symPos(6)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(7)),
			),
		},
		{
			pattern: "(a|b)+",
			ast: genConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('b'), symPos(2)),
				),
				newRepeatNode(
					genAltNode(
						newSymbolNodeWithPos(byte('a'), symPos(3)),
						newSymbolNodeWithPos(byte('b'), symPos(4)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(5)),
			),
		},
		{
			pattern:     "+",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "(+)",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "a|+",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "+|b",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern:     "a++",
			syntaxError: synErrRepNoTarget,
		},
		{
			pattern: ".",
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(0x00, 0x7f, symPos(1)),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xc2, 0xdf, symPos(2)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(3)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe0, 0xe0, symPos(4)),
						newRangeSymbolNodeWithPos(0xa0, 0xbf, symPos(5)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(6)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe1, 0xec, symPos(7)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(8)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(9)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xed, 0xed, symPos(10)),
						newRangeSymbolNodeWithPos(0x80, 0x9f, symPos(11)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(12)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xee, 0xef, symPos(13)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(14)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(15)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf0, 0xf0, symPos(16)),
						newRangeSymbolNodeWithPos(0x90, 0xbf, symPos(17)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(18)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(19)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf1, 0xf3, symPos(20)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(21)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(22)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(23)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf4, 0xf4, symPos(24)),
						newRangeSymbolNodeWithPos(0x80, 0x8f, symPos(25)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(26)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(27)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(28)),
			),
		},
		{
			pattern: "[a]",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "[abc]",
			ast: newConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('b'), symPos(2)),
					newSymbolNodeWithPos(byte('c'), symPos(3)),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "[a-z]",
			ast: newConcatNode(
				newRangeSymbolNodeWithPos(byte('a'), byte('z'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "[A-Za-z]",
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(byte('A'), byte('Z'), symPos(1)),
					newRangeSymbolNodeWithPos(byte('a'), byte('z'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern: "[\\u{004E}]",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('N'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern:     "[\\p{Lu}]",
			skipTestAST: true,
		},
		{
			pattern:     "a[]",
			syntaxError: synErrBExpNoElem,
		},
		{
			pattern:     "[]a",
			syntaxError: synErrBExpNoElem,
		},
		{
			pattern:     "[]",
			syntaxError: synErrBExpNoElem,
		},
		{
			pattern: "[^]",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('^'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern:     "[",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "[a",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([a",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "[a-",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([a-",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "[^",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([^",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "[^a",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([^a",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "[^a-",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([^a-",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern: "]",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte(']'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern:     "(]",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern: "a]",
			ast: newConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte(']'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern:     "(a]",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern:     "([)",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern:     "([a)",
			syntaxError: synErrBExpUnclosed,
		},
		{
			pattern: "[a-]",
			ast: newConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('-'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern: "[^a-]",
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(0x00, byte(44), symPos(1)),
					newRangeSymbolNodeWithPos(byte(46), byte(96), symPos(2)),
					newRangeSymbolNodeWithPos(byte(98), 0x7f, symPos(3)),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xc2, 0xdf, symPos(4)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(5)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe0, 0xe0, symPos(6)),
						newRangeSymbolNodeWithPos(0xa0, 0xbf, symPos(7)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(8)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe1, 0xec, symPos(9)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(10)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(11)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xed, 0xed, symPos(12)),
						newRangeSymbolNodeWithPos(0x80, 0x9f, symPos(13)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(14)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xee, 0xef, symPos(15)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(16)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(17)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf0, 0xf0, symPos(18)),
						newRangeSymbolNodeWithPos(0x90, 0xbf, symPos(19)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(20)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(21)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf1, 0xf3, symPos(22)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(23)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(24)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(25)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf4, 0xf4, symPos(26)),
						newRangeSymbolNodeWithPos(0x80, 0x8f, symPos(27)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(28)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(29)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(30)),
			),
		},
		{
			pattern: "[-z]",
			ast: newConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('-'), symPos(1)),
					newSymbolNodeWithPos(byte('z'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern: "[^-z]",
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(0x00, byte(44), symPos(1)),
					genAltNode(
						newRangeSymbolNodeWithPos(byte(46), byte(121), symPos(2)),
						newRangeSymbolNodeWithPos(byte(123), 0x7f, symPos(3)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xc2, 0xdf, symPos(4)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(5)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe0, 0xe0, symPos(6)),
						newRangeSymbolNodeWithPos(0xa0, 0xbf, symPos(7)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(8)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe1, 0xec, symPos(9)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(10)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(11)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xed, 0xed, symPos(12)),
						newRangeSymbolNodeWithPos(0x80, 0x9f, symPos(13)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(14)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xee, 0xef, symPos(15)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(16)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(17)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf0, 0xf0, symPos(18)),
						newRangeSymbolNodeWithPos(0x90, 0xbf, symPos(19)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(20)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(21)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf1, 0xf3, symPos(22)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(23)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(24)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(25)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf4, 0xf4, symPos(26)),
						newRangeSymbolNodeWithPos(0x80, 0x8f, symPos(27)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(28)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(29)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(30)),
			),
		},
		{
			pattern: "[-]",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('-'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "[^-]",
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(0x00, byte(44), symPos(1)),
					newRangeSymbolNodeWithPos(byte(46), 0x7f, symPos(2)),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xc2, 0xdf, symPos(3)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(4)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe0, 0xe0, symPos(5)),
						newRangeSymbolNodeWithPos(0xa0, 0xbf, symPos(6)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(7)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xe1, 0xec, symPos(8)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(9)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(10)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xed, 0xed, symPos(11)),
						newRangeSymbolNodeWithPos(0x80, 0x9f, symPos(12)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(13)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xee, 0xef, symPos(14)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(15)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(16)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf0, 0xf0, symPos(17)),
						newRangeSymbolNodeWithPos(0x90, 0xbf, symPos(18)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(19)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(20)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf1, 0xf3, symPos(21)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(22)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(23)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(24)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(0xf4, 0xf4, symPos(25)),
						newRangeSymbolNodeWithPos(0x80, 0x8f, symPos(26)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(27)),
						newRangeSymbolNodeWithPos(0x80, 0xbf, symPos(28)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(29)),
			),
		},
		{
			pattern: "\\u{006E}",
			ast: genConcatNode(
				newSymbolNodeWithPos(0x6E, symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "\\u{03BD}",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(0xCE, symPos(1)),
					newSymbolNodeWithPos(0xBD, symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			pattern: "\\u{306B}",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(0xE3, symPos(1)),
					newSymbolNodeWithPos(0x81, symPos(2)),
					newSymbolNodeWithPos(0xAB, symPos(3)),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "\\u{01F638}",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(0xF0, symPos(1)),
					newSymbolNodeWithPos(0x9F, symPos(2)),
					newSymbolNodeWithPos(0x98, symPos(3)),
					newSymbolNodeWithPos(0xB8, symPos(4)),
				),
				newEndMarkerNodeWithPos(1, endPos(5)),
			),
		},
		{
			pattern: "\\u{0000}",
			ast: genConcatNode(
				newSymbolNodeWithPos(0x00, symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "\\u{10FFFF}",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNodeWithPos(0xF4, symPos(1)),
					newSymbolNodeWithPos(0x8F, symPos(2)),
					newSymbolNodeWithPos(0xBF, symPos(3)),
					newSymbolNodeWithPos(0xBF, symPos(4)),
				),
				newEndMarkerNodeWithPos(1, endPos(5)),
			),
		},
		{
			pattern:     "\\u{110000}",
			syntaxError: synErrCPExpOutOfRange,
		},
		{
			pattern:     "\\u",
			syntaxError: synErrCPExpInvalidForm,
		},
		{
			pattern:     "\\u{",
			syntaxError: synErrCPExpInvalidForm,
		},
		{
			pattern:     "\\u{03BD",
			syntaxError: synErrCPExpInvalidForm,
		},
		{
			pattern:     "\\u{}",
			syntaxError: synErrCPExpInvalidForm,
		},
		{
			pattern:     "\\p{Letter}",
			skipTestAST: true,
		},
		{
			pattern:     "\\p{General_Category=Letter}",
			skipTestAST: true,
		},
		{
			pattern:     "\\p{ Letter }",
			skipTestAST: true,
		},
		{
			pattern:     "\\p{ General_Category = Letter }",
			skipTestAST: true,
		},
		{
			pattern:     "\\p",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern:     "\\p{",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern:     "\\p{Letter",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern:     "\\p{General_Category=}",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern:     "\\p{General_Category=  }",
			syntaxError: synErrCharPropInvalidSymbol,
		},
		{
			pattern:     "\\p{=Letter}",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern:     "\\p{  =Letter}",
			syntaxError: synErrCharPropInvalidSymbol,
		},
		{
			pattern:     "\\p{=}",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern:     "\\p{}",
			syntaxError: synErrCharPropExpInvalidForm,
		},
		{
			pattern: "\\f{a2c}",
			fragments: map[string]string{
				"a2c": "abc",
			},
			ast: genConcatNode(
				newFragmentNode("a2c",
					genConcatNode(
						newSymbolNodeWithPos(byte('a'), symPos(1)),
						newSymbolNodeWithPos(byte('b'), symPos(2)),
						newSymbolNodeWithPos(byte('c'), symPos(3)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern: "\\f{ a2c }",
			fragments: map[string]string{
				"a2c": "abc",
			},
			ast: genConcatNode(
				newFragmentNode("a2c",
					genConcatNode(
						newSymbolNodeWithPos(byte('a'), symPos(1)),
						newSymbolNodeWithPos(byte('b'), symPos(2)),
						newSymbolNodeWithPos(byte('c'), symPos(3)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(4)),
			),
		},
		{
			pattern:     "\\f",
			syntaxError: synErrFragmentExpInvalidForm,
		},
		{
			pattern:     "\\f{",
			syntaxError: synErrFragmentExpInvalidForm,
		},
		{
			pattern: "\\f{a2c",
			fragments: map[string]string{
				"a2c": "abc",
			},
			syntaxError: synErrFragmentExpInvalidForm,
		},
		{
			pattern: "(a)",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern: "(((a)))",
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			pattern:     "a()",
			syntaxError: synErrGroupNoElem,
		},
		{
			pattern:     "()a",
			syntaxError: synErrGroupNoElem,
		},
		{
			pattern:     "()",
			syntaxError: synErrGroupNoElem,
		},
		{
			pattern:     "(",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern:     "a(",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern:     "(a",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern:     "((",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern:     "((a)",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern:     ")",
			syntaxError: synErrGroupNoInitiator,
		},
		{
			pattern:     "a)",
			syntaxError: synErrGroupNoInitiator,
		},
		{
			pattern:     ")a",
			syntaxError: synErrGroupNoInitiator,
		},
		{
			pattern:     "))",
			syntaxError: synErrGroupNoInitiator,
		},
		{
			pattern:     "(a))",
			syntaxError: synErrGroupNoInitiator,
		},
		{
			pattern: "Mulder|Scully",
			ast: newConcatNode(
				genAltNode(
					genConcatNode(
						newSymbolNodeWithPos(byte('M'), symPos(1)),
						newSymbolNodeWithPos(byte('u'), symPos(2)),
						newSymbolNodeWithPos(byte('l'), symPos(3)),
						newSymbolNodeWithPos(byte('d'), symPos(4)),
						newSymbolNodeWithPos(byte('e'), symPos(5)),
						newSymbolNodeWithPos(byte('r'), symPos(6)),
					),
					genConcatNode(
						newSymbolNodeWithPos(byte('S'), symPos(7)),
						newSymbolNodeWithPos(byte('c'), symPos(8)),
						newSymbolNodeWithPos(byte('u'), symPos(9)),
						newSymbolNodeWithPos(byte('l'), symPos(10)),
						newSymbolNodeWithPos(byte('l'), symPos(11)),
						newSymbolNodeWithPos(byte('y'), symPos(12)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(13)),
			),
		},
		{
			pattern: "Langly|Frohike|Byers",
			ast: newConcatNode(
				genAltNode(
					genConcatNode(
						newSymbolNodeWithPos(byte('L'), symPos(1)),
						newSymbolNodeWithPos(byte('a'), symPos(2)),
						newSymbolNodeWithPos(byte('n'), symPos(3)),
						newSymbolNodeWithPos(byte('g'), symPos(4)),
						newSymbolNodeWithPos(byte('l'), symPos(5)),
						newSymbolNodeWithPos(byte('y'), symPos(6)),
					),
					genConcatNode(
						newSymbolNodeWithPos(byte('F'), symPos(7)),
						newSymbolNodeWithPos(byte('r'), symPos(8)),
						newSymbolNodeWithPos(byte('o'), symPos(9)),
						newSymbolNodeWithPos(byte('h'), symPos(10)),
						newSymbolNodeWithPos(byte('i'), symPos(11)),
						newSymbolNodeWithPos(byte('k'), symPos(12)),
						newSymbolNodeWithPos(byte('e'), symPos(13)),
					),
					genConcatNode(
						newSymbolNodeWithPos(byte('B'), symPos(14)),
						newSymbolNodeWithPos(byte('y'), symPos(15)),
						newSymbolNodeWithPos(byte('e'), symPos(16)),
						newSymbolNodeWithPos(byte('r'), symPos(17)),
						newSymbolNodeWithPos(byte('s'), symPos(18)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(19)),
			),
		},
		{
			pattern:     "|",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "||",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "Mulder|",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "|Scully",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "Langly|Frohike|",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "Langly||Byers",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "|Frohike|Byers",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "|Frohike|",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "Fox(|)Mulder",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "(Fox|)Mulder",
			syntaxError: synErrAltLackOfOperand,
		},
		{
			pattern:     "Fox(|Mulder)",
			syntaxError: synErrAltLackOfOperand,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v %v", i, tt.pattern), func(t *testing.T) {
			fragments := map[string][]byte{}
			for kind, pattern := range tt.fragments {
				fragments[kind] = []byte(pattern)
			}
			ast, _, err := parse([]*patternEntry{
				{
					id:      spec.LexModeKindIDMin,
					pattern: []byte(tt.pattern),
				},
			}, fragments)
			if tt.syntaxError != nil {
				// printAST(os.Stdout, ast, "", "", false)
				if err == nil {
					t.Fatalf("expected syntax error; got: nil")
				}
				parseErrs, ok := err.(*ParseErrors)
				if !ok {
					t.Fatalf("expected ParseErrors; got: %v (type: %T)", err, err)
				}
				parseErr := parseErrs.Errors[0].Cause
				synErr, ok := parseErr.(*SyntaxError)
				if !ok {
					t.Fatalf("expected SyntaxError; got: %v (type: %T)", parseErr, parseErr)
				}
				if synErr != tt.syntaxError {
					t.Fatalf("unexpected syntax error; want: %v, got: %v", tt.syntaxError, synErr)
				}
				if ast != nil {
					t.Fatalf("ast is not nil")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if ast == nil {
					t.Fatal("AST is nil")
				}
				// printAST(os.Stdout, ast, "", "", false)
				if !tt.skipTestAST {
					testAST(t, tt.ast, ast)
				}
			}
		})
	}
}

func TestParse_ContributoryPropertyIsNotExposed(t *testing.T) {
	for _, cProp := range ucd.ContributoryProperties() {
		t.Run(fmt.Sprintf("%v", cProp), func(t *testing.T) {
			ast, _, err := parse([]*patternEntry{
				{
					id:      spec.LexModeKindIDMin,
					pattern: []byte(fmt.Sprintf(`\p{%v=yes}`, cProp)),
				},
			}, nil)
			if err == nil {
				t.Fatalf("expected syntax error; got: nil")
			}
			parseErrs, ok := err.(*ParseErrors)
			if !ok {
				t.Fatalf("expected ParseErrors; got: %v (type: %T)", err, err)
			}
			parseErr := parseErrs.Errors[0].Cause
			synErr, ok := parseErr.(*SyntaxError)
			if !ok {
				t.Fatalf("expected SyntaxError; got: %v (type: %T)", parseErr, parseErr)
			}
			if synErr != synErrCharPropUnsupported {
				t.Fatalf("unexpected syntax error; want: %v, got: %v", synErrCharPropUnsupported, synErr)
			}
			if ast != nil {
				t.Fatalf("ast is not nil")
			}
		})
	}
}

func TestParse_FollowAndSymbolTable(t *testing.T) {
	root, symTab, err := parse([]*patternEntry{
		{
			id:      spec.LexModeKindIDMin,
			pattern: []byte("(a|b)*abb"),
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if root == nil {
		t.Fatal("root of AST is nil")
	}
	// printAST(os.Stdout, root, "", "", false)

	{
		expectedAST := genConcatNode(
			newRepeatNode(
				newAltNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('b'), symPos(2)),
				),
			),
			newSymbolNodeWithPos(byte('a'), symPos(3)),
			newSymbolNodeWithPos(byte('b'), symPos(4)),
			newSymbolNodeWithPos(byte('b'), symPos(5)),
			newEndMarkerNodeWithPos(1, endPos(6)),
		)
		testAST(t, expectedAST, root)
	}

	{
		followTab := genFollowTable(root)
		if followTab == nil {
			t.Fatal("follow table is nil")
		}
		expectedFollowTab := followTable{
			1: newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)),
			2: newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)),
			3: newSymbolPositionSet().add(symPos(4)),
			4: newSymbolPositionSet().add(symPos(5)),
			5: newSymbolPositionSet().add(endPos(6)),
		}
		testFollowTable(t, expectedFollowTab, followTab)
	}

	{
		entry := func(v byte) byteRange {
			return byteRange{
				from: v,
				to:   v,
			}
		}

		expectedSymTab := &symbolTable{
			symPos2Byte: map[symbolPosition]byteRange{
				symPos(1): entry(byte('a')),
				symPos(2): entry(byte('b')),
				symPos(3): entry(byte('a')),
				symPos(4): entry(byte('b')),
				symPos(5): entry(byte('b')),
			},
			endPos2ID: map[symbolPosition]spec.LexModeKindID{
				endPos(6): 1,
			},
		}
		testSymbolTable(t, expectedSymTab, symTab)
	}
}

func testAST(t *testing.T, expected, actual astNode) {
	t.Helper()

	aTy := reflect.TypeOf(actual)
	eTy := reflect.TypeOf(expected)
	if eTy != aTy {
		t.Fatalf("AST node type is mismatched; want: %v, got: %v", eTy, aTy)
	}

	if actual == nil {
		return
	}

	switch e := expected.(type) {
	case *symbolNode:
		a := actual.(*symbolNode)
		if a.pos != e.pos || a.from != e.from || a.to != e.to {
			t.Fatalf("unexpected node; want: %+v, got: %+v", e, a)
		}
	case *endMarkerNode:
		a := actual.(*endMarkerNode)
		if a.pos != e.pos {
			t.Fatalf("symbol position is mismatched; want: %v, got: %v", e.pos, a.pos)
		}
	}
	eLeft, eRight := expected.children()
	aLeft, aRight := actual.children()
	testAST(t, eLeft, aLeft)
	testAST(t, eRight, aRight)
}

func testFollowTable(t *testing.T, expected, actual followTable) {
	if len(actual) != len(expected) {
		t.Errorf("unexpected number of the follow table entries; want: %v, got: %v", len(expected), len(actual))
	}
	for ePos, eSet := range expected {
		aSet, ok := actual[ePos]
		if !ok {
			t.Fatalf("follow entry is not found; position: %v, follow: %v", ePos, eSet)
		}
		if aSet.hash() != eSet.hash() {
			t.Fatalf("follow entry of position %v is mismatched; want: %v, got: %v", ePos, aSet, eSet)
		}
	}
}

func testSymbolTable(t *testing.T, expected, actual *symbolTable) {
	t.Helper()

	if len(actual.symPos2Byte) != len(expected.symPos2Byte) {
		t.Errorf("unexpected symPos2Byte entries; want: %v entries, got: %v entries", len(expected.symPos2Byte), len(actual.symPos2Byte))
	}
	for ePos, eByte := range expected.symPos2Byte {
		byte, ok := actual.symPos2Byte[ePos]
		if !ok {
			t.Errorf("a symbol position entry was not found: %v -> %v", ePos, eByte)
			continue
		}
		if byte.from != eByte.from || byte.to != eByte.to {
			t.Errorf("unexpected symbol position entry; want: %v -> %v, got: %v -> %v", ePos, eByte, ePos, byte)
		}
	}

	if len(actual.endPos2ID) != len(expected.endPos2ID) {
		t.Errorf("unexpected endPos2ID entries; want: %v entries, got: %v entries", len(expected.endPos2ID), len(actual.endPos2ID))
	}
	for ePos, eID := range expected.endPos2ID {
		id, ok := actual.endPos2ID[ePos]
		if !ok {
			t.Errorf("an end position entry was not found: %v -> %v", ePos, eID)
			continue
		}
		if id != eID {
			t.Errorf("unexpected end position entry; want: %v -> %v, got: %v -> %v", ePos, eID, ePos, id)
		}
	}
}
