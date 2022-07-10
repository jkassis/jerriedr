package main

import (
	"fmt"
	"os"

	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed USAGE.txt
var usage string

// MAIN represents the base command when called without any subcommands
var MAIN = &cobra.Command{
	Use:   "jerriedr",
	Short: "A CLI for operations on jerrie services.",
	Long:  usage,
}

func main() {
	err := MAIN.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
