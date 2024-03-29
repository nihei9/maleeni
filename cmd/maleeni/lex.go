package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nihei9/maleeni/driver"
	"github.com/nihei9/maleeni/spec"
	"github.com/spf13/cobra"
)

var lexFlags = struct {
	source       *string
	output       *string
	breakOnError *bool
}{}

func init() {
	cmd := &cobra.Command{
		Use:   "lex clexspec",
		Short: "Tokenize a text stream",
		Long: `lex takes a text stream and tokenizes it according to a compiled lexical specification.
As use ` + "`maleeni compile`" + `, you can generate the specification.

Note that passive mode transitions are not performed. Thus, if there is a mode in
your lexical specification that is set passively, lexemes in that mode will not be recognized.`,
		Example: `  cat src | maleeni lex clexspec.json`,
		Args:    cobra.ExactArgs(1),
		RunE:    runLex,
	}
	lexFlags.source = cmd.Flags().StringP("source", "s", "", "source file path (default stdin)")
	lexFlags.output = cmd.Flags().StringP("output", "o", "", "output file path (default stdout)")
	lexFlags.breakOnError = cmd.Flags().BoolP("break-on-error", "b", false, "break lexical analysis with exit status 1 immediately when an error token appears.")
	rootCmd.AddCommand(cmd)
}

func runLex(cmd *cobra.Command, args []string) (retErr error) {
	clspec, err := readCompiledLexSpec(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled lexical specification: %w", err)
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
		lex, err = driver.NewLexer(driver.NewLexSpec(clspec), src)
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

	tok2JSON := genTokenJSONMarshaler(clspec)
	for {
		tok, err := lex.Next()
		if err != nil {
			return err
		}
		data, err := tok2JSON(tok)
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

func genTokenJSONMarshaler(clspec *spec.CompiledLexSpec) func(tok *driver.Token) ([]byte, error) {
	return func(tok *driver.Token) ([]byte, error) {
		return json.Marshal(struct {
			ModeID     int    `json:"mode_id"`
			ModeName   string `json:"mode_name"`
			KindID     int    `json:"kind_id"`
			ModeKindID int    `json:"mode_kind_id"`
			KindName   string `json:"kind_name"`
			Row        int    `json:"row"`
			Col        int    `json:"col"`
			Lexeme     string `json:"lexeme"`
			EOF        bool   `json:"eof"`
			Invalid    bool   `json:"invalid"`
		}{
			ModeID:     tok.ModeID.Int(),
			ModeName:   clspec.ModeNames[tok.ModeID].String(),
			KindID:     tok.KindID.Int(),
			ModeKindID: tok.ModeKindID.Int(),
			KindName:   clspec.KindNames[tok.KindID].String(),
			Row:        tok.Row,
			Col:        tok.Col,
			Lexeme:     string(tok.Lexeme),
			EOF:        tok.EOF,
			Invalid:    tok.Invalid,
		})
	}
}
