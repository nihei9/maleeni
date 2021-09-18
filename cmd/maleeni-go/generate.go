package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nihei9/maleeni/driver"
	"github.com/nihei9/maleeni/spec"
	"github.com/spf13/cobra"
)

func Execute() error {
	err := generateCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	return nil
}

var generateFlags = struct {
	pkgName *string
	output  *string
}{}

var generateCmd = &cobra.Command{
	Use:           "maleeni-go",
	Short:         "Generate a lexer for Go",
	Long:          `maleeni-go generates a lexer for Go. The lexer recognizes the lexical specification specified as the argument.`,
	Example:       `  maleeni-go clexspec.json`,
	Args:          cobra.ExactArgs(1),
	RunE:          runGenerate,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	generateFlags.pkgName = generateCmd.Flags().StringP("package", "p", "main", "package name")
	generateFlags.output = generateCmd.Flags().StringP("output", "o", "", "output file path")
}

func runGenerate(cmd *cobra.Command, args []string) (retErr error) {
	clspec, err := readCompiledLexSpec(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled lexical specification: %w", err)
	}

	b, err := driver.GenLexer(clspec, *generateFlags.pkgName)
	if err != nil {
		return fmt.Errorf("Failed to generate a lexer: %v", err)
	}

	var filePath string
	if *generateFlags.output != "" {
		filePath = *generateFlags.output
	} else {
		filePath = fmt.Sprintf("%v_lexer.go", clspec.Name)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to create an output file: %v", err)
	}
	defer f.Close()

	_, err = f.Write(b)
	if err != nil {
		return fmt.Errorf("Failed to write lexer source code: %v", err)
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
