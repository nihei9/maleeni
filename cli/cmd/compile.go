package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/nihei9/maleeni/compiler"
	"github.com/nihei9/maleeni/spec"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:     "compile",
		Short:   "Compile a lexical specification into a DFA",
		Long:    `compile takes a lexical specification and generates a DFA accepting the tokens described in the specification.`,
		Example: `  cat lexspec.json | maleeni compile > clexspec.json`,
		RunE:    runCompile,
	}
	rootCmd.AddCommand(cmd)
}

func runCompile(cmd *cobra.Command, args []string) (retErr error) {
	var lspec *spec.LexSpec
	{
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		lspec = &spec.LexSpec{}
		err = json.Unmarshal(data, lspec)
		if err != nil {
			return err
		}
	}
	var w io.Writer
	{
		f, err := os.OpenFile("maleeni-compile.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	fmt.Fprintf(w, `maleeni compile starts.
Date time: %v
---
`, time.Now().Format(time.RFC3339))
	defer func() {
		fmt.Fprintf(w, "---\n")
		if retErr != nil {
			fmt.Fprintf(w, "maleeni compile failed: %v\n", retErr)
		} else {
			fmt.Fprintf(w, "maleeni compile succeeded.\n")
		}
	}()
	clspec, err := compiler.Compile(lspec, compiler.EnableLogging(w))
	if err != nil {
		return err
	}
	out, err := json.Marshal(clspec)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%v\n", string(out))

	return nil
}
