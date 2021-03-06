package compiler

import (
	"fmt"
	"io"
	"strings"

	"github.com/nihei9/maleeni/compressor"
	"github.com/nihei9/maleeni/log"
	"github.com/nihei9/maleeni/spec"
)

type CompilerOption func(c *compilerConfig) error

func EnableLogging(w io.Writer) CompilerOption {
	return func(c *compilerConfig) error {
		logger, err := log.NewLogger(w)
		if err != nil {
			return err
		}
		c.logger = logger
		return nil
	}
}

func CompressionLevel(lv int) CompilerOption {
	return func(c *compilerConfig) error {
		if lv < CompressionLevelMin || lv > CompressionLevelMax {
			return fmt.Errorf("compression level must be %v to %v", CompressionLevelMin, CompressionLevelMax)
		}
		c.compLv = lv
		return nil
	}
}

type compilerConfig struct {
	logger log.Logger
	compLv int
}

func Compile(lexspec *spec.LexSpec, opts ...CompilerOption) (*spec.CompiledLexSpec, error) {
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

	modeEntries, modes, modeNums, fragmetns := groupEntriesByLexMode(lexspec.Entries)

	modeSpecs := []*spec.CompiledLexModeSpec{
		nil,
	}
	for i, es := range modeEntries[1:] {
		modeName := modes[i+1]
		config.logger.Log("Compile %v mode:", modeName)
		modeSpec, err := compile(es, modeNums, fragmetns, config)
		if err != nil {
			return nil, fmt.Errorf("failed to compile in %v mode: %w", modeName, err)
		}
		modeSpecs = append(modeSpecs, modeSpec)
	}

	return &spec.CompiledLexSpec{
		InitialMode:      spec.LexModeNumDefault,
		Modes:            modes,
		CompressionLevel: config.compLv,
		Specs:            modeSpecs,
	}, nil
}

func groupEntriesByLexMode(entries []*spec.LexEntry) ([][]*spec.LexEntry, []spec.LexModeName, map[spec.LexModeName]spec.LexModeNum, map[string]*spec.LexEntry) {
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
		{},
	}
	fragments := map[string]*spec.LexEntry{}
	for _, e := range entries {
		if e.Fragment {
			fragments[e.Kind.String()] = e
			continue
		}
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
	return modeEntries, modes, modeNums, fragments
}

func compile(entries []*spec.LexEntry, modeNums map[spec.LexModeName]spec.LexModeNum, fragments map[string]*spec.LexEntry, config *compilerConfig) (*spec.CompiledLexModeSpec, error) {
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

	fragmentPatterns := map[string][]byte{}
	for k, e := range fragments {
		fragmentPatterns[k] = []byte(e.Pattern)
	}

	var root astNode
	var symTab *symbolTable
	{
		var err error
		root, symTab, err = parse(patterns, fragmentPatterns)
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
  States: %v states (%v entries)
  Initial State: %v`, tranTab.RowCount, tranTab.RowCount*tranTab.ColCount, tranTab.InitialState)
		config.logger.Log("  Accepting States:")
		for state, symbol := range tranTab.AcceptingStates {
			config.logger.Log("    %v: %v", state, symbol)
		}
	}

	var err error
	switch config.compLv {
	case 2:
		tranTab, err = compressTransitionTableLv2(tranTab)
		if err != nil {
			return nil, err
		}
	case 1:
		tranTab, err = compressTransitionTableLv1(tranTab)
		if err != nil {
			return nil, err
		}
	}

	return &spec.CompiledLexModeSpec{
		Kinds: kinds,
		Push:  push,
		Pop:   pop,
		DFA:   tranTab,
	}, nil
}

const (
	CompressionLevelMin = 0
	CompressionLevelMax = 2
)

func compressTransitionTableLv2(tranTab *spec.TransitionTable) (*spec.TransitionTable, error) {
	ueTab := compressor.NewUniqueEntriesTable()
	{
		orig, err := compressor.NewOriginalTable(tranTab.UncompressedTransition, tranTab.ColCount)
		if err != nil {
			return nil, err
		}
		err = ueTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	rdTab := compressor.NewRowDisplacementTable(0)
	{
		orig, err := compressor.NewOriginalTable(ueTab.UniqueEntries, ueTab.OriginalColCount)
		if err != nil {
			return nil, err
		}
		err = rdTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	tranTab.Transition = &spec.UniqueEntriesTable{
		UniqueEntries: &spec.RowDisplacementTable{
			OriginalRowCount: rdTab.OriginalRowCount,
			OriginalColCount: rdTab.OriginalColCount,
			EmptyValue:       rdTab.EmptyValue,
			Entries:          rdTab.Entries,
			Bounds:           rdTab.Bounds,
			RowDisplacement:  rdTab.RowDisplacement,
		},
		RowNums:          ueTab.RowNums,
		OriginalRowCount: ueTab.OriginalRowCount,
		OriginalColCount: ueTab.OriginalColCount,
	}
	tranTab.UncompressedTransition = nil

	return tranTab, nil
}

func compressTransitionTableLv1(tranTab *spec.TransitionTable) (*spec.TransitionTable, error) {
	ueTab := compressor.NewUniqueEntriesTable()
	{
		orig, err := compressor.NewOriginalTable(tranTab.UncompressedTransition, tranTab.ColCount)
		if err != nil {
			return nil, err
		}
		err = ueTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	tranTab.Transition = &spec.UniqueEntriesTable{
		UncompressedUniqueEntries: ueTab.UniqueEntries,
		RowNums:                   ueTab.RowNums,
		OriginalRowCount:          ueTab.OriginalRowCount,
		OriginalColCount:          ueTab.OriginalColCount,
	}
	tranTab.UncompressedTransition = nil

	return tranTab, nil
}
