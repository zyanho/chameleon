package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zyanho/chameleon/pkg/plugin"
)

// TestPlugin implements the plugin interface
type TestPlugin struct {
	version string
	data    map[string]interface{}
}

var _ plugin.Bureau = (*TestPlugin)(nil)

func (p *TestPlugin) Name() string {
	return "version-test-plugin"
}

func (p *TestPlugin) Version() string {
	return p.version
}

func (p *TestPlugin) Init(args ...interface{}) error {
	p.data = make(map[string]interface{})
	fmt.Printf("Plugin v%s initialized with args: %v\n", p.version, args)
	return nil
}

func (p *TestPlugin) Free() error {
	fmt.Printf("Plugin v%s freed\n", p.version)
	return nil
}

// LongRunning simulates a long-running operation
func (p *TestPlugin) LongRunning(ctx context.Context, duration int) (string, error) {
	fmt.Printf("v%s: Starting long running operation for %d seconds\n", p.version, duration)

	select {
	case <-time.After(time.Duration(duration) * time.Second):
		return fmt.Sprintf("v%s: Completed long operation", p.version), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// GetVersion returns the current version
func (p *TestPlugin) GetVersion(ctx context.Context) string {
	return fmt.Sprintf("Running version: %s", p.version)
}

// Export exposes the plugin instance
var Export plugin.Bureau = &TestPlugin{version: "2.0.0"}
