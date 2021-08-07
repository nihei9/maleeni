package driver

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/maleeni/compiler"
	"github.com/nihei9/maleeni/spec"
)

func newLexEntry(modes []string, kind string, pattern string, push string, pop bool) *spec.LexEntry {
	ms := []spec.LexModeName{}
	for _, m := range modes {
		ms = append(ms, spec.LexModeName(m))
	}
	return &spec.LexEntry{
		Kind:    spec.LexKindName(kind),
		Pattern: spec.LexPattern(pattern),
		Modes:   ms,
		Push:    spec.LexModeName(push),
		Pop:     pop,
	}
}

func newLexEntryDefaultNOP(kind string, pattern string) *spec.LexEntry {
	return &spec.LexEntry{
		Kind:    spec.LexKindName(kind),
		Pattern: spec.LexPattern(pattern),
		Modes: []spec.LexModeName{
			spec.LexModeNameDefault,
		},
	}
}

func newLexEntryFragment(kind string, pattern string) *spec.LexEntry {
	return &spec.LexEntry{
		Kind:     spec.LexKindName(kind),
		Pattern:  spec.LexPattern(pattern),
		Fragment: true,
	}
}

func newToken(modeID spec.LexModeID, modeName spec.LexModeName, kindID spec.LexKindID, modeKindID spec.LexModeKindID, kindName spec.LexKindName, match byteSequence) *Token {
	return &Token{
		ModeID:     modeID,
		ModeName:   modeName,
		KindID:     kindID,
		ModeKindID: modeKindID,
		KindName:   kindName,
		match:      match,
	}
}

func newTokenDefault(kindID int, modeKindID int, kindName string, match byteSequence) *Token {
	return newToken(spec.LexModeIDDefault, spec.LexModeNameDefault, spec.LexKindID(kindID), spec.LexModeKindID(modeKindID), spec.LexKindName(kindName), match)
}

func newEOFToken(modeID spec.LexModeID, modeName spec.LexModeName) *Token {
	return &Token{
		ModeID:     modeID,
		ModeName:   modeName,
		ModeKindID: 0,
		EOF:        true,
	}
}

func newEOFTokenDefault() *Token {
	return newEOFToken(spec.LexModeIDDefault, spec.LexModeNameDefault)
}

func newInvalidToken(modeID spec.LexModeID, modeName spec.LexModeName, match byteSequence) *Token {
	return &Token{
		ModeID:     modeID,
		ModeName:   modeName,
		ModeKindID: 0,
		match:      match,
		Invalid:    true,
	}
}

func withPos(tok *Token, row, col int) *Token {
	tok.Row = row
	tok.Col = col
	return tok
}

