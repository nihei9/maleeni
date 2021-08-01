package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const lexKindPattern = "[A-Za-z_][0-9A-Za-z_]*"

var lexKindRE = regexp.MustCompile(lexKindPattern)

type LexKind string

const LexKindNil = LexKind("")

func (k LexKind) String() string {
	return string(k)
}

func (k LexKind) validate() error {
	if k == "" {
		return fmt.Errorf("kind doesn't allow to be the empty string")
	}
	if !lexKindRE.Match([]byte(k)) {
		return fmt.Errorf("kind must be %v", lexKindPattern)
	}
	return nil
}

// LexKindID is a unique ID among modes.
type LexKindID int

func (id LexKindID) Int() int {
	return int(id)
}

const (
	LexKindIDNil = LexKindID(0)
	LexKindIDMin = LexKindID(1)
)

type LexPattern string

func (p LexPattern) validate() error {
	if p == "" {
		return fmt.Errorf("pattern doesn't allow to be the empty string")
	}
	return nil
}

const lexModePattern = "[A-Za-z_][0-9A-Za-z_]*"

var lexModeRE = regexp.MustCompile(lexKindPattern)

type LexModeName string

const (
	LexModeNameNil     = LexModeName("")
	LexModeNameDefault = LexModeName("default")
)

func (m LexModeName) String() string {
	return string(m)
}

func (m LexModeName) validate() error {
	if m.isNil() || !lexModeRE.Match([]byte(m)) {
		return fmt.Errorf("mode must be %v", lexModePattern)
	}
	return nil
}

func (m LexModeName) isNil() bool {
	return m == LexModeNameNil
}

type LexModeNum int

const (
	LexModeNumNil     = LexModeNum(0)
	LexModeNumDefault = LexModeNum(1)
)

func (n LexModeNum) String() string {
	return strconv.Itoa(int(n))
}

func (n LexModeNum) Int() int {
	return int(n)
}

func (n LexModeNum) Succ() LexModeNum {
	return n + 1
}

func (n LexModeNum) IsNil() bool {
	return n == LexModeNumNil
}

type LexEntry struct {
	Kind     LexKind       `json:"kind"`
	Pattern  LexPattern    `json:"pattern"`
	Modes    []LexModeName `json:"modes"`
	Push     LexModeName   `json:"push"`
	Pop      bool          `json:"pop"`
	Fragment bool          `json:"fragment"`
}

func (e *LexEntry) validate() error {
	err := e.Kind.validate()
	if err != nil {
		return err
	}
	err = e.Pattern.validate()
	if err != nil {
		return err
	}
	if len(e.Modes) > 0 {
		for _, mode := range e.Modes {
			err = mode.validate()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type LexSpec struct {
	Entries []*LexEntry `json:"entries"`
}

func (s *LexSpec) Validate() error {
	if len(s.Entries) <= 0 {
		return fmt.Errorf("the lexical specification must have at least one entry")
	}
	{
		var errs []error
		for i, e := range s.Entries {
			err := e.validate()
			if err != nil {
				errs = append(errs, fmt.Errorf("entry #%v: %w", i+1, err))
			}
		}
		if len(errs) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "%v", errs[0])
			for _, err := range errs[1:] {
				fmt.Fprintf(&b, "\n%v", err)
			}
			return fmt.Errorf(b.String())
		}
	}
	{
		ks := map[string]struct{}{}
		fks := map[string]struct{}{}
		for _, e := range s.Entries {
			// Allow duplicate names between fragments and non-fragments.
			if e.Fragment {
				if _, exist := fks[e.Kind.String()]; exist {
					return fmt.Errorf("kinds `%v` are duplicates", e.Kind)
				}
				fks[e.Kind.String()] = struct{}{}
			} else {
				if _, exist := ks[e.Kind.String()]; exist {
					return fmt.Errorf("kinds `%v` are duplicates", e.Kind)
				}
				ks[e.Kind.String()] = struct{}{}
			}
		}
	}
	return nil
}

type RowDisplacementTable struct {
	OriginalRowCount int   `json:"original_row_count"`
	OriginalColCount int   `json:"original_col_count"`
	EmptyValue       int   `json:"empty_value"`
	Entries          []int `json:"entries"`
	Bounds           []int `json:"bounds"`
	RowDisplacement  []int `json:"row_displacement"`
}

type UniqueEntriesTable struct {
	UniqueEntries             *RowDisplacementTable `json:"unique_entries,omitempty"`
	UncompressedUniqueEntries []int                 `json:"uncompressed_unique_entries,omitempty"`
	RowNums                   []int                 `json:"row_nums"`
	OriginalRowCount          int                   `json:"original_row_count"`
	OriginalColCount          int                   `json:"original_col_count"`
	EmptyValue                int                   `json:"empty_value"`
}

type TransitionTable struct {
	InitialState           int                 `json:"initial_state"`
	AcceptingStates        []int               `json:"accepting_states"`
	RowCount               int                 `json:"row_count"`
	ColCount               int                 `json:"col_count"`
	Transition             *UniqueEntriesTable `json:"transition,omitempty"`
	UncompressedTransition []int               `json:"uncompressed_transition,omitempty"`
}

type CompiledLexModeSpec struct {
	Kinds []LexKind        `json:"kinds"`
	Push  []LexModeNum     `json:"push"`
	Pop   []int            `json:"pop"`
	DFA   *TransitionTable `json:"dfa"`
}

type CompiledLexSpec struct {
	InitialMode      LexModeNum             `json:"initial_mode"`
	Modes            []LexModeName          `json:"modes"`
	Kinds            []LexKind              `json:"kinds"`
	KindIDs          [][]LexKindID          `json:"kind_ids"`
	CompressionLevel int                    `json:"compression_level"`
	Specs            []*CompiledLexModeSpec `json:"specs"`
}
