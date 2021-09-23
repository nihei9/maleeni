package spec

import (
	"fmt"
	"regexp"
	"sort"
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
	err := validateIdentifier(k.String())
	if err != nil {
		return fmt.Errorf("invalid kind name: %v", err)
	}
	return nil
}

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
	err := validateIdentifier(m.String())
	if err != nil {
		return fmt.Errorf("invalid mode name: %v", err)
	}
	return nil
}

const idPattern = `^[a-z](_?[0-9a-z]+)*$`

var idRE = regexp.MustCompile(idPattern)

func validateIdentifier(id string) error {
	if id == "" {
		return fmt.Errorf("identifier doesn't allow to be the empty string")
	}
	if !idRE.MatchString(id) {
		return fmt.Errorf("identifier must be %v", idPattern)
	}
	return nil
}

func SnakeCaseToUpperCamelCase(snake string) string {
	elems := strings.Split(snake, "_")
	for i, e := range elems {
		if len(e) == 0 {
			continue
		}
		elems[i] = strings.ToUpper(string(e[0])) + e[1:]
	}

	return strings.Join(elems, "")
}

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
	Name    string      `json:"name"`
	Entries []*LexEntry `json:"entries"`
}

func (s *LexSpec) Validate() error {
	err := validateIdentifier(s.Name)
	if err != nil {
		return fmt.Errorf("invalid specification name: %v", err)
	}

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
	{
		kinds := []string{}
		modes := []string{
			LexModeNameDefault.String(), // This is a predefined mode.
		}
		for _, e := range s.Entries {
			if e.Fragment {
				continue
			}

			kinds = append(kinds, e.Kind.String())

			for _, m := range e.Modes {
				modes = append(modes, m.String())
			}
		}

		kindErrs := findSpellingInconsistenciesErrors(kinds, nil)
		modeErrs := findSpellingInconsistenciesErrors(modes, func(ids []string) error {
			if SnakeCaseToUpperCamelCase(ids[0]) == SnakeCaseToUpperCamelCase(LexModeNameDefault.String()) {
				var b strings.Builder
				fmt.Fprintf(&b, "%+v", ids[0])
				for _, id := range ids[1:] {
					fmt.Fprintf(&b, ", %+v", id)
				}
				return fmt.Errorf("these identifiers are treated as the same. please use the same spelling as predefined '%v': %v", LexModeNameDefault, b.String())
			}
			return nil
		})
		errs := append(kindErrs, modeErrs...)
		if len(errs) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "%v", errs[0])
			for _, err := range errs[1:] {
				fmt.Fprintf(&b, "\n%v", err)
			}
			return fmt.Errorf(b.String())
		}
	}

	return nil
}

func findSpellingInconsistenciesErrors(ids []string, hook func(ids []string) error) []error {
	duplicated := FindSpellingInconsistencies(ids)
	if len(duplicated) == 0 {
		return nil
	}

	var errs []error
	for _, dup := range duplicated {
		err := hook(dup)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		var b strings.Builder
		fmt.Fprintf(&b, "%+v", dup[0])
		for _, id := range dup[1:] {
			fmt.Fprintf(&b, ", %+v", id)
		}
		err = fmt.Errorf("these identifiers are treated as the same. please use the same spelling: %v", b.String())
		errs = append(errs, err)
	}

	return errs
}

// FindSpellingInconsistencies finds spelling inconsistencies in identifiers. The identifiers are considered to be the same
// if they are spelled the same when expressed in UpperCamelCase. For example, `left_paren` and `LeftParen` are spelled the same
// in UpperCamelCase. Thus they are considere to be spelling inconsistency.
func FindSpellingInconsistencies(ids []string) [][]string {
	m := map[string][]string{}
	for _, id := range removeDuplicates(ids) {
		c := SnakeCaseToUpperCamelCase(id)
		m[c] = append(m[c], id)
	}

	var duplicated [][]string
	for _, camels := range m {
		if len(camels) == 1 {
			continue
		}
		duplicated = append(duplicated, camels)
	}

	for _, dup := range duplicated {
		sort.Slice(dup, func(i, j int) bool {
			return dup[i] < dup[j]
		})
	}
	sort.Slice(duplicated, func(i, j int) bool {
		return duplicated[i][0] < duplicated[j][0]
	})

	return duplicated
}

func removeDuplicates(s []string) []string {
	m := map[string]struct{}{}
	for _, v := range s {
		m[v] = struct{}{}
	}

	var unique []string
	for v := range m {
		unique = append(unique, v)
	}

	return unique
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
	Name             string                 `json:"name"`
	InitialModeID    LexModeID              `json:"initial_mode_id"`
	ModeNames        []LexModeName          `json:"mode_names"`
	KindNames        []LexKindName          `json:"kind_names"`
	KindIDs          [][]LexKindID          `json:"kind_ids"`
	CompressionLevel int                    `json:"compression_level"`
	Specs            []*CompiledLexModeSpec `json:"specs"`
}
