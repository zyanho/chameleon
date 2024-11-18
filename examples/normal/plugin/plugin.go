package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/zyanho/chameleon/pkg/plugin"
)

// ExamplePlugin implements the plugin interface
type ExamplePlugin struct {
	data map[string]interface{}
}

// Ensure interface implementation
var _ plugin.Bureau = (*ExamplePlugin)(nil)

func (p *ExamplePlugin) Name() string {
	return "example-plugin"
}

func (p *ExamplePlugin) Version() string {
	return "1.0.1"
}

func (p *ExamplePlugin) Init(args ...interface{}) error {
	p.data = make(map[string]interface{})
	// Process initialization parameters
	for i, arg := range args {
		p.data[fmt.Sprintf("init-%d", i)] = arg
	}
	return nil
}

func (p *ExamplePlugin) Free() error {
	p.data = nil
	return nil
}

// Plugin custom method
func (p *ExamplePlugin) SetValue(ctx context.Context, key string, value interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if p.data == nil {
			p.data = make(map[string]interface{})
		}
		p.data[key] = value
		return nil
	}
}

// Plugin custom method
func (p *ExamplePlugin) Add(ctx context.Context, a int, b int) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		return a + b, nil
	}
}

func (p *ExamplePlugin) Some1111(ctx context.Context, a int, b int, c string) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		if strings.HasPrefix(c, "1111") {
			return a + b + 1111, nil
		}
		return a + b, nil
	}
}

// Export exposes the plugin instance
var Export plugin.Bureau = &ExamplePlugin{}
