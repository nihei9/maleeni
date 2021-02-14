package driver

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nihei9/maleeni/compiler"
)

func TestLexer_Next(t *testing.T) {
	test := []struct {
		regexps [][]byte
		src     string
		tokens  []*Token
	}{
		{
			regexps: [][]byte{
				[]byte("(a|b)*abb"),
				[]byte(" *"),
			},
			src: "abb aabb   aaabb babb bbabb abbbabb",
			tokens: []*Token{
				newToken(1, []byte("abb")),
				newToken(2, []byte(" ")),
				newToken(1, []byte("aabb")),
				newToken(2, []byte("   ")),
				newToken(1, []byte("aaabb")),
				newToken(2, []byte(" ")),
				newToken(1, []byte("babb")),
				newToken(2, []byte(" ")),
				newToken(1, []byte("bbabb")),
				newToken(2, []byte(" ")),
				newToken(1, []byte("abbbabb")),
				newEOFToken(),
			},
		},
		{
			regexps: [][]byte{
				[]byte("."),
			},
			src: string([]byte{
				0x00,
				0x7f,
				0xc2, 0x80,
				0xdf, 0xbf,
				0xe1, 0x80, 0x80,
				0xec, 0xbf, 0xbf,
				0xed, 0x80, 0x80,
				0xed, 0x9f, 0xbf,
				0xee, 0x80, 0x80,
				0xef, 0xbf, 0xbf,
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0xbf, 0xbf, 0xbf,
				0xf1, 0x80, 0x80, 0x80,
				0xf3, 0xbf, 0xbf, 0xbf,
				0xf4, 0x80, 0x80, 0x80,
				0xf4, 0x8f, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, []byte{0x00}),
				newToken(1, []byte{0x7f}),
				newToken(1, []byte{0xc2, 0x80}),
				newToken(1, []byte{0xdf, 0xbf}),
				newToken(1, []byte{0xe1, 0x80, 0x80}),
				newToken(1, []byte{0xec, 0xbf, 0xbf}),
				newToken(1, []byte{0xed, 0x80, 0x80}),
				newToken(1, []byte{0xed, 0x9f, 0xbf}),
				newToken(1, []byte{0xee, 0x80, 0x80}),
				newToken(1, []byte{0xef, 0xbf, 0xbf}),
				newToken(1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newToken(1, []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newToken(1, []byte{0xf1, 0x80, 0x80, 0x80}),
				newToken(1, []byte{0xf3, 0xbf, 0xbf, 0xbf}),
				newToken(1, []byte{0xf4, 0x80, 0x80, 0x80}),
				newToken(1, []byte{0xf4, 0x8f, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			regexps: [][]byte{
				[]byte("[ab.*|()[\\]]"),
			},
			src: "ab.*|()[]",
			tokens: []*Token{
				newToken(1, []byte("a")),
				newToken(1, []byte("b")),
				newToken(1, []byte(".")),
				newToken(1, []byte("*")),
				newToken(1, []byte("|")),
				newToken(1, []byte("(")),
				newToken(1, []byte(")")),
				newToken(1, []byte("[")),
				newToken(1, []byte("]")),
				newEOFToken(),
			},
		},
	}
	for _, tt := range test {
		res := map[int][]byte{}
		for i, re := range tt.regexps {
			res[i+1] = re
		}
		dfa, err := compiler.Compile(res)
		if err != nil {
			t.Fatalf("unexpected error occurred: %v", err)
		}
		tranTab, err := compiler.GenTransitionTable(dfa)
		if err != nil {
			t.Fatalf("unexpected error occurred: %v", err)
		}
		lexer, err := NewLexer(tranTab, strings.NewReader(tt.src))
		if err != nil {
			t.Fatalf("unexpecated error occurred; %v", err)
		}
		for _, eTok := range tt.tokens {
			tok, err := lexer.Next()
			if err != nil {
				t.Log(err)
				break
			}
			testToken(t, eTok, tok)
			t.Logf("token: ID: %v, Match: %+v Text: \"%v\", EOF: %v, Invalid: %v", tok.ID, tok.Match, string(tok.Match), tok.EOF, tok.Invalid)
			if tok.EOF {
				break
			}
		}
	}
}

func TestLexer_PeekN(t *testing.T) {
	dfa, err := compiler.Compile(map[int][]byte{
		1: []byte("foo"),
		2: []byte("bar"),
	})
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	tranTab, err := compiler.GenTransitionTable(dfa)
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	lex, err := NewLexer(tranTab, strings.NewReader("foobar"))
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}

	expectedTokens := []*Token{
		{
			ID:    1,
			Match: []byte("foo"),
		},
		{
			ID:    2,
			Match: []byte("bar"),
		},
		{
			EOF: true,
		},
	}

	tok, err := lex.Peek1()
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	if tok == nil {
		t.Fatalf("token is nil")
	}
	testToken(t, expectedTokens[0], tok)

	tok, err = lex.Peek2()
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	if tok == nil {
		t.Fatalf("token is nil")
	}
	testToken(t, expectedTokens[1], tok)

	tok, err = lex.Peek3()
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	if tok == nil {
		t.Fatalf("token is nil")
	}
	testToken(t, expectedTokens[2], tok)

	for _, eTok := range expectedTokens {
		tok, err = lex.Next()
		if err != nil {
			t.Fatalf("unexpected error occurred: %v", err)
		}
		if tok == nil {
			t.Fatalf("token is nil")
		}
		testToken(t, eTok, tok)
	}
}

func testToken(t *testing.T, expected, actual *Token) {
	t.Helper()

	if actual.ID != expected.ID || !bytes.Equal(actual.Match, expected.Match) || actual.EOF != expected.EOF || actual.Invalid != expected.Invalid {
		t.Errorf("unexpected token; want: %v (\"%v\"), got: %v (\"%v\")", expected, string(expected.Match), actual, string(actual.Match))
	}
}
