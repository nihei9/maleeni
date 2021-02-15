package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

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

func runCompile(cmd *cobra.Command, args []string) error {
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
	clspec, err := compiler.Compile(lspec)
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
