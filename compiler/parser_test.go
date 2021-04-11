package compiler

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	symPos := func(n uint8) symbolPosition {
		return newSymbolPosition(n, false)
	}

	endPos := func(n uint8) symbolPosition {
		return newSymbolPosition(n, true)
	}

	tests := []struct {
		patterns    []string
		ast         astNode
		syntaxError bool
	}{
		{
			patterns: []string{
				"a",
			},
			ast: genConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"abc",
			},
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
			patterns: []string{
				"a?",
			},
			ast: genConcatNode(
				newOptionNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"[abc]?",
			},
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
			patterns: []string{
				"(a)?",
			},
			ast: genConcatNode(
				newOptionNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"((a?)?)?",
			},
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
			patterns: []string{
				"(abc)?",
			},
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
			patterns: []string{
				"(a|b)?",
			},
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
			patterns: []string{
				"?",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"a??",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"a*",
			},
			ast: genConcatNode(
				newRepeatNode(
					newSymbolNodeWithPos(byte('a'), 1),
				),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"[abc]*",
			},
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
			patterns: []string{
				"((a*)*)*",
			},
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
			patterns: []string{
				"(abc)*",
			},
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
			patterns: []string{
				"(a|b)*",
			},
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
			patterns: []string{
				"*",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"a**",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"a+",
			},
			ast: genConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newRepeatNode(
					newSymbolNodeWithPos(byte('a'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			patterns: []string{
				"[abc]+",
			},
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
			patterns: []string{
				"((a+)+)+",
			},
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
			patterns: []string{
				"(abc)+",
			},
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
			patterns: []string{
				"(a|b)+",
			},
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
			patterns: []string{
				"+",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"a++",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				".",
			},
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(bounds1[1][0].min, bounds1[1][0].max, symPos(1)),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds2[1][0].min, bounds2[1][0].max, symPos(2)),
						newRangeSymbolNodeWithPos(bounds2[1][1].min, bounds2[1][1].max, symPos(3)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[1][0].min, bounds3[1][0].max, symPos(4)),
						newRangeSymbolNodeWithPos(bounds3[1][1].min, bounds3[1][1].max, symPos(5)),
						newRangeSymbolNodeWithPos(bounds3[1][2].min, bounds3[1][2].max, symPos(6)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[2][0].min, bounds3[2][0].max, symPos(7)),
						newRangeSymbolNodeWithPos(bounds3[2][1].min, bounds3[2][1].max, symPos(8)),
						newRangeSymbolNodeWithPos(bounds3[2][2].min, bounds3[2][2].max, symPos(9)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[3][0].min, bounds3[3][0].max, symPos(10)),
						newRangeSymbolNodeWithPos(bounds3[3][1].min, bounds3[3][1].max, symPos(11)),
						newRangeSymbolNodeWithPos(bounds3[3][2].min, bounds3[3][2].max, symPos(12)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[4][0].min, bounds3[4][0].max, symPos(13)),
						newRangeSymbolNodeWithPos(bounds3[4][1].min, bounds3[4][1].max, symPos(14)),
						newRangeSymbolNodeWithPos(bounds3[4][2].min, bounds3[4][2].max, symPos(15)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[1][0].min, bounds4[1][0].max, symPos(16)),
						newRangeSymbolNodeWithPos(bounds4[1][1].min, bounds4[1][1].max, symPos(17)),
						newRangeSymbolNodeWithPos(bounds4[1][2].min, bounds4[1][2].max, symPos(18)),
						newRangeSymbolNodeWithPos(bounds4[1][3].min, bounds4[1][3].max, symPos(19)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[2][0].min, bounds4[2][0].max, symPos(20)),
						newRangeSymbolNodeWithPos(bounds4[2][1].min, bounds4[2][1].max, symPos(21)),
						newRangeSymbolNodeWithPos(bounds4[2][2].min, bounds4[2][2].max, symPos(22)),
						newRangeSymbolNodeWithPos(bounds4[2][3].min, bounds4[2][3].max, symPos(23)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[3][0].min, bounds4[3][0].max, symPos(24)),
						newRangeSymbolNodeWithPos(bounds4[3][1].min, bounds4[3][1].max, symPos(25)),
						newRangeSymbolNodeWithPos(bounds4[3][2].min, bounds4[3][2].max, symPos(26)),
						newRangeSymbolNodeWithPos(bounds4[3][3].min, bounds4[3][3].max, symPos(27)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(28)),
			),
		},
		{
			patterns: []string{
				"[a]",
			},
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"[abc]",
			},
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
			patterns: []string{
				"[a-z]",
			},
			ast: newConcatNode(
				newRangeSymbolNodeWithPos(byte('a'), byte('z'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"[A-Za-z]",
			},
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(byte('A'), byte('Z'), symPos(1)),
					newRangeSymbolNodeWithPos(byte('a'), byte('z'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			patterns: []string{
				"a[]",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[]a",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[]",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[^]",
			},
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('^'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"[",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[a",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[a-",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[^",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[^a",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[^a-",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"]",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"[a-]",
			},
			ast: newConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('a'), symPos(1)),
					newSymbolNodeWithPos(byte('-'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			patterns: []string{
				"[^a-]",
			},
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(bounds1[1][0].min, byte(44), symPos(1)),
					newRangeSymbolNodeWithPos(byte(46), byte(96), symPos(2)),
					newRangeSymbolNodeWithPos(byte(98), bounds1[1][0].max, symPos(3)),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds2[1][0].min, bounds2[1][0].max, symPos(4)),
						newRangeSymbolNodeWithPos(bounds2[1][1].min, bounds2[1][1].max, symPos(5)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[1][0].min, bounds3[1][0].max, symPos(6)),
						newRangeSymbolNodeWithPos(bounds3[1][1].min, bounds3[1][1].max, symPos(7)),
						newRangeSymbolNodeWithPos(bounds3[1][2].min, bounds3[1][2].max, symPos(8)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[2][0].min, bounds3[2][0].max, symPos(9)),
						newRangeSymbolNodeWithPos(bounds3[2][1].min, bounds3[2][1].max, symPos(10)),
						newRangeSymbolNodeWithPos(bounds3[2][2].min, bounds3[2][2].max, symPos(11)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[3][0].min, bounds3[3][0].max, symPos(12)),
						newRangeSymbolNodeWithPos(bounds3[3][1].min, bounds3[3][1].max, symPos(13)),
						newRangeSymbolNodeWithPos(bounds3[3][2].min, bounds3[3][2].max, symPos(14)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[4][0].min, bounds3[4][0].max, symPos(15)),
						newRangeSymbolNodeWithPos(bounds3[4][1].min, bounds3[4][1].max, symPos(16)),
						newRangeSymbolNodeWithPos(bounds3[4][2].min, bounds3[4][2].max, symPos(17)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[1][0].min, bounds4[1][0].max, symPos(18)),
						newRangeSymbolNodeWithPos(bounds4[1][1].min, bounds4[1][1].max, symPos(19)),
						newRangeSymbolNodeWithPos(bounds4[1][2].min, bounds4[1][2].max, symPos(20)),
						newRangeSymbolNodeWithPos(bounds4[1][3].min, bounds4[1][3].max, symPos(21)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[2][0].min, bounds4[2][0].max, symPos(22)),
						newRangeSymbolNodeWithPos(bounds4[2][1].min, bounds4[2][1].max, symPos(23)),
						newRangeSymbolNodeWithPos(bounds4[2][2].min, bounds4[2][2].max, symPos(24)),
						newRangeSymbolNodeWithPos(bounds4[2][3].min, bounds4[2][3].max, symPos(25)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[3][0].min, bounds4[3][0].max, symPos(26)),
						newRangeSymbolNodeWithPos(bounds4[3][1].min, bounds4[3][1].max, symPos(27)),
						newRangeSymbolNodeWithPos(bounds4[3][2].min, bounds4[3][2].max, symPos(28)),
						newRangeSymbolNodeWithPos(bounds4[3][3].min, bounds4[3][3].max, symPos(29)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(30)),
			),
		},
		{
			patterns: []string{
				"[-z]",
			},
			ast: newConcatNode(
				genAltNode(
					newSymbolNodeWithPos(byte('-'), symPos(1)),
					newSymbolNodeWithPos(byte('z'), symPos(2)),
				),
				newEndMarkerNodeWithPos(1, endPos(3)),
			),
		},
		{
			patterns: []string{
				"[^-z]",
			},
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(bounds1[1][0].min, byte(44), symPos(1)),
					genAltNode(
						newRangeSymbolNodeWithPos(byte(46), byte(121), symPos(2)),
						newRangeSymbolNodeWithPos(byte(123), bounds1[1][0].max, symPos(3)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds2[1][0].min, bounds2[1][0].max, symPos(4)),
						newRangeSymbolNodeWithPos(bounds2[1][1].min, bounds2[1][1].max, symPos(5)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[1][0].min, bounds3[1][0].max, symPos(6)),
						newRangeSymbolNodeWithPos(bounds3[1][1].min, bounds3[1][1].max, symPos(7)),
						newRangeSymbolNodeWithPos(bounds3[1][2].min, bounds3[1][2].max, symPos(8)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[2][0].min, bounds3[2][0].max, symPos(9)),
						newRangeSymbolNodeWithPos(bounds3[2][1].min, bounds3[2][1].max, symPos(10)),
						newRangeSymbolNodeWithPos(bounds3[2][2].min, bounds3[2][2].max, symPos(11)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[3][0].min, bounds3[3][0].max, symPos(12)),
						newRangeSymbolNodeWithPos(bounds3[3][1].min, bounds3[3][1].max, symPos(13)),
						newRangeSymbolNodeWithPos(bounds3[3][2].min, bounds3[3][2].max, symPos(14)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[4][0].min, bounds3[4][0].max, symPos(15)),
						newRangeSymbolNodeWithPos(bounds3[4][1].min, bounds3[4][1].max, symPos(16)),
						newRangeSymbolNodeWithPos(bounds3[4][2].min, bounds3[4][2].max, symPos(17)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[1][0].min, bounds4[1][0].max, symPos(18)),
						newRangeSymbolNodeWithPos(bounds4[1][1].min, bounds4[1][1].max, symPos(19)),
						newRangeSymbolNodeWithPos(bounds4[1][2].min, bounds4[1][2].max, symPos(20)),
						newRangeSymbolNodeWithPos(bounds4[1][3].min, bounds4[1][3].max, symPos(21)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[2][0].min, bounds4[2][0].max, symPos(22)),
						newRangeSymbolNodeWithPos(bounds4[2][1].min, bounds4[2][1].max, symPos(23)),
						newRangeSymbolNodeWithPos(bounds4[2][2].min, bounds4[2][2].max, symPos(24)),
						newRangeSymbolNodeWithPos(bounds4[2][3].min, bounds4[2][3].max, symPos(25)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[3][0].min, bounds4[3][0].max, symPos(26)),
						newRangeSymbolNodeWithPos(bounds4[3][1].min, bounds4[3][1].max, symPos(27)),
						newRangeSymbolNodeWithPos(bounds4[3][2].min, bounds4[3][2].max, symPos(28)),
						newRangeSymbolNodeWithPos(bounds4[3][3].min, bounds4[3][3].max, symPos(29)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(30)),
			),
		},
		{
			patterns: []string{
				"[-]",
			},
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('-'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"[^-]",
			},
			ast: newConcatNode(
				genAltNode(
					newRangeSymbolNodeWithPos(bounds1[1][0].min, byte(44), symPos(1)),
					newRangeSymbolNodeWithPos(byte(46), bounds1[1][0].max, symPos(2)),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds2[1][0].min, bounds2[1][0].max, symPos(3)),
						newRangeSymbolNodeWithPos(bounds2[1][1].min, bounds2[1][1].max, symPos(4)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[1][0].min, bounds3[1][0].max, symPos(5)),
						newRangeSymbolNodeWithPos(bounds3[1][1].min, bounds3[1][1].max, symPos(6)),
						newRangeSymbolNodeWithPos(bounds3[1][2].min, bounds3[1][2].max, symPos(7)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[2][0].min, bounds3[2][0].max, symPos(8)),
						newRangeSymbolNodeWithPos(bounds3[2][1].min, bounds3[2][1].max, symPos(9)),
						newRangeSymbolNodeWithPos(bounds3[2][2].min, bounds3[2][2].max, symPos(10)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[3][0].min, bounds3[3][0].max, symPos(11)),
						newRangeSymbolNodeWithPos(bounds3[3][1].min, bounds3[3][1].max, symPos(12)),
						newRangeSymbolNodeWithPos(bounds3[3][2].min, bounds3[3][2].max, symPos(13)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds3[4][0].min, bounds3[4][0].max, symPos(14)),
						newRangeSymbolNodeWithPos(bounds3[4][1].min, bounds3[4][1].max, symPos(15)),
						newRangeSymbolNodeWithPos(bounds3[4][2].min, bounds3[4][2].max, symPos(16)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[1][0].min, bounds4[1][0].max, symPos(17)),
						newRangeSymbolNodeWithPos(bounds4[1][1].min, bounds4[1][1].max, symPos(18)),
						newRangeSymbolNodeWithPos(bounds4[1][2].min, bounds4[1][2].max, symPos(19)),
						newRangeSymbolNodeWithPos(bounds4[1][3].min, bounds4[1][3].max, symPos(20)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[2][0].min, bounds4[2][0].max, symPos(21)),
						newRangeSymbolNodeWithPos(bounds4[2][1].min, bounds4[2][1].max, symPos(22)),
						newRangeSymbolNodeWithPos(bounds4[2][2].min, bounds4[2][2].max, symPos(23)),
						newRangeSymbolNodeWithPos(bounds4[2][3].min, bounds4[2][3].max, symPos(24)),
					),
					genConcatNode(
						newRangeSymbolNodeWithPos(bounds4[3][0].min, bounds4[3][0].max, symPos(25)),
						newRangeSymbolNodeWithPos(bounds4[3][1].min, bounds4[3][1].max, symPos(26)),
						newRangeSymbolNodeWithPos(bounds4[3][2].min, bounds4[3][2].max, symPos(27)),
						newRangeSymbolNodeWithPos(bounds4[3][3].min, bounds4[3][3].max, symPos(28)),
					),
				),
				newEndMarkerNodeWithPos(1, endPos(29)),
			),
		},
		{
			patterns: []string{
				"(a)",
			},
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"(((a)))",
			},
			ast: newConcatNode(
				newSymbolNodeWithPos(byte('a'), symPos(1)),
				newEndMarkerNodeWithPos(1, endPos(2)),
			),
		},
		{
			patterns: []string{
				"a()",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"()a",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"()",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"(",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				")",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"Mulder|Scully",
			},
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
			patterns: []string{
				"Langly|Frohike|Byers",
			},
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
			patterns: []string{
				"|",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"||",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"Mulder|",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"|Scully",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"Langly|Frohike|",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"Langly||Byers",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"|Frohike|Byers",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"|Frohike|",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"Fox(|)Mulder",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"(Fox|)Mulder",
			},
			syntaxError: true,
		},
		{
			patterns: []string{
				"Fox(|Mulder)",
			},
			syntaxError: true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			ps := map[int][]byte{}
			for i, p := range tt.patterns {
				ps[i+1] = []byte(p)
			}
			root, _, err := parse(ps)
			if tt.syntaxError {
				// printAST(os.Stdout, root, "", "", false)
				if err == nil {
					t.Fatalf("expected syntax error; got: nil")
				}
				if _, ok := err.(*SyntaxError); !ok {
					t.Fatalf("expected syntax error; got: %v (type: %T)", err, err)
				}
				if root != nil {
					t.Fatalf("root is not nil")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if root == nil {
					t.Fatal("AST is nil")
				}
				// printAST(os.Stdout, root, "", "", false)
				testAST(t, tt.ast, root)
			}
		})
	}

	// Test a FOLLOE table and a symbol table
	{
		root, symTab, err := parse(map[int][]byte{
			1: []byte("(a|b)*abb"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if root == nil {
			t.Fatal("root of AST is nil")
		}
		printAST(os.Stdout, root, "", "", false)

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
				endPos2ID: map[symbolPosition]int{
					endPos(6): 1,
				},
			}
			testSymbolTable(t, expectedSymTab, symTab)
		}
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
		if a.pos != e.pos {
			t.Fatalf("symbol position is mismatched; want: %v, got: %v", e.pos, a.pos)
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
