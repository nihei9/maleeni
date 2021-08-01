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
	// ModeID is an ID of a lex mode.
	ModeID spec.LexModeID

	// ModeName is a name of a lex mode.
	ModeName spec.LexModeName

	// KindID is an ID of a kind. This is unique among all modes.
	KindID spec.LexKindID

	// ModeKindID is an ID of a lexical kind. This is unique only within a mode.
	// Note that you need to use KindID field if you want to identify a kind across all modes.
	ModeKindID spec.LexModeKindID

	// KindName is a name of a lexical kind.
	KindName spec.LexKindName

	// When this field is true, it means the token is the EOF token.
	EOF bool

	// When this field is true, it means the token is an error token.
	Invalid bool

	// match is a byte sequence matched a pattern of a lexical specification.
	match byteSequence
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

func newEOFToken(modeID spec.LexModeID, modeName spec.LexModeName) *Token {
	return &Token{
		ModeID:     modeID,
		ModeName:   modeName,
		ModeKindID: 0,
		EOF:        true,
	}
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

func (t *Token) String() string {
	if t.Invalid {
		return fmt.Sprintf("!{mode id: %v, mode name: %v, text: %v, byte: %v}", t.ModeID, t.ModeName, t.Text(), t.Match())
	}
	if t.EOF {
		return "{eof}"
	}
	return fmt.Sprintf("{mode id: %v, mode name: %v, kind id: %v, mode kind id: %v, kind name: %v, text: %v, byte: %v}", t.ModeID, t.ModeName, t.KindID, t.ModeKindID, t.KindName, t.Text(), t.Match())
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
	return json.Marshal(struct {
		ModeID     int          `json:"mode_id"`
		ModeName   string       `json:"mode_name"`
		KindID     int          `json:"kind_id"`
		ModeKindID int          `json:"mode_kind_id"`
		KindName   string       `json:"kind_name"`
		Match      byteSequence `json:"match"`
		Text       string       `json:"text"`
		EOF        bool         `json:"eof"`
		Invalid    bool         `json:"invalid"`
	}{
		ModeID:     t.ModeID.Int(),
		ModeName:   t.ModeName.String(),
		KindID:     t.KindID.Int(),
		ModeKindID: t.ModeKindID.Int(),
		KindName:   t.KindName.String(),
		Match:      t.match,
		Text:       t.Text(),
		EOF:        t.EOF,
		Invalid:    t.Invalid,
	})
}

type LexerOption func(l *Lexer) error

func DisableModeTransition() LexerOption {
	return func(l *Lexer) error {
		l.passiveModeTran = true
		return nil
	}
}

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
	clspec          *spec.CompiledLexSpec
	src             []byte
	srcPtr          int
	tokBuf          []*Token
	modeStack       []spec.LexModeID
	passiveModeTran bool
	logger          log.Logger
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
		modeStack: []spec.LexModeID{
			clspec.InitialModeID,
		},
		passiveModeTran: false,
		logger:          log.NewNopLogger(),
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
    token buffer: %v`, l.Mode(), l.clspec.ModeNames[l.Mode()], l.srcPtr, l.tokBuf)

	if len(l.tokBuf) > 0 {
		tok := l.tokBuf[0]
		l.tokBuf = l.tokBuf[1:]
		l.logger.Log(`  Returns a buffered token:
    token: %v
    token buffer: %v`, tok, l.tokBuf)
		return tok, nil
	}

	tok, err := l.nextAndTransition()
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
		tok, err = l.nextAndTransition()
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

func (l *Lexer) nextAndTransition() (*Token, error) {
	tok, err := l.next()
	if err != nil {
		return nil, err
	}
	if tok.EOF || tok.Invalid {
		return tok, nil
	}
	if l.passiveModeTran {
		return tok, nil
	}
	spec := l.clspec.Specs[l.Mode()]
	if spec.Pop[tok.ModeKindID] == 1 {
		err := l.PopMode()
		if err != nil {
			return nil, err
		}
	}
	mode := spec.Push[tok.ModeKindID]
	if !mode.IsNil() {
		l.PushMode(mode)
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
	mode := l.Mode()
	modeName := l.clspec.ModeNames[mode]
	spec := l.clspec.Specs[mode]
	state := spec.DFA.InitialStateID
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
		modeKindID := spec.DFA.AcceptingStates[state]
		if modeKindID != 0 {
			kindID := l.clspec.KindIDs[mode][modeKindID]
			tok = newToken(mode, modeName, kindID, modeKindID, spec.KindNames[modeKindID], newByteSequence(buf))
			unfixedBufLen = 0
		}
	}
}

func (l *Lexer) lookupNextState(mode spec.LexModeID, state spec.StateID, v int) (spec.StateID, bool) {
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
		if next == spec.StateIDNil {
			return spec.StateIDNil, false
		}
		return next, true
	}
	modeSpec := l.clspec.Specs[mode]
	next := modeSpec.DFA.UncompressedTransition[state.Int()*modeSpec.DFA.ColCount+v]
	if next == spec.StateIDNil {
		return spec.StateIDNil, false
	}
	return next, true
}

func (l *Lexer) Mode() spec.LexModeID {
	return l.modeStack[len(l.modeStack)-1]
}

func (l *Lexer) PushMode(mode spec.LexModeID) {
	l.modeStack = append(l.modeStack, mode)
}

func (l *Lexer) PopMode() error {
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
