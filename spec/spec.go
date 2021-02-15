package spec

type LexEntry struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
}

func NewLexEntry(kind string, pattern string) *LexEntry {
	return &LexEntry{
		Kind:    kind,
		Pattern: pattern,
	}
}

type LexSpec struct {
	Entries []*LexEntry `json:"entries"`
}

type TransitionTable struct {
	InitialState    int         `json:"initial_state"`
	AcceptingStates map[int]int `json:"accepting_states"`
	Transition      [][]int     `json:"transition"`
}

type CompiledLexSpec struct {
	Kinds []string         `json:"kinds"`
	DFA   *TransitionTable `json:"dfa"`
}
