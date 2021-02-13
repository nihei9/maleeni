package compiler

import (
	"reflect"
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
			caption: "lexer can recognize the special characters",
			src:     "*|()",
			tokens: []*token{
				newToken(tokenKindRepeat, nullChar),
				newToken(tokenKindAlt, nullChar),
				newToken(tokenKindGroupOpen, nullChar),
				newToken(tokenKindGroupClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the escape sequences",
			src:     "\\\\\\*\\|\\(\\)",
			tokens: []*token{
				newToken(tokenKindChar, '\\'),
				newToken(tokenKindChar, '*'),
				newToken(tokenKindChar, '|'),
				newToken(tokenKindChar, '('),
				newToken(tokenKindChar, ')'),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer raises an error when an invalid escape sequence appears",
			src:     "\\@",
			err:     &SyntaxError{},
		},
		{
			caption: "lexer raises an error when the incomplete escape sequence (EOF following \\) appears",
			src:     "\\",
			err:     &SyntaxError{},
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
			ty := reflect.TypeOf(err)
			eTy := reflect.TypeOf(tt.err)
			if ty != eTy {
				t.Fatalf("unexpected error type; want: %v, got: %v", eTy, ty)
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
