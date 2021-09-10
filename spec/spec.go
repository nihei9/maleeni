package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LexKindID represents an ID of a lexical kind and is unique across all modes.
type LexKindID int

const (
	LexKindIDNil = LexKindID(0)
	LexKindIDMin = LexKindID(1)
)

func (id LexKindID) Int() int {
	return int(id)
}

// LexModeKindID represents an ID of a lexical kind and is unique within a mode.
// Use LexKindID to identify a kind across all modes uniquely.
type LexModeKindID int

const (
	LexModeKindIDNil = LexModeKindID(0)
	LexModeKindIDMin = LexModeKindID(1)
)

func (id LexModeKindID) Int() int {
	return int(id)
}

// LexKindName represents a name of a lexical kind.
type LexKindName string

const LexKindNameNil = LexKindName("")

func (k LexKindName) String() string {
	return string(k)
}

func (k LexKindName) validate() error {
	if k == "" {
		return fmt.Errorf("kind doesn't allow to be the empty string")
	}
	if !lexKindNameRE.Match([]byte(k)) {
		return fmt.Errorf("kind must be %v", lexKindNamePattern)
	}
	return nil
}

const lexKindNamePattern = "[A-Za-z_][0-9A-Za-z_]*"

var lexKindNameRE = regexp.MustCompile(lexKindNamePattern)

// LexPattern represents a pattern of a lexeme.
// The pattern is written in regular expression.
type LexPattern string

func (p LexPattern) validate() error {
	if p == "" {
		return fmt.Errorf("pattern doesn't allow to be the empty string")
	}
	return nil
}

// LexModeID represents an ID of a lex mode.
type LexModeID int

const (
	LexModeIDNil     = LexModeID(0)
	LexModeIDDefault = LexModeID(1)
)

func (n LexModeID) String() string {
	return strconv.Itoa(int(n))
}

func (n LexModeID) Int() int {
	return int(n)
}

func (n LexModeID) IsNil() bool {
	return n == LexModeIDNil
}

// LexModeName represents a name of a lex mode.
type LexModeName string

const (
	LexModeNameNil     = LexModeName("")
	LexModeNameDefault = LexModeName("default")
)

func (m LexModeName) String() string {
	return string(m)
}

func (m LexModeName) validate() error {
	if m.isNil() || !lexModeNameRE.Match([]byte(m)) {
		return fmt.Errorf("mode must be %v", lexModeNamePattern)
	}
	return nil
}

func (m LexModeName) isNil() bool {
	return m == LexModeNameNil
}

const lexModeNamePattern = "[A-Za-z_][0-9A-Za-z_]*"

var lexModeNameRE = regexp.MustCompile(lexModeNamePattern)

type LexEntry struct {
	Kind     LexKindName   `json:"kind"`
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

// StateID represents an ID of a state of a transition table.
type StateID int

const (
	// StateIDNil represents an empty entry of a transition table.
	// When the driver reads this value, it raises an error meaning lexical analysis failed.
	StateIDNil = StateID(0)

	// StateIDMin is the minimum value of the state ID. All valid state IDs are represented as
	// sequential numbers starting from this value.
	StateIDMin = StateID(1)
)

func (id StateID) Int() int {
	return int(id)
}

type RowDisplacementTable struct {
	OriginalRowCount int       `json:"original_row_count"`
	OriginalColCount int       `json:"original_col_count"`
	EmptyValue       StateID   `json:"empty_value"`
	Entries          []StateID `json:"entries"`
	Bounds           []int     `json:"bounds"`
	RowDisplacement  []int     `json:"row_displacement"`
}

type UniqueEntriesTable struct {
	UniqueEntries             *RowDisplacementTable `json:"unique_entries,omitempty"`
	UncompressedUniqueEntries []StateID             `json:"uncompressed_unique_entries,omitempty"`
	RowNums                   []int                 `json:"row_nums"`
	OriginalRowCount          int                   `json:"original_row_count"`
	OriginalColCount          int                   `json:"original_col_count"`
	EmptyValue                int                   `json:"empty_value"`
}

type TransitionTable struct {
	InitialStateID         StateID             `json:"initial_state_id"`
	AcceptingStates        []LexModeKindID     `json:"accepting_states"`
	RowCount               int                 `json:"row_count"`
	ColCount               int                 `json:"col_count"`
	Transition             *UniqueEntriesTable `json:"transition,omitempty"`
	UncompressedTransition []StateID           `json:"uncompressed_transition,omitempty"`
}

type CompiledLexModeSpec struct {
	KindNames []LexKindName    `json:"kind_names"`
	Push      []LexModeID      `json:"push"`
	Pop       []int            `json:"pop"`
	DFA       *TransitionTable `json:"dfa"`
}

type CompiledLexSpec struct {
	InitialModeID    LexModeID              `json:"initial_mode_id"`
	ModeNames        []LexModeName          `json:"mode_names"`
	KindNames        []LexKindName          `json:"kind_names"`
	KindIDs          [][]LexKindID          `json:"kind_ids"`
	CompressionLevel int                    `json:"compression_level"`
	Specs            []*CompiledLexModeSpec `json:"specs"`
}
