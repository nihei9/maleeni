package main

import (
	"os"

	"github.com/nihei9/maleeni/cli/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
