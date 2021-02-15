package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "maleeni",
	Short: "Generate a portable DFA from a lexical specification",
	Long: `maleeni provides two features:
* Generates a portable DFA from a lexical specification.
* Tokenizes a text stream according to the lexical specification.
  This feature is primarily aimed at debugging the lexical specification.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}
