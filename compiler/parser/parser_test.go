package parser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/nihei9/maleeni/spec"
	"github.com/nihei9/maleeni/ucd"
)

func TestParse(t *testing.T) {
	tests := []struct {
		pattern     string
		fragments   map[spec.LexKindName]string
		ast         CPTree
		syntaxError error

		// When an AST is large, as patterns containing a character property expression, this test only checks
		// that the pattern is parsable. The check of the validity of such AST is performed by checking that it
		// can be matched correctly using the driver.
		skipTestAST bool
	}{
		{
			pattern: "a",
			ast:     newSymbolNode('a'),
		},
		{
			pattern: "abc",
			ast: genConcatNode(
				newSymbolNode('a'),
				newSymbolNode('b'),
				newSymbolNode('c'),
			),
		},
		{
			pattern: "a?",
			ast: newOptionNode(
				newSymbolNode('a'),
			),
		},
		{
			pattern: "[abc]?",
			ast: newOptionNode(
				genAltNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
			),
		},
		{
			pattern: "\\u{3042}?",
			ast: newOptionNode(
				newSymbolNode('\u3042'),
			),
		},
		{
			pattern:     "\\p{Letter}?",
			skipTestAST: true,
		},
		{
			pattern: "\\f{a2c}?",
			fragments: map[spec.LexKindName]string{
				"a2c": "abc",
			},
			ast: newOptionNode(
				newFragmentNode("a2c",
					genConcatNode(
						newSymbolNode('a'),
						newSymbolNode('b'),
						newSymbolNode('c'),
					),
				),
			),
		},
		{
			pattern: "(a)?",
			ast: newOptionNode(
				newSymbolNode('a'),
			),
		},
		{
			pattern: "((a?)?)?",
			ast: newOptionNode(
				newOptionNode(
					newOptionNode(
						newSymbolNode('a'),
					),
				),
			),
		},
		{
			pattern: "(abc)?",
			ast: newOptionNode(
				genConcatNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
			),
		},
		{
			pattern: "(a|b)?",
			ast: newOptionNode(
				genAltNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
				),
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
			ast: newRepeatNode(
				newSymbolNode('a'),
			),
		},
		{
			pattern: "[abc]*",
			ast: newRepeatNode(
				genAltNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
			),
		},
		{
			pattern: "\\u{3042}*",
			ast: newRepeatNode(
				newSymbolNode('\u3042'),
			),
		},
		{
			pattern:     "\\p{Letter}*",
			skipTestAST: true,
		},
		{
			pattern: "\\f{a2c}*",
			fragments: map[spec.LexKindName]string{
				"a2c": "abc",
			},
			ast: newRepeatNode(
				newFragmentNode("a2c",
					genConcatNode(
						newSymbolNode('a'),
						newSymbolNode('b'),
						newSymbolNode('c'),
					),
				),
			),
		},
		{
			pattern: "((a*)*)*",
			ast: newRepeatNode(
				newRepeatNode(
					newRepeatNode(
						newSymbolNode('a'),
					),
				),
			),
		},
		{
			pattern: "(abc)*",
			ast: newRepeatNode(
				genConcatNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
			),
		},
		{
			pattern: "(a|b)*",
			ast: newRepeatNode(
				genAltNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
				),
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
				newSymbolNode('a'),
				newRepeatNode(
					newSymbolNode('a'),
				),
			),
		},
		{
			pattern: "[abc]+",
			ast: genConcatNode(
				genAltNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
				newRepeatNode(
					genAltNode(
						newSymbolNode('a'),
						newSymbolNode('b'),
						newSymbolNode('c'),
					),
				),
			),
		},
		{
			pattern: "\\u{3042}+",
			ast: genConcatNode(
				newSymbolNode('\u3042'),
				newRepeatNode(
					newSymbolNode('\u3042'),
				),
			),
		},
		{
			pattern:     "\\p{Letter}+",
			skipTestAST: true,
		},
		{
			pattern: "\\f{a2c}+",
			fragments: map[spec.LexKindName]string{
				"a2c": "abc",
			},
			ast: genConcatNode(
				newFragmentNode("a2c",
					genConcatNode(
						newSymbolNode('a'),
						newSymbolNode('b'),
						newSymbolNode('c'),
					),
				),
				newRepeatNode(
					newFragmentNode("a2c",
						genConcatNode(
							newSymbolNode('a'),
							newSymbolNode('b'),
							newSymbolNode('c'),
						),
					),
				),
			),
		},
		{
			pattern: "((a+)+)+",
			ast: genConcatNode(
				genConcatNode(
					genConcatNode(
						genConcatNode(
							newSymbolNode('a'),
							newRepeatNode(
								newSymbolNode('a'),
							),
						),
						newRepeatNode(
							genConcatNode(
								newSymbolNode('a'),
								newRepeatNode(
									newSymbolNode('a'),
								),
							),
						),
					),
					newRepeatNode(
						genConcatNode(
							genConcatNode(
								newSymbolNode('a'),
								newRepeatNode(
									newSymbolNode('a'),
								),
							),
							newRepeatNode(
								genConcatNode(
									newSymbolNode('a'),
									newRepeatNode(
										newSymbolNode('a'),
									),
								),
							),
						),
					),
				),
			),
		},
		{
			pattern: "(abc)+",
			ast: genConcatNode(
				genConcatNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
				newRepeatNode(
					genConcatNode(
						newSymbolNode('a'),
						newSymbolNode('b'),
						newSymbolNode('c'),
					),
				),
			),
		},
		{
			pattern: "(a|b)+",
			ast: genConcatNode(
				genAltNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
				),
				newRepeatNode(
					genAltNode(
						newSymbolNode('a'),
						newSymbolNode('b'),
					),
				),
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
			ast:     newRangeSymbolNode(0x00, 0x10FFFF),
		},
		{
			pattern: "[a]",
			ast:     newSymbolNode('a'),
		},
		{
			pattern: "[abc]",
			ast: genAltNode(
				newSymbolNode('a'),
				newSymbolNode('b'),
				newSymbolNode('c'),
			),
		},
		{
			pattern: "[a-z]",
			ast:     newRangeSymbolNode('a', 'z'),
		},
		{
			pattern: "[A-Za-z]",
			ast: genAltNode(
				newRangeSymbolNode('A', 'Z'),
				newRangeSymbolNode('a', 'z'),
			),
		},
		{
			pattern: "[\\u{004E}]",
			ast:     newSymbolNode('N'),
		},
		{
			pattern: "[\\u{0061}-\\u{007A}]",
			ast:     newRangeSymbolNode('a', 'z'),
		},
		{
			pattern:     "[\\p{Lu}]",
			skipTestAST: true,
		},
		{
			pattern:     "[a-\\p{Lu}]",
			syntaxError: synErrRangePropIsUnavailable,
		},
		{
			pattern:     "[\\p{Lu}-z]",
			syntaxError: synErrRangePropIsUnavailable,
		},
		{
			pattern:     "[\\p{Lu}-\\p{Ll}]",
			syntaxError: synErrRangePropIsUnavailable,
		},
		{
			pattern:     "[z-a]",
			syntaxError: synErrRangeInvalidOrder,
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
			pattern:     "[^a-z]",
			skipTestAST: true,
		},
		{
			pattern:     "[^\\u{004E}]",
			skipTestAST: true,
		},
		{
			pattern:     "[^\\u{0061}-\\u{007A}]",
			skipTestAST: true,
		},
		{
			pattern:     "[^\\p{Lu}]",
			skipTestAST: true,
		},
		{
			pattern:     "[^a-\\p{Lu}]",
			syntaxError: synErrRangePropIsUnavailable,
		},
		{
			pattern:     "[^\\p{Lu}-z]",
			syntaxError: synErrRangePropIsUnavailable,
		},
		{
			pattern:     "[^\\p{Lu}-\\p{Ll}]",
			syntaxError: synErrRangePropIsUnavailable,
		},
		{
			pattern: "[^]",
			ast:     newSymbolNode('^'),
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
			ast:     newSymbolNode(']'),
		},
		{
			pattern:     "(]",
			syntaxError: synErrGroupUnclosed,
		},
		{
			pattern: "a]",
			ast: genConcatNode(
				newSymbolNode('a'),
				newSymbolNode(']'),
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
			ast: genAltNode(
				newSymbolNode('a'),
				newSymbolNode('-'),
			),
		},
		{
			pattern: "[^a-]",
			ast: genAltNode(
				newRangeSymbolNode(0x00, 0x2C),
				newRangeSymbolNode(0x2E, 0x60),
				newRangeSymbolNode(0x62, 0x10FFFF),
			),
		},
		{
			pattern: "[-z]",
			ast: genAltNode(
				newSymbolNode('-'),
				newSymbolNode('z'),
			),
		},
		{
			pattern: "[^-z]",
			ast: newAltNode(
				newRangeSymbolNode(0x00, 0x2C),
				newAltNode(
					newRangeSymbolNode(0x2E, 0x79),
					newRangeSymbolNode(0x7B, 0x10FFFF),
				),
			),
		},
		{
			pattern: "[-]",
			ast:     newSymbolNode('-'),
		},
		{
			pattern: "[^-]",
			ast: genAltNode(
				newRangeSymbolNode(0x00, 0x2C),
				newRangeSymbolNode(0x2E, 0x10FFFF),
			),
		},
		{
			pattern: "\\u{006E}",
			ast:     newSymbolNode('\u006E'),
		},
		{
			pattern: "\\u{03BD}",
			ast:     newSymbolNode('\u03BD'),
		},
		{
			pattern: "\\u{306B}",
			ast:     newSymbolNode('\u306B'),
		},
		{
			pattern: "\\u{01F638}",
			ast:     newSymbolNode('\U0001F638'),
		},
		{
			pattern: "\\u{0000}",
			ast:     newSymbolNode('\u0000'),
		},
		{
			pattern: "\\u{10FFFF}",
			ast:     newSymbolNode('\U0010FFFF'),
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
			fragments: map[spec.LexKindName]string{
				"a2c": "abc",
			},
			ast: newFragmentNode("a2c",
				genConcatNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
			),
		},
		{
			pattern: "\\f{ a2c }",
			fragments: map[spec.LexKindName]string{
				"a2c": "abc",
			},
			ast: newFragmentNode("a2c",
				genConcatNode(
					newSymbolNode('a'),
					newSymbolNode('b'),
					newSymbolNode('c'),
				),
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
			fragments: map[spec.LexKindName]string{
				"a2c": "abc",
			},
			syntaxError: synErrFragmentExpInvalidForm,
		},
		{
			pattern: "(a)",
			ast:     newSymbolNode('a'),
		},
		{
			pattern: "(((a)))",
			ast:     newSymbolNode('a'),
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
			ast: genAltNode(
				genConcatNode(
					newSymbolNode('M'),
					newSymbolNode('u'),
					newSymbolNode('l'),
					newSymbolNode('d'),
					newSymbolNode('e'),
					newSymbolNode('r'),
				),
				genConcatNode(
					newSymbolNode('S'),
					newSymbolNode('c'),
					newSymbolNode('u'),
					newSymbolNode('l'),
					newSymbolNode('l'),
					newSymbolNode('y'),
				),
			),
		},
		{
			pattern: "Langly|Frohike|Byers",
			ast: genAltNode(
				genConcatNode(
					newSymbolNode('L'),
					newSymbolNode('a'),
					newSymbolNode('n'),
					newSymbolNode('g'),
					newSymbolNode('l'),
					newSymbolNode('y'),
				),
				genConcatNode(
					newSymbolNode('F'),
					newSymbolNode('r'),
					newSymbolNode('o'),
					newSymbolNode('h'),
					newSymbolNode('i'),
					newSymbolNode('k'),
					newSymbolNode('e'),
				),
				genConcatNode(
					newSymbolNode('B'),
					newSymbolNode('y'),
					newSymbolNode('e'),
					newSymbolNode('r'),
					newSymbolNode('s'),
				),
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
			fragmentTrees := map[spec.LexKindName]CPTree{}
			for kind, pattern := range tt.fragments {
				p := NewParser(kind, strings.NewReader(pattern))
				root, err := p.Parse()
				if err != nil {
					t.Fatal(err)
				}

				fragmentTrees[kind] = root
			}
			err := CompleteFragments(fragmentTrees)
			if err != nil {
				t.Fatal(err)
			}

			p := NewParser(spec.LexKindName("test"), strings.NewReader(tt.pattern))
			root, err := p.Parse()
			if tt.syntaxError != nil {
				// printCPTree(os.Stdout, root, "", "")
				if err != ParseErr {
					t.Fatalf("unexpected error: want: %v, got: %v", ParseErr, err)
				}
				_, synErr := p.Error()
				if synErr != tt.syntaxError {
					t.Fatalf("unexpected syntax error: want: %v, got: %v", tt.syntaxError, synErr)
				}
				if root != nil {
					t.Fatalf("tree must be nil")
				}
			} else {
				if err != nil {
					detail, cause := p.Error()
					t.Fatalf("%v: %v: %v", err, cause, detail)
				}
				if root == nil {
					t.Fatal("tree must be non-nil")
				}

				complete, err := ApplyFragments(root, fragmentTrees)
				if err != nil {
					t.Fatal(err)
				}
				if !complete {
					t.Fatalf("incomplete fragments")
				}

				// printCPTree(os.Stdout, root, "", "")
				if !tt.skipTestAST {
					r := root.(*rootNode)
					testAST(t, tt.ast, r.tree)
				}
			}
		})
	}
}

func TestParse_ContributoryPropertyIsNotExposed(t *testing.T) {
	for _, cProp := range ucd.ContributoryProperties() {
		t.Run(fmt.Sprintf("%v", cProp), func(t *testing.T) {
			p := NewParser(spec.LexKindName("test"), strings.NewReader(fmt.Sprintf(`\p{%v=yes}`, cProp)))
			root, err := p.Parse()
			if err == nil {
				t.Fatalf("expected syntax error: got: nil")
			}
			_, synErr := p.Error()
			if synErr != synErrCharPropUnsupported {
				t.Fatalf("unexpected syntax error: want: %v, got: %v", synErrCharPropUnsupported, synErr)
			}
			if root != nil {
				t.Fatalf("tree is not nil")
			}
		})
	}
}

func testAST(t *testing.T, expected, actual CPTree) {
	t.Helper()

	aTy := reflect.TypeOf(actual)
	eTy := reflect.TypeOf(expected)
	if eTy != aTy {
		t.Fatalf("unexpected node: want: %+v, got: %+v", eTy, aTy)
	}

	if actual == nil {
		return
	}

	switch e := expected.(type) {
	case *symbolNode:
		a := actual.(*symbolNode)
		if a.From != e.From || a.To != e.To {
			t.Fatalf("unexpected node: want: %+v, got: %+v", e, a)
		}
	}
	eLeft, eRight := expected.children()
	aLeft, aRight := actual.children()
	testAST(t, eLeft, aLeft)
	testAST(t, eRight, aRight)
}
