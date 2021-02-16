package driver

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/nihei9/maleeni/log"
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

func (t *Token) String() string {
	if t.Invalid {
		return fmt.Sprintf("!{text: %v, byte: %v}", string(t.Match), t.Match)
	}
	if t.EOF {
		return "{eof}"
	}
	return fmt.Sprintf("{id: %v, kind: %v, text: %v, byte: %v}", t.ID, t.Kind, string(t.Match), t.Match)
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

type lexerOption func(l *lexer) error

func EnableLogging(w io.Writer) lexerOption {
	return func(l *lexer) error {
		logger, err := log.NewLogger(w)
		if err != nil {
			return err
		}
		l.logger = logger
		return nil
	}
}

type lexer struct {
	clspec *spec.CompiledLexSpec
	src    []byte
	srcPtr int
	tokBuf []*Token
	logger log.Logger
}

func NewLexer(clspec *spec.CompiledLexSpec, src io.Reader, opts ...lexerOption) (*lexer, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	l := &lexer{
		clspec: clspec,
		src:    b,
		srcPtr: 0,
		logger: log.NewNopLogger(),
	}
	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			return nil, err
		}
	}
	l.logger.Log("Initializing the lexer finished.")

	return l, nil
}

func (l *lexer) Next() (*Token, error) {
	l.logger.Log(`lexer#Next():
  State:
    pointer: %v
    token buffer: %v`, l.srcPtr, l.tokBuf)

	if len(l.tokBuf) > 0 {
		tok := l.tokBuf[0]
		l.tokBuf = l.tokBuf[1:]
		l.logger.Log(`  Returns a buffered token:
    token: %v
    token buffer: %v`, tok, l.tokBuf)
		return tok, nil
	}

	tok, err := l.next()
	if err != nil {
		l.logger.Log("  Detectes an error: %v", err)
		return nil, err
	}
	l.logger.Log("  Detects a token: %v", tok)
	if !tok.Invalid {
		l.logger.Log(`  Returns a token:
    token: %v
    token buffer: %v`, tok, l.tokBuf)
		return tok, nil
	}
	errTok := tok
	for {
		tok, err = l.next()
		if err != nil {
			l.logger.Log("  Detectes an error: %v", err)
			return nil, err
		}
		l.logger.Log("  Detects a token: %v", tok)
		if !tok.Invalid {
			break
		}
		errTok.Match = append(errTok.Match, tok.Match...)
		l.logger.Log("  error token: %v", errTok)
	}
	l.tokBuf = append(l.tokBuf, tok)
	l.logger.Log(`  Returns a token:
    token: %v
    token buffer: %v`, errTok, l.tokBuf)

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
