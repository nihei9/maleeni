package spec

import (
	"fmt"
	"regexp"
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

type LexPattern string

func (p LexPattern) validate() error {
	if p == "" {
		return fmt.Errorf("pattern doesn't allow to be the empty string")
	}
	return nil
}

type LexEntry struct {
	Kind    LexKind    `json:"kind"`
	Pattern LexPattern `json:"pattern"`
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
		for _, e := range s.Entries {
			if _, exist := ks[e.Kind.String()]; exist {
				return fmt.Errorf("kinds `%v` are duplicates", e.Kind)
			}
			ks[e.Kind.String()] = struct{}{}
		}
	}
	return nil
}

type TransitionTable struct {
	InitialState    int         `json:"initial_state"`
	AcceptingStates map[int]int `json:"accepting_states"`
	Transition      [][]int     `json:"transition"`
}

type CompiledLexSpec struct {
	Kinds []LexKind        `json:"kinds"`
	DFA   *TransitionTable `json:"dfa"`
}
