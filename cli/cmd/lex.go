package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

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

func runLex(cmd *cobra.Command, args []string) (retErr error) {
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
	var w io.Writer
	{
		f, err := os.OpenFile("maleeni-lex.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	fmt.Fprintf(w, `maleeni lex starts.
Date time: %v
---
`, time.Now().Format(time.RFC3339))
	defer func() {
		fmt.Fprintf(w, "---\n")
		if retErr != nil {
			fmt.Fprintf(w, "maleeni lex failed: %v\n", retErr)
		} else {
			fmt.Fprintf(w, "maleeni lex succeeded.\n")
		}
	}()
	lex, err := driver.NewLexer(clspec, os.Stdin, driver.EnableLogging(w))
	if err != nil {
		return err
	}
	for {
		tok, err := lex.Next()
		if err != nil {
			return err
		}
		data, err := json.Marshal(tok)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal a token; token: %v, error: %v\n", tok, err)
		}
		fmt.Fprintf(os.Stdout, "%v\n", string(data))
		if tok.EOF {
			break
		}
	}

	return nil
}
