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

	modeEntries, modeNames, modeName2ID, fragmetns := groupEntriesByLexMode(lexspec.Entries)

	modeSpecs := []*spec.CompiledLexModeSpec{
		nil,
	}
	for i, es := range modeEntries[1:] {
		modeName := modeNames[i+1]
		config.logger.Log("Compile %v mode:", modeName)
		modeSpec, err := compile(es, modeName2ID, fragmetns, config)
		if err != nil {
			return nil, fmt.Errorf("failed to compile in %v mode: %w", modeName, err)
		}
		modeSpecs = append(modeSpecs, modeSpec)
	}

	var kindNames []spec.LexKindName
	var name2ID map[spec.LexKindName]spec.LexKindID
	{
		name2ID = map[spec.LexKindName]spec.LexKindID{}
		id := spec.LexKindIDMin
		for _, modeSpec := range modeSpecs[1:] {
			for _, name := range modeSpec.KindNames[1:] {
				if _, ok := name2ID[name]; ok {
					continue
				}
				name2ID[name] = id
				id++
			}
		}

		kindNames = make([]spec.LexKindName, len(name2ID)+1)
		for name, id := range name2ID {
			kindNames[id] = name
		}
	}

	var kindIDs [][]spec.LexKindID
	{
		kindIDs = make([][]spec.LexKindID, len(modeSpecs))
		for i, modeSpec := range modeSpecs[1:] {
			ids := make([]spec.LexKindID, len(modeSpec.KindNames))
			for modeID, name := range modeSpec.KindNames {
				if modeID == 0 {
					continue
				}
				ids[modeID] = name2ID[name]
			}
			kindIDs[i+1] = ids
		}
	}

	return &spec.CompiledLexSpec{
		Name:             lexspec.Name,
		InitialModeID:    spec.LexModeIDDefault,
		ModeNames:        modeNames,
		KindNames:        kindNames,
		KindIDs:          kindIDs,
		CompressionLevel: config.compLv,
		Specs:            modeSpecs,
	}, nil
}

func groupEntriesByLexMode(entries []*spec.LexEntry) ([][]*spec.LexEntry, []spec.LexModeName, map[spec.LexModeName]spec.LexModeID, map[string]*spec.LexEntry) {
	modeNames := []spec.LexModeName{
		spec.LexModeNameNil,
		spec.LexModeNameDefault,
	}
	modeName2ID := map[spec.LexModeName]spec.LexModeID{
		spec.LexModeNameNil:     spec.LexModeIDNil,
		spec.LexModeNameDefault: spec.LexModeIDDefault,
	}
	lastModeID := spec.LexModeIDDefault
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
		for _, modeName := range ms {
			modeID, ok := modeName2ID[modeName]
			if !ok {
				modeID = lastModeID + 1
				lastModeID = modeID
				modeName2ID[modeName] = modeID
				modeNames = append(modeNames, modeName)
				modeEntries = append(modeEntries, []*spec.LexEntry{})
			}
			modeEntries[modeID] = append(modeEntries[modeID], e)
		}
	}
	return modeEntries, modeNames, modeName2ID, fragments
}

func compile(entries []*spec.LexEntry, modeName2ID map[spec.LexModeName]spec.LexModeID, fragments map[string]*spec.LexEntry, config *compilerConfig) (*spec.CompiledLexModeSpec, error) {
	var kindNames []spec.LexKindName
	var patterns map[spec.LexModeKindID][]byte
	{
		kindNames = append(kindNames, spec.LexKindNameNil)
		patterns = map[spec.LexModeKindID][]byte{}
		for i, e := range entries {
			kindNames = append(kindNames, e.Kind)
			patterns[spec.LexModeKindID(i+1)] = []byte(e.Pattern)
		}

		config.logger.Log("Patterns:")
		for i, p := range patterns {
			config.logger.Log("  #%v %v", i, string(p))
		}
	}

	push := []spec.LexModeID{
		spec.LexModeIDNil,
	}
	pop := []int{
		0,
	}
	for _, e := range entries {
		pushV := spec.LexModeIDNil
		if e.Push != "" {
			pushV = modeName2ID[e.Push]
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
  Initial State ID: %v`, tranTab.RowCount, tranTab.RowCount*tranTab.ColCount, tranTab.InitialStateID)
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
		KindNames: kindNames,
		Push:      push,
		Pop:       pop,
		DFA:       tranTab,
	}, nil
}

const (
	CompressionLevelMin = 0
	CompressionLevelMax = 2
)

func compressTransitionTableLv2(tranTab *spec.TransitionTable) (*spec.TransitionTable, error) {
	ueTab := compressor.NewUniqueEntriesTable()
	{
		orig, err := compressor.NewOriginalTable(convertStateIDSliceToIntSlice(tranTab.UncompressedTransition), tranTab.ColCount)
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
			EmptyValue:       spec.StateIDNil,
			Entries:          convertIntSliceToStateIDSlice(rdTab.Entries),
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
		orig, err := compressor.NewOriginalTable(convertStateIDSliceToIntSlice(tranTab.UncompressedTransition), tranTab.ColCount)
		if err != nil {
			return nil, err
		}
		err = ueTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	tranTab.Transition = &spec.UniqueEntriesTable{
		UncompressedUniqueEntries: convertIntSliceToStateIDSlice(ueTab.UniqueEntries),
		RowNums:                   ueTab.RowNums,
		OriginalRowCount:          ueTab.OriginalRowCount,
		OriginalColCount:          ueTab.OriginalColCount,
	}
	tranTab.UncompressedTransition = nil

	return tranTab, nil
}

func convertStateIDSliceToIntSlice(s []spec.StateID) []int {
	is := make([]int, len(s))
	for i, v := range s {
		is[i] = v.Int()
	}
	return is
}

func convertIntSliceToStateIDSlice(s []int) []spec.StateID {
	ss := make([]spec.StateID, len(s))
	for i, v := range s {
		ss[i] = spec.StateID(v)
	}
	return ss
}
