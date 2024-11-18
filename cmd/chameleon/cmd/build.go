package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zyanho/chameleon/cmd/chameleon/generator"
)

var buildCmd = &cobra.Command{
	Use:   "build [plugin directory]",
	Short: "Build a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuild,
}

func init() {
	buildCmd.Flags().StringP("output", "o", "", "output file path")
}

// runBuild handles the plugin build process
func runBuild(cmd *cobra.Command, args []string) error {
	pluginDir := args[0]
	outputPath, _ := cmd.Flags().GetString("output")

	if err := validatePluginDir(pluginDir); err != nil {
		return err
	}

	if err := generator.Generate(pluginDir); err != nil {
		return fmt.Errorf("failed to generate wrapper: %w", err)
	}

	if err := buildPlugin(pluginDir, outputPath); err != nil {
		return fmt.Errorf("failed to build plugin: %w", err)
	}

	return nil
}

// validatePluginDir checks if the plugin directory is valid
func validatePluginDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("plugin directory not found: %w", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return fmt.Errorf("go.mod not found in plugin directory")
	}

	return nil
}

// buildPlugin compiles the plugin into a shared object file
func buildPlugin(dir, output string) error {
	if output == "" {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, "plugin.so")), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		output = filepath.Join(dir, "plugin.so")
	}

	cmd := exec.Command("go", "build",
		"-buildmode=plugin",
		"-o", output,
		".",
	)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
