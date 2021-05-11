package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/nihei9/maleeni/log"
	"github.com/nihei9/maleeni/spec"
)

type byteSequence []byte

func newByteSequence(b []byte) byteSequence {
	return byteSequence(b)
}

func (s byteSequence) ByteSlice() []byte {
	return []byte(s)
}

func (s byteSequence) String() string {
	if len(s) <= 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%X", s[0])
	for _, d := range s[1:] {
		fmt.Fprintf(&b, " %X", d)
	}
	return b.String()
}

func (s byteSequence) GoString() string {
	return fmt.Sprintf("\"%v\"", s.String())
}

func (s byteSequence) MarshalJSON() ([]byte, error) {
	if len(s) <= 0 {
		return []byte("[]"), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[%v", uint8(s[0]))
	for _, e := range s[1:] {
		fmt.Fprintf(&b, ", %v", uint8(e))
	}
	fmt.Fprintf(&b, "]")
	return []byte(b.String()), nil
}

func (s byteSequence) merge(a byteSequence) byteSequence {
	return append([]byte(s), []byte(a)...)
}

// Token representes a token.
type Token struct {
	// `Mode` represents a number that corresponds to a `ModeName`.
	Mode spec.LexModeNum

	// `ModeName` is a mode name that represents in which mode the lexer detected the token.
	ModeName spec.LexModeName

	// `ID` represents an ID that corresponds to a `Kind`.
	ID int

	// `Kind` is a kind name that represents what kind the token has.
	Kind string

	// If `EOF` is true, it means the token is the EOF token.
	EOF bool

	// If `Invalid` is true, it means the token is an error token.
	Invalid bool

	// `match` is a byte sequence matched a pattern of a lexical specification.
	match byteSequence
}

func newToken(mode spec.LexModeNum, modeName spec.LexModeName, id int, kind string, match byteSequence) *Token {
	return &Token{
		Mode:     mode,
		ModeName: modeName,
		ID:       id,
		Kind:     kind,
		match:    match,
	}
}

func newEOFToken(mode spec.LexModeNum, modeName spec.LexModeName) *Token {
	return &Token{
		Mode:     mode,
		ModeName: modeName,
		ID:       0,
		EOF:      true,
	}
}

func newInvalidToken(mode spec.LexModeNum, modeName spec.LexModeName, match byteSequence) *Token {
	return &Token{
		Mode:     mode,
		ModeName: modeName,
		ID:       0,
		match:    match,
		Invalid:  true,
	}
}

func (t *Token) String() string {
	if t.Invalid {
		return fmt.Sprintf("!{mode: %v, mode name: %v, text: %v, byte: %v}", t.Mode, t.ModeName, t.Text(), t.Match())
	}
	if t.EOF {
		return "{eof}"
	}
	return fmt.Sprintf("{mode: %v, mode name: %v, id: %v, kind: %v, text: %v, byte: %v}", t.Mode, t.ModeName, t.ID, t.Kind, t.Text(), t.Match())
}

// Match returns a byte slice matched a pattern of a lexical specification.
func (t *Token) Match() []byte {
	return t.match.ByteSlice()
}

// Text returns a string representation of a matched byte sequence.
func (t *Token) Text() string {
	return string(t.Match())
}

func (t *Token) MarshalJSON() ([]byte, error) {
	m := t.match.ByteSlice()
	return json.Marshal(struct {
		Mode     int    `json:"mode"`
		ModeName string `json:"mode_name"`
		ID       int    `json:"id"`
		Kind     string `json:"kind"`
		Match    []byte `json:"match"`
		Text     string `json:"text"`
		EOF      bool   `json:"eof"`
		Invalid  bool   `json:"invalid"`
	}{
		Mode:     t.Mode.Int(),
		ModeName: t.ModeName.String(),
		ID:       t.ID,
		Kind:     t.Kind,
		Match:    m,
		Text:     string(m),
		EOF:      t.EOF,
		Invalid:  t.Invalid,
	})
}

type LexerOption func(l *Lexer) error

func EnableLogging(w io.Writer) LexerOption {
	return func(l *Lexer) error {
		logger, err := log.NewLogger(w)
		if err != nil {
			return err
		}
		l.logger = logger
		return nil
	}
}

type Lexer struct {
	clspec    *spec.CompiledLexSpec
	src       []byte
	srcPtr    int
	tokBuf    []*Token
	modeStack []spec.LexModeNum
	logger    log.Logger
}

func NewLexer(clspec *spec.CompiledLexSpec, src io.Reader, opts ...LexerOption) (*Lexer, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	l := &Lexer{
		clspec: clspec,
		src:    b,
		srcPtr: 0,
		modeStack: []spec.LexModeNum{
			clspec.InitialMode,
		},
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

func (l *Lexer) Next() (*Token, error) {
	l.logger.Log(`lexer#Next():
  State:
    mode: #%v %v
    pointer: %v
    token buffer: %v`, l.mode(), l.clspec.Modes[l.mode()], l.srcPtr, l.tokBuf)

	if len(l.tokBuf) > 0 {
		tok := l.tokBuf[0]
		l.tokBuf = l.tokBuf[1:]
		l.logger.Log(`  Returns a buffered token:
    token: %v
    token buffer: %v`, tok, l.tokBuf)
		return tok, nil
	}

	tok, err := l.nextAndTranMode()
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
		tok, err = l.nextAndTranMode()
		if err != nil {
			l.logger.Log("  Detectes an error: %v", err)
			return nil, err
		}
		l.logger.Log("  Detects a token: %v", tok)
		if !tok.Invalid {
			break
		}
		errTok.match = errTok.match.merge(tok.match)
		l.logger.Log("  error token: %v", errTok)
	}
	l.tokBuf = append(l.tokBuf, tok)
	l.logger.Log(`  Returns a token:
    token: %v
    token buffer: %v`, errTok, l.tokBuf)

	return errTok, nil
}

func (l *Lexer) nextAndTranMode() (*Token, error) {
	tok, err := l.next()
	if err != nil {
		return nil, err
	}
	if tok.EOF || tok.Invalid {
		return tok, nil
	}
	spec := l.clspec.Specs[l.mode()]
	if spec.Pop[tok.ID] == 1 {
		err := l.popMode()
		if err != nil {
			return nil, err
		}
	}
	mode := spec.Push[tok.ID]
	if !mode.IsNil() {
		l.pushMode(mode)
	}
	// The checking length of the mode stack must be at after pop and push operations
	// because those operations can be performed at the same time.
	// When the mode stack has just one element and popped it, the mode stack will be temporarily emptied.
	// However, since a push operation may be performed immediately after it,
	// the lexer allows the stack to be temporarily empty.
	if len(l.modeStack) == 0 {
		return nil, fmt.Errorf("a mode stack must have at least one element")
	}
	return tok, nil
}

func (l *Lexer) next() (*Token, error) {
	mode := l.mode()
	modeName := l.clspec.Modes[mode]
	spec := l.clspec.Specs[mode]
	state := spec.DFA.InitialState
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
			// When `buf` has unaccepted data and reads the EOF,
			// the lexer treats the buffered data as an invalid token.
			if len(buf) > 0 {
				return newInvalidToken(mode, modeName, newByteSequence(buf)), nil
			}
			return newEOFToken(mode, modeName), nil
		}
		buf = append(buf, v)
		unfixedBufLen++
		nextState, ok := l.lookupNextState(mode, state, int(v))
		if !ok {
			if tok != nil {
				l.unread(unfixedBufLen)
				return tok, nil
			}
			return newInvalidToken(mode, modeName, newByteSequence(buf)), nil
		}
		state = nextState
		id := spec.DFA.AcceptingStates[state]
		if id != 0 {
			tok = newToken(mode, modeName, id, spec.Kinds[id].String(), newByteSequence(buf))
			unfixedBufLen = 0
		}
	}
}

func (l *Lexer) lookupNextState(mode spec.LexModeNum, state int, v int) (int, bool) {
	switch l.clspec.CompressionLevel {
	case 2:
		tab := l.clspec.Specs[mode].DFA.Transition
		rowNum := tab.RowNums[state]
		d := tab.UniqueEntries.RowDisplacement[rowNum]
		if tab.UniqueEntries.Bounds[d+v] != rowNum {
			return tab.UniqueEntries.EmptyValue, false
		}
		return tab.UniqueEntries.Entries[d+v], true
	case 1:
		tab := l.clspec.Specs[mode].DFA.Transition
		next := tab.UncompressedUniqueEntries[tab.RowNums[state]*tab.OriginalColCount+v]
		if next == 0 {
			return 0, false
		}
		return next, true
	}
	spec := l.clspec.Specs[mode]
	next := spec.DFA.UncompressedTransition[state*spec.DFA.ColCount+v]
	if next == 0 {
		return 0, false
	}
	return next, true
}

func (l *Lexer) mode() spec.LexModeNum {
	return l.modeStack[len(l.modeStack)-1]
}

func (l *Lexer) pushMode(mode spec.LexModeNum) {
	l.modeStack = append(l.modeStack, mode)
}

func (l *Lexer) popMode() error {
	sLen := len(l.modeStack)
	if sLen == 0 {
		return fmt.Errorf("cannot pop a lex mode from a lex mode stack any more")
	}
	l.modeStack = l.modeStack[:sLen-1]
	return nil
}

func (l *Lexer) read() (byte, bool) {
	if l.srcPtr >= len(l.src) {
		return 0, true
	}
	b := l.src[l.srcPtr]
	l.srcPtr++
	return b, false
}

func (l *Lexer) unread(n int) {
	l.srcPtr -= n
}
