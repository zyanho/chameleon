package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zyanho/chameleon/cmd/chameleon/generator"
)

var generateCmd = &cobra.Command{
	Use:   "generate [plugin_dir]",
	Short: "Generate plugin wrapper code",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generator.Generate(args[0])
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
