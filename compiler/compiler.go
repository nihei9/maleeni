package compiler

import (
	"fmt"
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
	err := lexspec.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid lexical specification:\n%w", err)
	}

	config := &compilerConfig{
		logger: log.NewNopLogger(),
	}
	for _, opt := range opts {
		err := opt(config)
		if err != nil {
			return nil, err
		}
	}

	modeEntries, modes, modeNums := groupEntriesByLexMode(lexspec.Entries)

	modeSpecs := []*spec.CompiledLexModeSpec{
		nil,
	}
	for i, es := range modeEntries[1:] {
		modeName := modes[i+1]
		config.logger.Log("Compile %v mode:", modeName)
		modeSpec, err := compile(es, modeNums, config)
		if err != nil {
			return nil, fmt.Errorf("failed to compile in %v mode: %w", modeName, err)
		}
		modeSpecs = append(modeSpecs, modeSpec)
	}

	return &spec.CompiledLexSpec{
		InitialMode: spec.LexModeNumDefault,
		Modes:       modes,
		Specs:       modeSpecs,
	}, nil
}

func groupEntriesByLexMode(entries []*spec.LexEntry) ([][]*spec.LexEntry, []spec.LexModeName, map[spec.LexModeName]spec.LexModeNum) {
	modes := []spec.LexModeName{
		spec.LexModeNameNil,
		spec.LexModeNameDefault,
	}
	modeNums := map[spec.LexModeName]spec.LexModeNum{
		spec.LexModeNameNil:     spec.LexModeNumNil,
		spec.LexModeNameDefault: spec.LexModeNumDefault,
	}
	lastModeNum := spec.LexModeNumDefault
	modeEntries := [][]*spec.LexEntry{
		nil,
		[]*spec.LexEntry{},
	}
	for _, e := range entries {
		ms := e.Modes
		if len(ms) == 0 {
			ms = []spec.LexModeName{
				spec.LexModeNameDefault,
			}
		}
		for _, mode := range ms {
			num, ok := modeNums[mode]
			if !ok {
				num = lastModeNum.Succ()
				lastModeNum = num
				modeNums[mode] = num
				modes = append(modes, mode)
				modeEntries = append(modeEntries, []*spec.LexEntry{})
			}
			modeEntries[num] = append(modeEntries[num], e)
		}
	}
	return modeEntries, modes, modeNums
}

func compile(entries []*spec.LexEntry, modeNums map[spec.LexModeName]spec.LexModeNum, config *compilerConfig) (*spec.CompiledLexModeSpec, error) {
	var kinds []spec.LexKind
	var patterns map[int][]byte
	{
		kinds = append(kinds, spec.LexKindNil)
		patterns = map[int][]byte{}
		for i, e := range entries {
			kinds = append(kinds, e.Kind)
			patterns[i+1] = []byte(e.Pattern)
		}

		config.logger.Log("Patterns:")
		for i, p := range patterns {
			config.logger.Log("  #%v %v", i, string(p))
		}
	}

	push := []spec.LexModeNum{
		spec.LexModeNumNil,
	}
	pop := []int{
		0,
	}
	for _, e := range entries {
		pushV := spec.LexModeNumNil
		if e.Push != "" {
			pushV = modeNums[e.Push]
		}
		push = append(push, pushV)
		popV := 0
		if e.Pop {
			popV = 1
		}
		pop = append(pop, popV)
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

	return &spec.CompiledLexModeSpec{
		Kinds: kinds,
		Push:  push,
		Pop:   pop,
		DFA:   tranTab,
	}, nil
}
