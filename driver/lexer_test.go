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
		Kind:    spec.LexKind(kind),
		Pattern: spec.LexPattern(pattern),
		Modes:   ms,
		Push:    spec.LexModeName(push),
		Pop:     pop,
	}
}

func newLexEntryDefaultNOP(kind string, pattern string) *spec.LexEntry {
	return &spec.LexEntry{
		Kind:    spec.LexKind(kind),
		Pattern: spec.LexPattern(pattern),
		Modes: []spec.LexModeName{
			spec.LexModeNameDefault,
		},
	}
}

func newTokenDefault(id int, kind string, match byteSequence) *Token {
	return newToken(spec.LexModeNumDefault, spec.LexModeNameDefault, id, kind, match)
}

func newEOFTokenDefault() *Token {
	return newEOFToken(spec.LexModeNumDefault, spec.LexModeNameDefault)
}

func TestLexer_Next(t *testing.T) {
	test := []struct {
		lspec  *spec.LexSpec
		src    string
		tokens []*Token
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
				newTokenDefault(1, "t1", newByteSequence([]byte("abb"))),
				newTokenDefault(2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("aabb"))),
				newTokenDefault(2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("aaabb"))),
				newTokenDefault(2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("babb"))),
				newTokenDefault(2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("bbabb"))),
				newTokenDefault(2, "t2", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("abbbabb"))),
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
				newTokenDefault(1, "t1", newByteSequence([]byte("ba"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("baaa"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("a"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(1, "t1", newByteSequence([]byte("aaa"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, "t2", newByteSequence([]byte("abcd"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, "t2", newByteSequence([]byte("abcdcdcd"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, "t2", newByteSequence([]byte("cd"))),
				newTokenDefault(3, "t3", newByteSequence([]byte(" "))),
				newTokenDefault(2, "t2", newByteSequence([]byte("cdcdcd"))),
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
				newTokenDefault(1, "t1", newByteSequence([]byte{0x00})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0x7f})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xc2, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xdf, 0xbf})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xe1, 0x80, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xec, 0xbf, 0xbf})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xed, 0x80, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xed, 0x9f, 0xbf})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xee, 0x80, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xef, 0xbf, 0xbf})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xf1, 0x80, 0x80, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xf3, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xf4, 0x80, 0x80, 0x80})),
				newTokenDefault(1, "t1", newByteSequence([]byte{0xf4, 0x8f, 0xbf, 0xbf})),
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
				newTokenDefault(1, "t1", newByteSequence([]byte("a"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("b"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("."))),
				newTokenDefault(1, "t1", newByteSequence([]byte("*"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("+"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("?"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("|"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("("))),
				newTokenDefault(1, "t1", newByteSequence([]byte(")"))),
				newTokenDefault(1, "t1", newByteSequence([]byte("["))),
				newTokenDefault(1, "t1", newByteSequence([]byte("]"))),
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
				newTokenDefault(1, "1ByteChar", newByteSequence([]byte{0x01})),
				newTokenDefault(1, "1ByteChar", newByteSequence([]byte{0x02})),
				newTokenDefault(1, "1ByteChar", newByteSequence([]byte{0x7e})),
				newTokenDefault(1, "1ByteChar", newByteSequence([]byte{0x7f})),
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
				newTokenDefault(1, "2ByteChar", newByteSequence([]byte{0xc2, 0x80})),
				newTokenDefault(1, "2ByteChar", newByteSequence([]byte{0xc2, 0x81})),
				newTokenDefault(1, "2ByteChar", newByteSequence([]byte{0xdf, 0xbe})),
				newTokenDefault(1, "2ByteChar", newByteSequence([]byte{0xdf, 0xbf})),
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
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
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
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x81})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0xbe})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0xbf})),
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
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x81})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbe})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbf})),
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
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x80})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xa0, 0x81})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbe})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe0, 0xbf, 0xbf})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe1, 0x80, 0x80})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xe1, 0x80, 0x81})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xec, 0xbf, 0xbe})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xec, 0xbf, 0xbf})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xed, 0x80, 0x80})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xed, 0x80, 0x81})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xed, 0x9f, 0xbe})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xed, 0x9f, 0xbf})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xee, 0x80, 0x80})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xee, 0x80, 0x81})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xef, 0xbf, 0xbe})),
				newTokenDefault(1, "3ByteChar", newByteSequence([]byte{0xef, 0xbf, 0xbf})),
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
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
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
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0xbe})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0xbf})),
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
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0xbf, 0xbe})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0xbf, 0xbf})),
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
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbe})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbf})),
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
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x80})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0x90, 0x80, 0x81})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbe})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf0, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf1, 0x80, 0x80, 0x80})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf1, 0x80, 0x80, 0x81})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf3, 0xbf, 0xbf, 0xbe})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf3, 0xbf, 0xbf, 0xbf})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf4, 0x80, 0x80, 0x80})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf4, 0x80, 0x80, 0x81})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf4, 0x8f, 0xbf, 0xbe})),
				newTokenDefault(1, "4ByteChar", newByteSequence([]byte{0xf4, 0x8f, 0xbf, 0xbf})),
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
				newTokenDefault(1, "NonNumber", newByteSequence([]byte("foo9"))),
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
				newTokenDefault(1, "1ByteChar", newByteSequence([]byte{0x6E})),
				newTokenDefault(2, "2ByteChar", newByteSequence([]byte{0xCE, 0xBD})),
				newTokenDefault(3, "3ByteChar", newByteSequence([]byte{0xE3, 0x81, 0xAB})),
				newTokenDefault(4, "4ByteChar", newByteSequence([]byte{0xF0, 0x9F, 0x98, 0xB8})),
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
				newTokenDefault(1, "codePointsAlt", newByteSequence([]byte{0x6E})),
				newTokenDefault(1, "codePointsAlt", newByteSequence([]byte{0xCE, 0xBD})),
				newTokenDefault(1, "codePointsAlt", newByteSequence([]byte{0xE3, 0x81, 0xAB})),
				newTokenDefault(1, "codePointsAlt", newByteSequence([]byte{0xF0, 0x9F, 0x98, 0xB8})),
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
				newToken(1, "default", 2, "string_open", newByteSequence([]byte(`"`))),
				newToken(2, "string", 3, "string_close", newByteSequence([]byte(`"`))),
				newToken(1, "default", 1, "white_space", newByteSequence([]byte(` `))),
				newToken(1, "default", 2, "string_open", newByteSequence([]byte(`"`))),
				newToken(2, "string", 2, "char_sequence", newByteSequence([]byte(`Hello world.`))),
				newToken(2, "string", 1, "escape_sequence", newByteSequence([]byte(`\n`))),
				newToken(2, "string", 1, "escape_sequence", newByteSequence([]byte(`\"`))),
				newToken(2, "string", 2, "char_sequence", newByteSequence([]byte(`Hello world.`))),
				newToken(2, "string", 1, "escape_sequence", newByteSequence([]byte(`\"`))),
				newToken(2, "string", 3, "string_close", newByteSequence([]byte(`"`))),
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
				newToken(1, "default", 1, "white_space", newByteSequence([]byte(` `))),
				newToken(1, "default", 2, "char_a", newByteSequence([]byte(`a`))),
				newToken(2, "state_a", 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "state_a", 2, "char_b", newByteSequence([]byte(`b`))),
				newToken(3, "state_b", 1, "white_space", newByteSequence([]byte(` `))),
				newToken(3, "state_b", 2, "back_from_b", newByteSequence([]byte(`<`))),
				newToken(2, "state_a", 1, "white_space", newByteSequence([]byte(` `))),
				newToken(2, "state_a", 3, "back_from_a", newByteSequence([]byte(`<`))),
				newToken(1, "default", 1, "white_space", newByteSequence([]byte(` `))),
				newEOFTokenDefault(),
			},
		},
	}
	for i, tt := range test {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
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
		})
	}
}

func TestLexer_PeekN(t *testing.T) {
	clspec, err := compiler.Compile(&spec.LexSpec{
		Entries: []*spec.LexEntry{
			newLexEntryDefaultNOP("t1", "foo"),
			newLexEntryDefaultNOP("t2", "bar"),
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
		newTokenDefault(1, "t1", []byte("foo")),
		newTokenDefault(2, "t2", []byte("bar")),
		newEOFTokenDefault(),
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

	if actual.Mode != expected.Mode ||
		actual.ModeName != actual.ModeName ||
		actual.ID != expected.ID ||
		actual.Kind != expected.Kind ||
		!bytes.Equal(actual.Match, expected.Match) ||
		actual.EOF != expected.EOF ||
		actual.Invalid != expected.Invalid {
		t.Fatalf(`unexpected token; want: %v ("%v"), got: %v ("%v")`, expected, string(expected.Match), actual, string(actual.Match))
	}
}
