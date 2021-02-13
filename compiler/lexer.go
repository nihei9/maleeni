package compiler

import (
	"bufio"
	"fmt"
	"io"
)

type tokenKind string

const (
	tokenKindChar       = tokenKind("char")
	tokenKindRepeat     = tokenKind("*")
	tokenKindAlt        = tokenKind("|")
	tokenKindGroupOpen  = tokenKind("(")
	tokenKindGroupClose = tokenKind(")")
	tokenKindEOF        = tokenKind("eof")
)

type token struct {
	kind tokenKind
	char rune
}

const nullChar = '\u0000'

func newToken(kind tokenKind, char rune) *token {
	return &token{
		kind: kind,
		char: char,
	}
}

type lexer struct {
	src        *bufio.Reader
	lastChar   rune
	prevChar   rune
	reachedEOF bool
}

func newLexer(src io.Reader) *lexer {
	return &lexer{
		src:        bufio.NewReader(src),
		lastChar:   nullChar,
		prevChar:   nullChar,
		reachedEOF: false,
	}
}

func (l *lexer) next() (*token, error) {
	c, eof, err := l.read()
	if err != nil {
		return nil, err
	}
	if eof {
		return newToken(tokenKindEOF, nullChar), nil
	}

	switch c {
	case '*':
		return newToken(tokenKindRepeat, nullChar), nil
	case '|':
		return newToken(tokenKindAlt, nullChar), nil
	case '(':
		return newToken(tokenKindGroupOpen, nullChar), nil
	case ')':
		return newToken(tokenKindGroupClose, nullChar), nil
	case '\\':
		c, eof, err := l.read()
		if err != nil {
			return nil, err
		}
		if eof {
			return nil, &SyntaxError{
				message: "incompleted escape sequence; unexpected EOF follows \\ character",
			}
		}
		switch {
		case c == '\\' || c == '*' || c == '|' || c == '(' || c == ')':
			return newToken(tokenKindChar, c), nil
		default:
			return nil, &SyntaxError{
				message: fmt.Sprintf("invalid escape sequence '\\%s'", string(c)),
			}
		}
	default:
		return newToken(tokenKindChar, c), nil
	}
}

func (l *lexer) read() (rune, bool, error) {
	c, _, err := l.src.ReadRune()
	if err != nil {
		if err == io.EOF {
			l.prevChar = l.lastChar
			l.lastChar = nullChar
			l.reachedEOF = true
			return nullChar, true, nil
		}
		return nullChar, false, err
	}
	l.prevChar = l.lastChar
	l.lastChar = c
	return c, false, nil
}

func (l *lexer) restore() error {
	if l.reachedEOF {
		l.lastChar = l.prevChar
		l.prevChar = nullChar
		l.reachedEOF = false
		return l.src.UnreadRune()
	}
	if l.lastChar == nullChar {
		return fmt.Errorf("the lexer failed to call restore() because the last character is null")
	}
	l.lastChar = l.prevChar
	l.prevChar = nullChar
	return l.src.UnreadRune()
}
