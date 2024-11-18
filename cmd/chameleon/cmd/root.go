// Package cmd implements command-line interface for the Chameleon tool
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "chameleon",
	Short: "Chameleon plugin builder",
	Long: `Chameleon is a tool for building Go plugins that can be loaded
dynamically at runtime with independent dependency management.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
