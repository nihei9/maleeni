package driver

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/nihei9/maleeni/spec"
)

type Token struct {
	ID      int
	Kind    string
	Match   []byte
	EOF     bool
	Invalid bool
}

func newToken(id int, kind string, match []byte) *Token {
	return &Token{
		ID:    id,
		Kind:  kind,
		Match: match,
	}
}

func newEOFToken() *Token {
	return &Token{
		ID:  0,
		EOF: true,
	}
}

func newInvalidToken(match []byte) *Token {
	return &Token{
		ID:      0,
		Match:   match,
		Invalid: true,
	}
}

type lexer struct {
	clspec *spec.CompiledLexSpec
	src    []byte
	srcPtr int
	tokBuf []*Token
}

func NewLexer(clspec *spec.CompiledLexSpec, src io.Reader) (*lexer, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	return &lexer{
		clspec: clspec,
		src:    b,
		srcPtr: 0,
	}, nil
}

func (l *lexer) Next() (*Token, error) {
	if len(l.tokBuf) > 0 {
		tok := l.tokBuf[0]
		l.tokBuf = l.tokBuf[1:]
		return tok, nil
	}

	tok, err := l.next()
	if err != nil {
		return nil, err
	}
	if !tok.Invalid {
		return tok, nil
	}
	errTok := tok
	for {
		tok, err = l.next()
		if err != nil {
			return nil, err
		}
		if !tok.Invalid {
			break
		}
		errTok.Match = append(errTok.Match, tok.Match...)
	}
	l.tokBuf = append(l.tokBuf, tok)
	return errTok, nil
}

func (l *lexer) Peek1() (*Token, error) {
	return l.peekN(0)
}

func (l *lexer) Peek2() (*Token, error) {
	return l.peekN(1)
}

func (l *lexer) Peek3() (*Token, error) {
	return l.peekN(2)
}

func (l *lexer) peekN(n int) (*Token, error) {
	if n < 0 || n > 2 {
		return nil, fmt.Errorf("peekN() can handle only [0..2]")
	}
	for len(l.tokBuf) < n+1 {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		l.tokBuf = append(l.tokBuf, tok)
	}
	return l.tokBuf[n], nil
}

func (l *lexer) next() (*Token, error) {
	state := l.clspec.DFA.InitialState
	buf := []byte{}
	unfixedBufLen := 0
	var tok *Token
	for {
		v, eof := l.read()
		if eof {
			if tok != nil {
				l.unread(unfixedBufLen)
				return tok, nil
			}
			return newEOFToken(), nil
		}
		buf = append(buf, v)
		unfixedBufLen++
		entry := l.clspec.DFA.Transition[state]
		if len(entry) == 0 {
			return nil, fmt.Errorf("no transition entry; state: %v", state)
		}
		nextState := entry[v]
		if nextState == 0 {
			if tok != nil {
				l.unread(unfixedBufLen)
				return tok, nil
			}
			return newInvalidToken(buf), nil
		}
		state = nextState
		id, ok := l.clspec.DFA.AcceptingStates[state]
		if ok {
			tok = newToken(id, l.clspec.Kinds[id], buf)
			unfixedBufLen = 0
		}
	}
}

func (l *lexer) read() (byte, bool) {
	if l.srcPtr >= len(l.src) {
		return 0, true
	}
	b := l.src[l.srcPtr]
	l.srcPtr++
	return b, false
}

func (l *lexer) unread(n int) {
	l.srcPtr -= n
}
