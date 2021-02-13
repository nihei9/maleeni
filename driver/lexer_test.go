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
		t.Errorf("unexpected token; want: %v, got: %v", expected, actual)
	}
}
