// Package main provides the entry point for the Chameleon CLI tool
package main

import (
	"fmt"
	"os"

	"github.com/zyanho/chameleon/cmd/chameleon/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
