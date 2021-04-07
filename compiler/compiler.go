package compiler

import (
	"io"
	"strings"

	"github.com/nihei9/maleeni/log"
	"github.com/nihei9/maleeni/spec"
)

type compilerOption func(c *compilerConfig) error

func EnableLogging(w io.Writer) compilerOption {
	return func(c *compilerConfig) error {
		logger, err := log.NewLogger(w)
		if err != nil {
			return err
		}
		c.logger = logger
		return nil
	}
}

type compilerConfig struct {
	logger log.Logger
}

func Compile(lexspec *spec.LexSpec, opts ...compilerOption) (*spec.CompiledLexSpec, error) {
	config := &compilerConfig{
		logger: log.NewNopLogger(),
	}
	for _, opt := range opts {
		err := opt(config)
		if err != nil {
			return nil, err
		}
	}

	var kinds []string
	var patterns map[int][]byte
	{
		kinds = append(kinds, "")
		patterns = map[int][]byte{}
		for i, e := range lexspec.Entries {
			kinds = append(kinds, e.Kind)
			patterns[i+1] = []byte(e.Pattern)
		}

		config.logger.Log("Patterns:")
		for i, p := range patterns {
			config.logger.Log("  #%v %v", i, string(p))
		}
	}

	var root astNode
	var symTab *symbolTable
	{
		var err error
		root, symTab, err = parse(patterns)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		printAST(&b, root, "", "", false)
		config.logger.Log("AST:\n%v", b.String())
	}

	var tranTab *spec.TransitionTable
	{
		dfa := genDFA(root, symTab)
		var err error
		tranTab, err = genTransitionTable(dfa)
		if err != nil {
			return nil, err
		}

		config.logger.Log(`DFA:
  States: %v states
  Initial State: %v`, len(tranTab.Transition), tranTab.InitialState)
		config.logger.Log("  Accepting States:")
		for state, symbol := range tranTab.AcceptingStates {
			config.logger.Log("    %v: %v", state, symbol)
		}
	}

	return &spec.CompiledLexSpec{
		Kinds: kinds,
		DFA:   tranTab,
	}, nil
}
