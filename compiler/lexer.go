package compiler

import (
	"bufio"
	"fmt"
	"io"
)

type tokenKind string

const (
	tokenKindChar            = tokenKind("char")
	tokenKindAnyChar         = tokenKind(".")
	tokenKindRepeat          = tokenKind("*")
	tokenKindRepeatOneOrMore = tokenKind("+")
	tokenKindOption          = tokenKind("?")
	tokenKindAlt             = tokenKind("|")
	tokenKindGroupOpen       = tokenKind("(")
	tokenKindGroupClose      = tokenKind(")")
	tokenKindBExpOpen        = tokenKind("[")
	tokenKindBExpClose       = tokenKind("]")
	tokenKindCharRange       = tokenKind("-")
	tokenKindEOF             = tokenKind("eof")
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

type lexerMode string

const (
	lexerModeDefault = lexerMode("default")
	lexerModeBExp    = lexerMode("bracket expression")
)

type lexer struct {
	src        *bufio.Reader
	lastChar   rune
	prevChar   rune
	reachedEOF bool
	mode       lexerMode
}

func newLexer(src io.Reader) *lexer {
	return &lexer{
		src:        bufio.NewReader(src),
		lastChar:   nullChar,
		prevChar:   nullChar,
		reachedEOF: false,
		mode:       lexerModeDefault,
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

	switch l.mode {
	case lexerModeBExp:
		return l.nextInBExp(c)
	default:
		return l.nextInDefault(c)
	}
}

func (l *lexer) nextInDefault(c rune) (*token, error) {
	switch c {
	case '*':
		return newToken(tokenKindRepeat, nullChar), nil
	case '+':
		return newToken(tokenKindRepeatOneOrMore, nullChar), nil
	case '?':
		return newToken(tokenKindOption, nullChar), nil
	case '.':
		return newToken(tokenKindAnyChar, nullChar), nil
	case '|':
		return newToken(tokenKindAlt, nullChar), nil
	case '(':
		return newToken(tokenKindGroupOpen, nullChar), nil
	case ')':
		return newToken(tokenKindGroupClose, nullChar), nil
	case '[':
		l.mode = lexerModeBExp
		return newToken(tokenKindBExpOpen, nullChar), nil
	case ']':
		return newToken(tokenKindBExpClose, nullChar), nil
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
		case c == '\\' || c == '.' || c == '*' || c == '+' || c == '?' || c == '|' || c == '(' || c == ')' || c == '[' || c == ']':
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

func (l *lexer) nextInBExp(c rune) (*token, error) {
	switch c {
	case '-':
		return newToken(tokenKindCharRange, nullChar), nil
	case ']':
		l.mode = lexerModeDefault
		return newToken(tokenKindBExpClose, nullChar), nil
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
		case c == '\\' || c == '-' || c == ']':
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
