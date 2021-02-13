package compiler

func Compile(regexps map[int][]byte) (*DFA, error) {
	root, symTab, err := parse(regexps)
	if err != nil {
		return nil, err
	}
	return genDFA(root, symTab), nil
}
