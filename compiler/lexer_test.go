package compiler

import (
	"strings"
	"testing"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		tokens  []*token
		err     error
	}{
		{
			caption: "lexer can recognize ordinaly characters",
			src:     "123abcいろは",
			tokens: []*token{
				newToken(tokenKindChar, '1'),
				newToken(tokenKindChar, '2'),
				newToken(tokenKindChar, '3'),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, 'b'),
				newToken(tokenKindChar, 'c'),
				newToken(tokenKindChar, 'い'),
				newToken(tokenKindChar, 'ろ'),
				newToken(tokenKindChar, 'は'),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the special characters in default mode",
			src:     ".*+?|()[",
			tokens: []*token{
				newToken(tokenKindAnyChar, nullChar),
				newToken(tokenKindRepeat, nullChar),
				newToken(tokenKindRepeatOneOrMore, nullChar),
				newToken(tokenKindOption, nullChar),
				newToken(tokenKindAlt, nullChar),
				newToken(tokenKindGroupOpen, nullChar),
				newToken(tokenKindGroupClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the escape sequences in default mode",
			src:     "\\\\\\.\\*\\+\\?\\|\\(\\)\\[",
			tokens: []*token{
				newToken(tokenKindChar, '\\'),
				newToken(tokenKindChar, '.'),
				newToken(tokenKindChar, '*'),
				newToken(tokenKindChar, '+'),
				newToken(tokenKindChar, '?'),
				newToken(tokenKindChar, '|'),
				newToken(tokenKindChar, '('),
				newToken(tokenKindChar, ')'),
				newToken(tokenKindChar, '['),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "] is treated as an ordinary character in default mode",
			src:     "]",
			tokens: []*token{
				newToken(tokenKindChar, ']'),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the special characters in bracket expression mode",
			src:     "[a-z][^a-z]",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the escape sequences in bracket expression mode",
			src:     "[\\^a\\-z]",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '^'),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "in a bracket expression, the special characters are also handled as normal characters",
			src:     "[\\\\.*+?|()[",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '\\'),
				newToken(tokenKindChar, '.'),
				newToken(tokenKindChar, '*'),
				newToken(tokenKindChar, '+'),
				newToken(tokenKindChar, '?'),
				newToken(tokenKindChar, '|'),
				newToken(tokenKindChar, '('),
				newToken(tokenKindChar, ')'),
				newToken(tokenKindChar, '['),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "hyphen symbols that appear in bracket expressions are handled as the character range symbol or ordinary characters",
			// [...-...][...-][-...][-]
			//  ~~~~~~~     ~  ~     ~
			//     ^        ^  ^     ^
			//     |        |  |     `-- Ordinary Character (b)
			//     |        |  `-- Ordinary Character (b)
			//     |        `-- Ordinary Character (b)
			//     `-- Character Range (a)
			//
			// a. *-* is handled as a character range expression.
			// b. *-, -*, or - are handled as ordinary characters.
			src: "[a-z][a-][-z][-][--][---][^a-z][^a-][^-z][^-][^--][^---]",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "caret symbols that appear in bracket expressions are handled as the logical inverse symbol or ordinary characters",
			// [^...^...][^]
			// ~~   ~    ~~
			// ^    ^    ^^
			// |    |    |`-- Ordinary Character (c)
			// |    |    `-- Bracket Expression
			// |    `-- Ordinary Character (b)
			// `-- Inverse Bracket Expression (a)
			//
			// a. Bracket expressions that have a caret symbol at the beginning are handled as logical inverse expressions.
			// b. caret symbols that appear as the second and the subsequent symbols are handled as ordinary symbols.
			// c. When a bracket expression has just one symbol, a caret symbol at the beginning is handled as an ordinary character.
			src: "[^^][^]",
			tokens: []*token{
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '^'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '^'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer raises an error when an invalid escape sequence appears",
			src:     "\\@",
			err:     synErrInvalidEscSeq,
		},
		{
			caption: "lexer raises an error when the incomplete escape sequence (EOF following \\) appears",
			src:     "\\",
			err:     synErrIncompletedEscSeq,
		},
		{
			caption: "lexer raises an error when an invalid escape sequence appears",
			src:     "[\\@",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
			},
			err: synErrInvalidEscSeq,
		},
		{
			caption: "lexer raises an error when the incomplete escape sequence (EOF following \\) appears",
			src:     "[\\",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
			},
			err: synErrIncompletedEscSeq,
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			lex := newLexer(strings.NewReader(tt.src))
			var err error
			var tok *token
			i := 0
			for {
				tok, err = lex.next()
				if err != nil {
					break
				}
				if i >= len(tt.tokens) {
					break
				}
				eTok := tt.tokens[i]
				i++
				testToken(t, tok, eTok)

				if tok.kind == tokenKindEOF {
					break
				}
			}
			if err != tt.err {
				t.Fatalf("unexpected error; want: %v, got: %v", tt.err, err)
			}
			if i < len(tt.tokens) {
				t.Fatalf("expecte more tokens")
			}
		})
	}
}

func testToken(t *testing.T, a, e *token) {
	t.Helper()
	if e.kind != a.kind || e.char != a.char {
		t.Fatalf("unexpected token; want: %v, got: %v", e, a)
	}
}
