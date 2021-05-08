package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/nihei9/maleeni/driver"
	"github.com/nihei9/maleeni/spec"
	"github.com/spf13/cobra"
)

var lexFlags = struct {
	debug        *bool
	source       *string
	output       *string
	breakOnError *bool
}{}

func init() {
	cmd := &cobra.Command{
		Use:   "lex clexspec",
		Short: "Tokenize a text stream",
		Long: `lex takes a text stream and tokenizes it according to a compiled lexical specification.
As use ` + "`maleeni compile`" + `, you can generate the specification.`,
		Example: `  cat src | maleeni lex clexspec.json`,
		Args:    cobra.ExactArgs(1),
		RunE:    runLex,
	}
	lexFlags.debug = cmd.Flags().BoolP("debug", "d", false, "enable logging")
	lexFlags.source = cmd.Flags().StringP("source", "s", "", "source file path (default: stdin)")
	lexFlags.output = cmd.Flags().StringP("output", "o", "", "output file path (default: stdout)")
	lexFlags.breakOnError = cmd.Flags().BoolP("break-on-error", "b", false, "break lexical analysis with exit status 1 immediately when an error token appears.")
	rootCmd.AddCommand(cmd)
}

func runLex(cmd *cobra.Command, args []string) (retErr error) {
	clspec, err := readCompiledLexSpec(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled lexical specification: %w", err)
	}

	var opts []driver.LexerOption
	if *lexFlags.debug {
		fileName := "maleeni-lex.log"
		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Cannot open the log file %s: %w", fileName, err)
		}
		defer f.Close()
		fmt.Fprintf(f, `maleeni lex starts.
Date time: %v
---
`, time.Now().Format(time.RFC3339))
		defer func() {
			fmt.Fprintf(f, "---\n")
			if retErr != nil {
				fmt.Fprintf(f, "maleeni lex failed: %v\n", retErr)
			} else {
				fmt.Fprintf(f, "maleeni lex succeeded.\n")
			}
		}()

		opts = append(opts, driver.EnableLogging(f))
	}

	var lex *driver.Lexer
	{
		src := os.Stdin
		if *lexFlags.source != "" {
			f, err := os.Open(*lexFlags.source)
			if err != nil {
				return fmt.Errorf("Cannot open the source file %s: %w", *lexFlags.source, err)
			}
			defer f.Close()
			src = f
		}
		lex, err = driver.NewLexer(clspec, src, opts...)
		if err != nil {
			return err
		}
	}
	w := os.Stdout
	if *lexFlags.output != "" {
		f, err := os.OpenFile(*lexFlags.output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Cannot open the output file %s: %w", *lexFlags.output, err)
		}
		defer f.Close()
		w = f
	}
	for {
		tok, err := lex.Next()
		if err != nil {
			return err
		}
		data, err := json.Marshal(tok)
		if err != nil {
			return fmt.Errorf("failed to marshal a token; token: %v, error: %v\n", tok, err)
		}
		if tok.Invalid && *lexFlags.breakOnError {
			return fmt.Errorf("detected an error token: %v", string(data))
		}
		fmt.Fprintf(w, "%v\n", string(data))
		if tok.EOF {
			break
		}
	}

	return nil
}

func readCompiledLexSpec(path string) (*spec.CompiledLexSpec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	clspec := &spec.CompiledLexSpec{}
	err = json.Unmarshal(data, clspec)
	if err != nil {
		return nil, err
	}
	return clspec, nil
}
