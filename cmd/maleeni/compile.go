package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/nihei9/maleeni/compiler"
	"github.com/nihei9/maleeni/spec"
	"github.com/spf13/cobra"
)

var compileFlags = struct {
	debug   *bool
	lexSpec *string
	compLv  *int
	output  *string
}{}

func init() {
	cmd := &cobra.Command{
		Use:     "compile",
		Short:   "Compile a lexical specification into a DFA",
		Long:    `compile takes a lexical specification and generates a DFA accepting the tokens described in the specification.`,
		Example: `  cat lexspec.json | maleeni compile > clexspec.json`,
		RunE:    runCompile,
	}
	compileFlags.debug = cmd.Flags().BoolP("debug", "d", false, "enable logging")
	compileFlags.lexSpec = cmd.Flags().StringP("lex-spec", "l", "", "lexical specification file path (default stdin)")
	compileFlags.compLv = cmd.Flags().Int("compression-level", compiler.CompressionLevelMax, "compression level")
	compileFlags.output = cmd.Flags().StringP("output", "o", "", "output file path (default stdout)")
	rootCmd.AddCommand(cmd)
}

func runCompile(cmd *cobra.Command, args []string) (retErr error) {
	lspec, err := readLexSpec(*compileFlags.lexSpec)
	if err != nil {
		return fmt.Errorf("Cannot read a lexical specification: %w", err)
	}

	opts := []compiler.CompilerOption{
		compiler.CompressionLevel(*compileFlags.compLv),
	}
	if *compileFlags.debug {
		fileName := "maleeni-compile.log"
		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Cannot open the log file %s: %w", fileName, err)
		}
		defer f.Close()
		fmt.Fprintf(f, `maleeni compile starts.
Date time: %v
---
`, time.Now().Format(time.RFC3339))
		defer func() {
			fmt.Fprintf(f, "---\n")
			if retErr != nil {
				fmt.Fprintf(f, "maleeni compile failed: %v\n", retErr)
			} else {
				fmt.Fprintf(f, "maleeni compile succeeded.\n")
			}
		}()

		opts = append(opts, compiler.EnableLogging(f))
	}

	clspec, err := compiler.Compile(lspec, opts...)
	if err != nil {
		return err
	}
	err = writeCompiledLexSpec(clspec, *compileFlags.output)
	if err != nil {
		return fmt.Errorf("Cannot write a compiled lexical specification: %w", err)
	}

	return nil
}

func readLexSpec(path string) (*spec.LexSpec, error) {
	r := os.Stdin
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Cannot open the lexical specification file %s: %w", path, err)
		}
		defer f.Close()
		r = f
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	lspec := &spec.LexSpec{}
	err = json.Unmarshal(data, lspec)
	if err != nil {
		return nil, err
	}
	return lspec, nil
}

func writeCompiledLexSpec(clspec *spec.CompiledLexSpec, path string) error {
	out, err := json.Marshal(clspec)
	if err != nil {
		return err
	}
	w := os.Stdout
	if path != "" {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Cannot open the output file %s: %w", path, err)
		}
		defer f.Close()
		w = f
	}
	fmt.Fprintf(w, "%v\n", string(out))
	return nil
}
