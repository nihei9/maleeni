package compiler

import "github.com/nihei9/maleeni/spec"

func Compile(lexspec *spec.LexSpec) (*spec.CompiledLexSpec, error) {
	var kinds []string
	var patterns map[int][]byte
	{
		kinds = append(kinds, "")
		patterns = map[int][]byte{}
		for i, e := range lexspec.Entries {
			kinds = append(kinds, e.Kind)
			patterns[i+1] = []byte(e.Pattern)
		}
	}
	root, symTab, err := parse(patterns)
	if err != nil {
		return nil, err
	}
	dfa := genDFA(root, symTab)
	tranTab, err := genTransitionTable(dfa)
	if err != nil {
		return nil, err
	}
	return &spec.CompiledLexSpec{
		Kinds: kinds,
		DFA:   tranTab,
	}, nil
}
