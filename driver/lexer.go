package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

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

	// Row is a row number where a lexeme appears.
	Row int

	// Col is a column number where a lexeme appears.
	// Note that Col is counted in code points, not bytes.
	Col int

	// When this field is true, it means the token is the EOF token.
	EOF bool

	// When this field is true, it means the token is an error token.
	Invalid bool

	// match is a byte sequence matched a pattern of a lexical specification.
	match byteSequence
}

func (t *Token) String() string {
	if t.Invalid {
		return fmt.Sprintf("!{mode id: %v, mode name: %v, row: %v, col: %v, text: %v, byte: %v}", t.ModeID, t.ModeName, t.Row, t.Col, t.Text(), t.Match())
	}
	if t.EOF {
		return fmt.Sprintf("{kind name: eof, row: %v, col: %v}", t.Row, t.Col)
	}
	return fmt.Sprintf("{mode id: %v, mode name: %v, kind id: %v, mode kind id: %v, kind name: %v, row: %v, col: %v, text: %v, byte: %v}", t.ModeID, t.ModeName, t.KindID, t.ModeKindID, t.KindName, t.Row, t.Col, t.Text(), t.Match())
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
		Row        int          `json:"row"`
		Col        int          `json:"col"`
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
		Row:        t.Row,
		Col:        t.Col,
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

type Lexer struct {
	clspec          *spec.CompiledLexSpec
	src             []byte
	srcPtr          int
	row             int
	col             int
	prevRow         int
	prevCol         int
	tokBuf          []*Token
	modeStack       []spec.LexModeID
	passiveModeTran bool
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
		row:    0,
		col:    0,
		modeStack: []spec.LexModeID{
			clspec.InitialModeID,
		},
		passiveModeTran: false,
	}
	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func (l *Lexer) Next() (*Token, error) {
	if len(l.tokBuf) > 0 {
		tok := l.tokBuf[0]
		l.tokBuf = l.tokBuf[1:]
		return tok, nil
	}

	tok, err := l.nextAndTransition()
	if err != nil {
		return nil, err
	}
	if !tok.Invalid {
		return tok, nil
	}
	errTok := tok
	for {
		tok, err = l.nextAndTransition()
		if err != nil {
			return nil, err
		}
		if !tok.Invalid {
			break
		}
		errTok.match = errTok.match.merge(tok.match)
	}
	l.tokBuf = append(l.tokBuf, tok)

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
	row := l.row
	col := l.col
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
				return &Token{
					ModeID:     mode,
					ModeName:   modeName,
					ModeKindID: 0,
					Row:        row,
					Col:        col,
					match:      newByteSequence(buf),
					Invalid:    true,
				}, nil
			}
			return &Token{
				ModeID:     mode,
				ModeName:   modeName,
				ModeKindID: 0,
				Row:        0,
				Col:        0,
				EOF:        true,
			}, nil
		}
		buf = append(buf, v)
		unfixedBufLen++
		nextState, ok := l.lookupNextState(mode, state, int(v))
		if !ok {
			if tok != nil {
				l.unread(unfixedBufLen)
				return tok, nil
			}
			return &Token{
				ModeID:     mode,
				ModeName:   modeName,
				ModeKindID: 0,
				Row:        row,
				Col:        col,
				match:      newByteSequence(buf),
				Invalid:    true,
			}, nil
		}
		state = nextState
		modeKindID := spec.DFA.AcceptingStates[state]
		if modeKindID != 0 {
			kindID := l.clspec.KindIDs[mode][modeKindID]
			tok = &Token{
				ModeID:     mode,
				ModeName:   modeName,
				KindID:     kindID,
				ModeKindID: modeKindID,
				KindName:   spec.KindNames[modeKindID],
				Row:        row,
				Col:        col,
				match:      newByteSequence(buf),
			}
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

	l.prevRow = l.row
	l.prevCol = l.col

	// Count the token positions.
	// The driver treats LF as the end of lines and counts columns in code points, not bytes.
	// To count in code points, we refer to the First Byte column in the Table 3-6.
	//
	// Reference:
	// - [Table 3-6] https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf > Table 3-6.  UTF-8 Bit Distribution
	if b < 128 {
		// 0x0A is LF.
		if b == 0x0A {
			l.row++
			l.col = 0
		} else {
			l.col++
		}
	} else if b>>5 == 6 || b>>4 == 14 || b>>3 == 30 {
		l.col++
	}

	return b, false
}

// You must not call this function consecutively to record the token position correctly.
func (l *Lexer) unread(n int) {
	l.srcPtr -= n

	l.row = l.prevRow
	l.col = l.prevCol
}
