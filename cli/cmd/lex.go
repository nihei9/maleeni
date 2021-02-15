package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nihei9/maleeni/driver"
	"github.com/nihei9/maleeni/spec"
	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(cmd)
}

func runLex(cmd *cobra.Command, args []string) error {
	var clspec *spec.CompiledLexSpec
	{
		clspecPath := args[0]
		f, err := os.Open(clspecPath)
		if err != nil {
			return err
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		clspec = &spec.CompiledLexSpec{}
		err = json.Unmarshal(data, clspec)
		if err != nil {
			return err
		}
	}
	lex, err := driver.NewLexer(clspec, os.Stdin)
	if err != nil {
		return err
	}
	for {
		tok, err := lex.Next()
		if err != nil {
			return err
		}
		if tok.EOF {
			break
		}
		if tok.Invalid {
			fmt.Fprintf(os.Stdout, "-: -: ")
		} else {
			fmt.Fprintf(os.Stdout, "%v: %v: ", tok.ID, clspec.Kinds[tok.ID])
		}
		fmt.Fprintf(os.Stdout, "\"%v\"\n", string(tok.Match))
	}
	return nil
}
