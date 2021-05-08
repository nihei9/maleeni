package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "maleeni",
	Short: "Generate a portable DFA from a lexical specification",
	Long: `maleeni provides two features:
* Generates a portable DFA from a lexical specification.
* Tokenizes a text stream according to the lexical specification.
  This feature is primarily aimed at debugging the lexical specification.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	return nil
}
