package driver

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nihei9/maleeni/compiler"
	"github.com/nihei9/maleeni/spec"
)

func TestLexer_Next(t *testing.T) {
	test := []struct {
		lspec  *spec.LexSpec
		src    string
		tokens []*Token
	}{
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					spec.NewLexEntry("t1", "(a|b)*abb"),
					spec.NewLexEntry("t2", " +"),
				},
			},
			src: "abb aabb aaabb babb bbabb abbbabb",
			tokens: []*Token{
				newToken(1, "t1", []byte("abb")),
				newToken(2, "t2", []byte(" ")),
				newToken(1, "t1", []byte("aabb")),
				newToken(2, "t2", []byte(" ")),
				newToken(1, "t1", []byte("aaabb")),
				newToken(2, "t2", []byte(" ")),
				newToken(1, "t1", []byte("babb")),
				newToken(2, "t2", []byte(" ")),
				newToken(1, "t1", []byte("bbabb")),
				newToken(2, "t2", []byte(" ")),
				newToken(1, "t1", []byte("abbbabb")),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					spec.NewLexEntry("t1", "b?a+"),
					spec.NewLexEntry("t2", "(ab)?(cd)+"),
					spec.NewLexEntry("t3", " +"),
				},
			},
			src: "ba baaa a aaa abcd abcdcdcd cd cdcdcd",
			tokens: []*Token{
				newToken(1, "t1", []byte("ba")),
				newToken(3, "t3", []byte(" ")),
				newToken(1, "t1", []byte("baaa")),
				newToken(3, "t3", []byte(" ")),
				newToken(1, "t1", []byte("a")),
				newToken(3, "t3", []byte(" ")),
				newToken(1, "t1", []byte("aaa")),
				newToken(3, "t3", []byte(" ")),
				newToken(2, "t2", []byte("abcd")),
				newToken(3, "t3", []byte(" ")),
				newToken(2, "t2", []byte("abcdcdcd")),
				newToken(3, "t3", []byte(" ")),
				newToken(2, "t2", []byte("cd")),
				newToken(3, "t3", []byte(" ")),
				newToken(2, "t2", []byte("cdcdcd")),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					spec.NewLexEntry("t1", "."),
				},
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
				newToken(1, "t1", []byte{0x00}),
				newToken(1, "t1", []byte{0x7f}),
				newToken(1, "t1", []byte{0xc2, 0x80}),
				newToken(1, "t1", []byte{0xdf, 0xbf}),
				newToken(1, "t1", []byte{0xe1, 0x80, 0x80}),
				newToken(1, "t1", []byte{0xec, 0xbf, 0xbf}),
				newToken(1, "t1", []byte{0xed, 0x80, 0x80}),
				newToken(1, "t1", []byte{0xed, 0x9f, 0xbf}),
				newToken(1, "t1", []byte{0xee, 0x80, 0x80}),
				newToken(1, "t1", []byte{0xef, 0xbf, 0xbf}),
				newToken(1, "t1", []byte{0xf0, 0x90, 0x80, 0x80}),
				newToken(1, "t1", []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newToken(1, "t1", []byte{0xf1, 0x80, 0x80, 0x80}),
				newToken(1, "t1", []byte{0xf3, 0xbf, 0xbf, 0xbf}),
				newToken(1, "t1", []byte{0xf4, 0x80, 0x80, 0x80}),
				newToken(1, "t1", []byte{0xf4, 0x8f, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					spec.NewLexEntry("t1", "[ab.*+?|()[\\]]"),
				},
			},
			src: "ab.*+?|()[]",
			tokens: []*Token{
				newToken(1, "t1", []byte("a")),
				newToken(1, "t1", []byte("b")),
				newToken(1, "t1", []byte(".")),
				newToken(1, "t1", []byte("*")),
				newToken(1, "t1", []byte("+")),
				newToken(1, "t1", []byte("?")),
				newToken(1, "t1", []byte("|")),
				newToken(1, "t1", []byte("(")),
				newToken(1, "t1", []byte(")")),
				newToken(1, "t1", []byte("[")),
				newToken(1, "t1", []byte("]")),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 1 byte characters
					spec.NewLexEntry("1ByteChar", "[\x00-\x7f]"),
				},
			},
			src: string([]byte{
				0x00,
				0x01,
				0x7e,
				0x7f,
			}),
			tokens: []*Token{
				newToken(1, "1ByteChar", []byte{0x00}),
				newToken(1, "1ByteChar", []byte{0x01}),
				newToken(1, "1ByteChar", []byte{0x7e}),
				newToken(1, "1ByteChar", []byte{0x7f}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 2 byte characters
					spec.NewLexEntry("2ByteChar", "[\xc2\x80-\xdf\xbf]"),
				},
			},
			src: string([]byte{
				0xc2, 0x80,
				0xc2, 0x81,
				0xdf, 0xbe,
				0xdf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "2ByteChar", []byte{0xc2, 0x80}),
				newToken(1, "2ByteChar", []byte{0xc2, 0x81}),
				newToken(1, "2ByteChar", []byte{0xdf, 0xbe}),
				newToken(1, "2ByteChar", []byte{0xdf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// All bytes are the same.
					spec.NewLexEntry("3ByteChar", "[\xe0\xa0\x80-\xe0\xa0\x80]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
			}),
			tokens: []*Token{
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x80}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first two bytes are the same.
					spec.NewLexEntry("3ByteChar", "[\xe0\xa0\x80-\xe0\xa0\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xa0, 0xbe,
				0xe0, 0xa0, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x80}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x81}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0xbe}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first byte are the same.
					spec.NewLexEntry("3ByteChar", "[\xe0\xa0\x80-\xe0\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xbf, 0xbe,
				0xe0, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x80}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x81}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xbf, 0xbe}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 3 byte characters
					spec.NewLexEntry("3ByteChar", "[\xe0\xa0\x80-\xef\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xbf, 0xbe,
				0xe0, 0xbf, 0xbf,
				0xe1, 0x80, 0x80,
				0xe1, 0x80, 0x81,
				0xec, 0xbf, 0xbe,
				0xec, 0xbf, 0xbf,
				0xed, 0x80, 0x80,
				0xed, 0x80, 0x81,
				0xed, 0x9f, 0xbe,
				0xed, 0x9f, 0xbf,
				0xee, 0x80, 0x80,
				0xee, 0x80, 0x81,
				0xef, 0xbf, 0xbe,
				0xef, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x80}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xa0, 0x81}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xbf, 0xbe}),
				newToken(1, "3ByteChar", []byte{0xe0, 0xbf, 0xbf}),
				newToken(1, "3ByteChar", []byte{0xe1, 0x80, 0x80}),
				newToken(1, "3ByteChar", []byte{0xe1, 0x80, 0x81}),
				newToken(1, "3ByteChar", []byte{0xec, 0xbf, 0xbe}),
				newToken(1, "3ByteChar", []byte{0xec, 0xbf, 0xbf}),
				newToken(1, "3ByteChar", []byte{0xed, 0x80, 0x80}),
				newToken(1, "3ByteChar", []byte{0xed, 0x80, 0x81}),
				newToken(1, "3ByteChar", []byte{0xed, 0x9f, 0xbe}),
				newToken(1, "3ByteChar", []byte{0xed, 0x9f, 0xbf}),
				newToken(1, "3ByteChar", []byte{0xee, 0x80, 0x80}),
				newToken(1, "3ByteChar", []byte{0xee, 0x80, 0x81}),
				newToken(1, "3ByteChar", []byte{0xef, 0xbf, 0xbe}),
				newToken(1, "3ByteChar", []byte{0xef, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// All bytes are the same.
					spec.NewLexEntry("4ByteChar", "[\xf0\x90\x80\x80-\xf0\x90\x80\x80]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
			}),
			tokens: []*Token{
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x80}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first 3 bytes are the same.
					spec.NewLexEntry("4ByteChar", "[\xf0\x90\x80\x80-\xf0\x90\x80\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0x90, 0x80, 0xbe,
				0xf0, 0x90, 0x80, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x80}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x81}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0xbe}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first 2 bytes are the same.
					spec.NewLexEntry("4ByteChar", "[\xf0\x90\x80\x80-\xf0\x90\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0x90, 0xbf, 0xbe,
				0xf0, 0x90, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x80}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x81}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0xbf, 0xbe}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first byte are the same.
					spec.NewLexEntry("4ByteChar", "[\xf0\x90\x80\x80-\xf0\xbf\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0xbf, 0xbf, 0xbe,
				0xf0, 0xbf, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x80}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x81}),
				newToken(1, "4ByteChar", []byte{0xf0, 0xbf, 0xbf, 0xbe}),
				newToken(1, "4ByteChar", []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 4 byte characters
					spec.NewLexEntry("4ByteChar", "[\xf0\x90\x80\x80-\xf4\x8f\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0xbf, 0xbf, 0xbe,
				0xf0, 0xbf, 0xbf, 0xbf,
				0xf1, 0x80, 0x80, 0x80,
				0xf1, 0x80, 0x80, 0x81,
				0xf3, 0xbf, 0xbf, 0xbe,
				0xf3, 0xbf, 0xbf, 0xbf,
				0xf4, 0x80, 0x80, 0x80,
				0xf4, 0x80, 0x80, 0x81,
				0xf4, 0x8f, 0xbf, 0xbe,
				0xf4, 0x8f, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x80}),
				newToken(1, "4ByteChar", []byte{0xf0, 0x90, 0x80, 0x81}),
				newToken(1, "4ByteChar", []byte{0xf0, 0xbf, 0xbf, 0xbe}),
				newToken(1, "4ByteChar", []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newToken(1, "4ByteChar", []byte{0xf1, 0x80, 0x80, 0x80}),
				newToken(1, "4ByteChar", []byte{0xf1, 0x80, 0x80, 0x81}),
				newToken(1, "4ByteChar", []byte{0xf3, 0xbf, 0xbf, 0xbe}),
				newToken(1, "4ByteChar", []byte{0xf3, 0xbf, 0xbf, 0xbf}),
				newToken(1, "4ByteChar", []byte{0xf4, 0x80, 0x80, 0x80}),
				newToken(1, "4ByteChar", []byte{0xf4, 0x80, 0x80, 0x81}),
				newToken(1, "4ByteChar", []byte{0xf4, 0x8f, 0xbf, 0xbe}),
				newToken(1, "4ByteChar", []byte{0xf4, 0x8f, 0xbf, 0xbf}),
				newEOFToken(),
			},
		},
	}
	for _, tt := range test {
		clspec, err := compiler.Compile(tt.lspec)
		if err != nil {
			t.Fatalf("unexpected error occurred: %v", err)
		}
		lexer, err := NewLexer(clspec, strings.NewReader(tt.src))
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
			// t.Logf("token: ID: %v, Match: %+v Text: \"%v\", EOF: %v, Invalid: %v", tok.ID, tok.Match, string(tok.Match), tok.EOF, tok.Invalid)
			if tok.EOF {
				break
			}
		}
	}
}

func TestLexer_PeekN(t *testing.T) {
	clspec, err := compiler.Compile(&spec.LexSpec{
		Entries: []*spec.LexEntry{
			spec.NewLexEntry("", "foo"),
			spec.NewLexEntry("", "bar"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	lex, err := NewLexer(clspec, strings.NewReader("foobar"))
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

	if actual.ID != expected.ID || actual.Kind != expected.Kind || !bytes.Equal(actual.Match, expected.Match) || actual.EOF != expected.EOF || actual.Invalid != expected.Invalid {
		t.Errorf("unexpected token; want: %v (\"%v\"), got: %v (\"%v\")", expected, string(expected.Match), actual, string(actual.Match))
	}
}