func TestLexer_Next(t *testing.T) {
	test := []struct {
		lspec           *spec.LexSpec
		src             string
		tokens          []*Token
		passiveModeTran bool
		tran            func(l *Lexer, tok *Token) error
	}{
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "(a|b)*abb"),
					newLexEntryDefaultNOP("t2", " +"),
				},
			},
			src: "abb aabb aaabb babb bbabb abbbabb",
			tokens: []*Token{
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("abb"))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("aabb"))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("aaabb"))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("babb"))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("bbabb"))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("abbbabb"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "b?a+"),
					newLexEntryDefaultNOP("t2", "(ab)?(cd)+"),
					newLexEntryDefaultNOP("t3", " +"),
				},
			},
			src: "ba baaa a aaa abcd abcdcdcd cd cdcdcd",
			tokens: []*Token{
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("ba"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("baaa"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("a"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("aaa"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte("abcd"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte("abcdcdcd"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte("cd"))),
				newTokenDefault(3, 3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, 2, "t2", newByteSequence([]byte("cdcdcd"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "."),
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
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0x00})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0x7f})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xc2, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xdf, 0xbf})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xe1, 0x80, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xec, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xed, 0x80, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xed, 0x9f, 0xbf})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xee, 0x80, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xef, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xf1, 0x80, 0x80, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xf3, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xf4, 0x80, 0x80, 0x80})),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte{0xf4, 0x8f, 0xbf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "[ab.*+?|()[\\]]"),
				},
			},
			src: "ab.*+?|()[]",
			tokens: []*Token{
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("a"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("b"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("."))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("*"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("+"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("?"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("|"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("("))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte(")"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("["))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("]"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 1 byte characters except null character (U+0000)
					//
					// NOTE:
					// maleeni cannot handle the null character in patterns because compiler.lexer,
					// specifically read() and restore(), recognizes the null characters as that a symbol doesn't exist.
					// If a pattern needs a null character, use code point expression \u{0000}.
					newLexEntryDefaultNOP("1ByteChar", "[\x01-\x7f]"),
				},
			},
			src: string([]byte{
				0x01,
				0x02,
				0x7e,
				0x7f,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "1ByteChar", newByteSequence([]byte{0x01})),
				newTokenDefault(1, 1, "1ByteChar", newByteSequence([]byte{0x02})),
				newTokenDefault(1, 1, "1ByteChar", newByteSequence([]byte{0x7e})),
				newTokenDefault(1, 1, "1ByteChar", newByteSequence([]byte{0x7f})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 2 byte characters
					newLexEntryDefaultNOP("2ByteChar", "[\xc2\x80-\xdf\xbf]"),
				},
			},
			src: string([]byte{
				0xc2, 0x80,
				0xc2, 0x81,
				0xdf, 0xbe,
				0xdf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "2ByteChar", newByteSequence([]byte{0xc2, 0x80})),
				newTokenDefault(1, 1, "2ByteChar", newByteSequence([]byte{0xc2, 0x81})),
				newTokenDefault(1, 1, "2ByteChar", newByteSequence([]byte{0xdf, 0xbe})),
				newTokenDefault(1, 1, "2ByteChar", newByteSequence([]byte{0xdf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// All bytes are the same.
					newLexEntryDefaultNOP("3ByteChar", "[\xe0\xa0\x80-\xe0\xa0\x80]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first two bytes are the same.
					newLexEntryDefaultNOP("3ByteChar", "[\xe0\xa0\x80-\xe0\xa0\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xa0, 0xbe,
				0xe0, 0xa0, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x81})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0xbe})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first byte are the same.
					newLexEntryDefaultNOP("3ByteChar", "[\xe0\xa0\x80-\xe0\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xbf, 0xbe,
				0xe0, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x81})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 3 byte characters
					newLexEntryDefaultNOP("3ByteChar", "[\xe0\xa0\x80-\xef\xbf\xbf]"),
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
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x81})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe1, 0x80, 0x80})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xe1, 0x80, 0x81})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xec, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xec, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xed, 0x80, 0x80})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xed, 0x80, 0x81})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xed, 0x9f, 0xbe})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xed, 0x9f, 0xbf})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xee, 0x80, 0x80})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xee, 0x80, 0x81})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xef, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "3ByteChar", newByteSequence([]byte{0xef, 0xbf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// All bytes are the same.
					newLexEntryDefaultNOP("4ByteChar", "[\xf0\x90\x80\x80-\xf0\x90\x80\x80]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first 3 bytes are the same.
					newLexEntryDefaultNOP("4ByteChar", "[\xf0\x90\x80\x80-\xf0\x90\x80\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0x90, 0x80, 0xbe,
				0xf0, 0x90, 0x80, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0xbe})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first 2 bytes are the same.
					newLexEntryDefaultNOP("4ByteChar", "[\xf0\x90\x80\x80-\xf0\x90\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0x90, 0xbf, 0xbe,
				0xf0, 0x90, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0xbf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// The first byte are the same.
					newLexEntryDefaultNOP("4ByteChar", "[\xf0\x90\x80\x80-\xf0\xbf\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0xbf, 0xbf, 0xbe,
				0xf0, 0xbf, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// all 4 byte characters
					newLexEntryDefaultNOP("4ByteChar", "[\xf0\x90\x80\x80-\xf4\x8f\xbf\xbf]"),
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
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf1, 0x80, 0x80, 0x80})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf1, 0x80, 0x80, 0x81})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf3, 0xbf, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf3, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf4, 0x80, 0x80, 0x80})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf4, 0x80, 0x80, 0x81})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf4, 0x8f, 0xbf, 0xbe})),
				newTokenDefault(1, 1, "4ByteChar", newByteSequence([]byte{0xf4, 0x8f, 0xbf, 0xbf})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("NonNumber", "[^0-9]+[0-9]"),
				},
			},
			src: "foo9",
			tokens: []*Token{
				newTokenDefault(1, 1, "NonNumber", newByteSequence([]byte("foo9"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("1ByteChar", "\\u{006E}"),
					newLexEntryDefaultNOP("2ByteChar", "\\u{03BD}"),
					newLexEntryDefaultNOP("3ByteChar", "\\u{306B}"),
					newLexEntryDefaultNOP("4ByteChar", "\\u{01F638}"),
				},
			},
			src: "nνに😸",
			tokens: []*Token{
				newTokenDefault(1, 1, "1ByteChar", newByteSequence([]byte{0x6E})),
				newTokenDefault(2, 2, "2ByteChar", newByteSequence([]byte{0xCE, 0xBD})),
				newTokenDefault(3, 3, "3ByteChar", newByteSequence([]byte{0xE3, 0x81, 0xAB})),
				newTokenDefault(4, 4, "4ByteChar", newByteSequence([]byte{0xF0, 0x9F, 0x98, 0xB8})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("codePointsAlt", "[\\u{006E}\\u{03BD}\\u{306B}\\u{01F638}]"),
				},
			},
			src: "nνに😸",
			tokens: []*Token{
				newTokenDefault(1, 1, "codePointsAlt", newByteSequence([]byte{0x6E})),
				newTokenDefault(1, 1, "codePointsAlt", newByteSequence([]byte{0xCE, 0xBD})),
				newTokenDefault(1, 1, "codePointsAlt", newByteSequence([]byte{0xE3, 0x81, 0xAB})),
				newTokenDefault(1, 1, "codePointsAlt", newByteSequence([]byte{0xF0, 0x9F, 0x98, 0xB8})),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "\\f{a2c}\\f{d2f}+"),
					newLexEntryFragment("a2c", "abc"),
					newLexEntryFragment("d2f", "def"),
				},
			},
			src: "abcdefdefabcdef",
			tokens: []*Token{
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("abcdefdef"))),
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("abcdef"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "(\\f{a2c}|\\f{d2f})+"),
					newLexEntryFragment("a2c", "abc"),
					newLexEntryFragment("d2f", "def"),
				},
			},
			src: "abcdefdefabc",
			tokens: []*Token{
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("abcdefdefabc"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("t1", "\\f{a2c_or_d2f}+"),
					newLexEntryFragment("a2c_or_d2f", "\\f{a2c}|\\f{d2f}"),
					newLexEntryFragment("a2c", "abc"),
					newLexEntryFragment("d2f", "def"),
				},
			},
			src: "abcdefdefabc",
			tokens: []*Token{
				newTokenDefault(1, 1, "t1", newByteSequence([]byte("abcdefdefabc"))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("white_space", ` *`),
					newLexEntry([]string{"default"}, "string_open", `"`, "string", false),
					newLexEntry([]string{"string"}, "escape_sequence", `\\[n"\\]`, "", false),
					newLexEntry([]string{"string"}, "char_sequence", `[^"\\]*`, "", false),
					newLexEntry([]string{"string"}, "string_close", `"`, "", true),
				},
			},
			src: `"" "Hello world.\n\"Hello world.\""`,
			tokens: []*Token{
				newToken(1, "default", 2, 2, "string_open", newByteSequence([]byte(`"`))),
				newToken(2, "string", 5, 3, "string_close", newByteSequence([]byte(`"`))),
				newToken(1, "default", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(1, "default", 2, 2, "string_open", newByteSequence([]byte(`"`))),
				newToken(2, "string", 4, 2, "char_sequence", newByteSequence([]byte(`Hello world.`))),
				newToken(2, "string", 3, 1, "escape_sequence", newByteSequence([]byte(`\n`))),
				newToken(2, "string", 3, 1, "escape_sequence", newByteSequence([]byte(`\"`))),
				newToken(2, "string", 4, 2, "char_sequence", newByteSequence([]byte(`Hello world.`))),
				newToken(2, "string", 3, 1, "escape_sequence", newByteSequence([]byte(`\"`))),
				newToken(2, "string", 5, 3, "string_close", newByteSequence([]byte(`"`))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					// `white_space` is enabled in multiple modes.
					newLexEntry([]string{"default", "state_a", "state_b"}, "white_space", ` *`, "", false),
					newLexEntry([]string{"default"}, "char_a", `a`, "state_a", false),
					newLexEntry([]string{"state_a"}, "char_b", `b`, "state_b", false),
					newLexEntry([]string{"state_a"}, "back_from_a", `<`, "", true),
					newLexEntry([]string{"state_b"}, "back_from_b", `<`, "", true),
				},
			},
			src: ` a b < < `,
			tokens: []*Token{
				newToken(1, "default", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(1, "default", 2, 2, "char_a", newByteSequence([]byte(`a`))),
				newToken(2, "state_a", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "state_a", 3, 2, "char_b", newByteSequence([]byte(`b`))),
				newToken(3, "state_b", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(3, "state_b", 5, 2, "back_from_b", newByteSequence([]byte(`<`))),
				newToken(2, "state_a", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "state_a", 4, 3, "back_from_a", newByteSequence([]byte(`<`))),
				newToken(1, "default", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntry([]string{"default", "mode_1", "mode_2"}, "white_space", ` *`, "", false),
					newLexEntry([]string{"default"}, "char", `.`, "", false),
					newLexEntry([]string{"default"}, "push_1", `-> 1`, "", false),
					newLexEntry([]string{"mode_1"}, "push_2", `-> 2`, "", false),
					newLexEntry([]string{"mode_1"}, "pop_1", `<-`, "", false),
					newLexEntry([]string{"mode_2"}, "pop_2", `<-`, "", false),
				},
			},
			src: `-> 1 -> 2 <- <- a`,
			tokens: []*Token{
				newToken(1, "default", 3, 3, "push_1", newByteSequence([]byte(`-> 1`))),
				newToken(2, "mode_1", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "mode_1", 4, 2, "push_2", newByteSequence([]byte(`-> 2`))),
				newToken(3, "mode_2", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(3, "mode_2", 6, 2, "pop_2", newByteSequence([]byte(`<-`))),
				newToken(2, "mode_1", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "mode_1", 5, 3, "pop_1", newByteSequence([]byte(`<-`))),
				newToken(1, "default", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(1, "default", 2, 2, "char", newByteSequence([]byte(`a`))),
				newEOFTokenDefault(),
			},
			passiveModeTran: true,
			tran: func(l *Lexer, tok *Token) error {
				switch l.clspec.ModeNames[l.Mode()] {
				case "default":
					switch tok.KindName {
					case "push_1":
						l.PushMode(2)
					}
				case "mode_1":
					switch tok.KindName {
					case "push_2":
						l.PushMode(3)
					case "pop_1":
						return l.PopMode()
					}
				case "mode_2":
					switch tok.KindName {
					case "pop_2":
						return l.PopMode()
					}
				}
				return nil
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntry([]string{"default", "mode_1", "mode_2"}, "white_space", ` *`, "", false),
					newLexEntry([]string{"default"}, "char", `.`, "", false),
					newLexEntry([]string{"default"}, "push_1", `-> 1`, "mode_1", false),
					newLexEntry([]string{"mode_1"}, "push_2", `-> 2`, "", false),
					newLexEntry([]string{"mode_1"}, "pop_1", `<-`, "", false),
					newLexEntry([]string{"mode_2"}, "pop_2", `<-`, "", true),
				},
			},
			src: `-> 1 -> 2 <- <- a`,
			tokens: []*Token{
				newToken(1, "default", 3, 3, "push_1", newByteSequence([]byte(`-> 1`))),
				newToken(2, "mode_1", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "mode_1", 4, 2, "push_2", newByteSequence([]byte(`-> 2`))),
				newToken(3, "mode_2", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(3, "mode_2", 6, 2, "pop_2", newByteSequence([]byte(`<-`))),
				newToken(2, "mode_1", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "mode_1", 5, 3, "pop_1", newByteSequence([]byte(`<-`))),
				newToken(1, "default", 1, 1, "white_space", newByteSequence([]byte(` `))),
				newToken(1, "default", 2, 2, "char", newByteSequence([]byte(`a`))),
				newEOFTokenDefault(),
			},
			// Active mode transition and an external transition function can be used together.
			passiveModeTran: false,
			tran: func(l *Lexer, tok *Token) error {
				switch l.clspec.ModeNames[l.Mode()] {
				case "mode_1":
					switch tok.KindName {
					case "push_2":
						l.PushMode(3)
					case "pop_1":
						return l.PopMode()
					}
				}
				return nil
			},
		},
		{
			lspec: &spec.LexSpec{
				Entries: []*spec.LexEntry{
					newLexEntryDefaultNOP("dot", spec.EscapePattern(`.`)),
					newLexEntryDefaultNOP("star", spec.EscapePattern(`*`)),
					newLexEntryDefaultNOP("plus", spec.EscapePattern(`+`)),
					newLexEntryDefaultNOP("question", spec.EscapePattern(`?`)),
					newLexEntryDefaultNOP("vbar", spec.EscapePattern(`|`)),
					newLexEntryDefaultNOP("lparen", spec.EscapePattern(`(`)),
					newLexEntryDefaultNOP("rparen", spec.EscapePattern(`)`)),
					newLexEntryDefaultNOP("lbrace", spec.EscapePattern(`[`)),
					newLexEntryDefaultNOP("backslash", spec.EscapePattern(`\`)),
				},
			},
			src: `.*+?|()[\`,
			tokens: []*Token{
				newTokenDefault(1, 1, "dot", newByteSequence([]byte(`.`))),
				newTokenDefault(2, 2, "star", newByteSequence([]byte(`*`))),
				newTokenDefault(3, 3, "plus", newByteSequence([]byte(`+`))),
				newTokenDefault(4, 4, "question", newByteSequence([]byte(`?`))),
				newTokenDefault(5, 5, "vbar", newByteSequence([]byte(`|`))),
				newTokenDefault(6, 6, "lparen", newByteSequence([]byte(`(`))),
				newTokenDefault(7, 7, "rparen", newByteSequence([]byte(`)`))),
				newTokenDefault(8, 8, "lbrace", newByteSequence([]byte(`[`))),
				newTokenDefault(9, 9, "backslash", newByteSequence([]byte(`\`))),
				newEOFTokenDefault(),
			},
		},
	}
	for i, tt := range test {
		for compLv := compiler.CompressionLevelMin; compLv <= compiler.CompressionLevelMax; compLv++ {
			t.Run(fmt.Sprintf("#%v-%v", i, compLv), func(t *testing.T) {
				clspec, err := compiler.Compile(tt.lspec, compiler.CompressionLevel(compLv))
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				opts := []LexerOption{}
				if tt.passiveModeTran {
					opts = append(opts, DisableModeTransition())
				}
				lexer, err := NewLexer(clspec, strings.NewReader(tt.src), opts...)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, eTok := range tt.tokens {
					tok, err := lexer.Next()
					if err != nil {
						t.Log(err)
						break
					}
					testToken(t, eTok, tok, false)
					// t.Logf("token: ID: %v, Match: %+v Text: \"%v\", EOF: %v, Invalid: %v", tok.ID, tok.Match(), tok.Text(), tok.EOF, tok.Invalid)
					if tok.EOF {
						break
					}

					if tt.tran != nil {
						err := tt.tran(lexer, tok)
						if err != nil {
							t.Fatalf("unexpected error: %v", err)
						}
					}
				}
			})
		}
	}
}

func TestLexer_Next_WithPosition(t *testing.T) {
	lspec := &spec.LexSpec{
		Entries: []*spec.LexEntry{
			newLexEntryDefaultNOP("newline", `\u{000A}+`),
			newLexEntryDefaultNOP("any", `.`),
		},
	}

	clspec, err := compiler.Compile(lspec, compiler.CompressionLevel(compiler.CompressionLevelMax))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string([]byte{
		0x00,
		0x7F,
		0x0A,

		0xC2, 0x80,
		0xDF, 0xBF,
		0x0A,

		0xE0, 0xA0, 0x80,
		0xE0, 0xBF, 0xBF,
		0xE1, 0x80, 0x80,
		0xEC, 0xBF, 0xBF,
		0xED, 0x80, 0x80,
		0xED, 0x9F, 0xBF,
		0xEE, 0x80, 0x80,
		0xEF, 0xBF, 0xBF,
		0x0A,

		0xF0, 0x90, 0x80, 0x80,
		0xF0, 0xBF, 0xBF, 0xBF,
		0xF1, 0x80, 0x80, 0x80,
		0xF3, 0xBF, 0xBF, 0xBF,
		0xF4, 0x80, 0x80, 0x80,
		0xF4, 0x8F, 0xBF, 0xBF,
		0x0A,
		0x0A,
		0x0A,
	})

	expected := []*Token{
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0x00})), 0, 0),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0x7F})), 0, 1),
		withPos(newTokenDefault(1, 1, "newline", newByteSequence([]byte{0x0A})), 0, 2),

		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xC2, 0x80})), 1, 0),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xDF, 0xBF})), 1, 1),
		withPos(newTokenDefault(1, 1, "newline", newByteSequence([]byte{0x0A})), 1, 2),

		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xE0, 0xA0, 0x80})), 2, 0),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xE0, 0xBF, 0xBF})), 2, 1),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xE1, 0x80, 0x80})), 2, 2),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xEC, 0xBF, 0xBF})), 2, 3),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xED, 0x80, 0x80})), 2, 4),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xED, 0x9F, 0xBF})), 2, 5),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xEE, 0x80, 0x80})), 2, 6),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xEF, 0xBF, 0xBF})), 2, 7),
		withPos(newTokenDefault(1, 1, "newline", newByteSequence([]byte{0x0A})), 2, 8),

		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xF0, 0x90, 0x80, 0x80})), 3, 0),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xF0, 0xBF, 0xBF, 0xBF})), 3, 1),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xF1, 0x80, 0x80, 0x80})), 3, 2),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xF3, 0xBF, 0xBF, 0xBF})), 3, 3),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xF4, 0x80, 0x80, 0x80})), 3, 4),
		withPos(newTokenDefault(2, 2, "any", newByteSequence([]byte{0xF4, 0x8F, 0xBF, 0xBF})), 3, 5),

		// When a token contains multiple line breaks, the driver sets the token position to
		// the line number where a lexeme first appears.
		withPos(newTokenDefault(1, 1, "newline", newByteSequence([]byte{0x0A, 0x0A, 0x0A})), 3, 6),

		withPos(newEOFTokenDefault(), 0, 0),
	}

	lexer, err := NewLexer(clspec, strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, eTok := range expected {
		tok, err := lexer.Next()
		if err != nil {
			t.Fatal(err)
		}

		testToken(t, eTok, tok, true)

		if tok.EOF {
			break
		}
	}
}

func testToken(t *testing.T, expected, actual *Token, checkPosition bool) {
	t.Helper()

	if actual.ModeID != expected.ModeID ||
		actual.ModeName != expected.ModeName ||
		actual.KindID != expected.KindID ||
		actual.ModeKindID != expected.ModeKindID ||
		actual.KindName != expected.KindName ||
		!bytes.Equal(actual.Match(), expected.Match()) ||
		actual.EOF != expected.EOF ||
		actual.Invalid != expected.Invalid {
		t.Fatalf(`unexpected token; want: %v ("%v"), got: %v ("%v")`, expected, expected.Text(), actual, actual.Text())
	}

	if checkPosition {
		if actual.Row != expected.Row || actual.Col != expected.Col {
			t.Fatalf(`unexpected token; want: %v ("%v"), got: %v ("%v")`, expected, expected.Text(), actual, actual.Text())
		}
	}
}
