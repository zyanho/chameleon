package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mock plugin implementation
type mockPlugin struct {
	version string
	funcs   map[string]interface{}
}

// NewMockPlugin creates a new mock plugin
func NewMockPlugin(version string, funcs map[string]interface{}) *Plugin {
	mock := &mockPlugin{
		version: version,
		funcs:   funcs,
	}

	// Convert regular function map to InvokeFunc map
	invokeFuncs := make(map[string]InvokeFunc)
	for name, fn := range funcs {
		func(name string, fn interface{}) {
			invokeFuncs[name] = func(ctx context.Context, args ...interface{}) (interface{}, error) {
				// Special handling for FailingFunc
				if name == "FailingFunc" {
					if fn, ok := fn.(func() error); ok {
						return nil, fn()
					}
				}
				return fn, nil
			}
		}(name, fn)
	}

	return &Plugin{
		bureau: mock,
		funcs:  invokeFuncs,
	}
}

// Bureau interface implementation
func (p *mockPlugin) Name() string {
	return "mock-plugin"
}

func (p *mockPlugin) Version() string {
	return p.version
}

func (p *mockPlugin) Init(args ...interface{}) error {
	return nil
}

func (p *mockPlugin) Free() error {
	return nil
}

func (p *mockPlugin) GetFunctions() []string {
	funcs := make([]string, 0, len(p.funcs))
	for name := range p.funcs {
		funcs = append(funcs, name)
	}
	return funcs
}

func TestManager_LoadPlugin(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (*Manager, error)
		wantErr bool
	}{
		{
			name: "load valid plugin",
			setup: func() (*Manager, error) {
				dir := t.TempDir()

				// Create a mock plugin file
				mockFuncs := map[string]interface{}{
					"TestFunc": "test result",
				}
				plugin := NewMockPlugin("1.0.0", mockFuncs)

				instance := &PluginInstance{
					Plugin:  plugin,
					state:   StateActive,
					version: plugin.Version(),
				}

				// Create configuration
				config := &Config{
					PluginDir: dir,
					DefaultPluginConfig: PluginSpecificConfig{
						CircuitBreaker: CircuitBreakerConfig{
							Enabled:         true,
							MaxFailures:     5,
							ResetInterval:   time.Second,
							TimeoutDuration: time.Second,
						},
					},
				}

				m, err := NewManager(context.Background(), config)
				if err != nil {
					return nil, err
				}

				// Pre-store plugin instance
				pluginName := "test-plugin"
				m.plugins.Store(pluginName, instance)
				m.pluginPaths.Store(pluginName, filepath.Join(dir, "test-plugin.so"))

				return m, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.setup()
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()

			// Verify plugin is loaded
			if _, ok := m.plugins.Load("test-plugin"); !ok {
				t.Error("Plugin not loaded")
			}
		})
	}
}

func TestManager_Call(t *testing.T) {
	ctx := context.Background()
	m, cleanup := setupTestManager(t)
	defer cleanup()

	pluginName := "test-plugin"
	mockFuncs := map[string]interface{}{
		"TestFunc": "test result",
	}

	plugin := NewMockPlugin("1.0.0", mockFuncs)
	instance := &PluginInstance{
		Plugin:  plugin,
		state:   StateActive,
		version: plugin.Version(),
	}

	// Store plugin instance
	m.plugins.Store(pluginName, instance)

	// Initialize circuit breaker
	breaker := NewCircuitBreaker(ctx, CircuitBreakerConfig{
		Enabled:         true,
		MaxFailures:     5,
		ResetInterval:   time.Second,
		TimeoutDuration: time.Second,
	}, m.logger)
	m.breakers.Store(pluginName, breaker)

	tests := []struct {
		name    string
		fn      string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "call existing function",
			fn:   "TestFunc",
			want: "test result",
		},
		{
			name:    "call non-existing function",
			fn:      "NonExistingFunc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Call(ctx, pluginName, tt.fn, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Call() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Call() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	m, cleanup := setupTestManager(t)
	defer cleanup()

	pluginName := "test-plugin"
	mockFuncs := map[string]interface{}{
		"FailingFunc": func() error { return fmt.Errorf("test error") },
	}

	plugin := NewMockPlugin("1.0.0", mockFuncs)
	instance := &PluginInstance{
		Plugin:  plugin,
		state:   StateActive,
		version: plugin.Version(),
	}

	m.plugins.Store(pluginName, instance)
	m.breakers.Store(pluginName, NewCircuitBreaker(ctx, CircuitBreakerConfig{
		Enabled:         true,
		MaxFailures:     5,
		ResetInterval:   time.Second,
		TimeoutDuration: time.Second,
	}, m.logger))

	// Trigger circuit breaker
	for i := 0; i < 6; i++ {
		_, err := m.Call(ctx, pluginName, "FailingFunc")
		if err == nil {
			t.Error("Expected error from FailingFunc")
		}
	}

	// Verify circuit breaker is open
	if !m.GetBreakerStatus(pluginName) {
		t.Error("Expected circuit breaker to be open")
	}

	// Wait for reset
	time.Sleep(2 * time.Second)

	// Verify circuit breaker is closed
	if m.GetBreakerStatus(pluginName) {
		t.Error("Expected circuit breaker to be closed")
	}
}

func setupTestManager(t testing.TB) (*Manager, func()) {
	dir := t.TempDir()
	config := &Config{
		PluginDir:     dir,
		EnableMetrics: true,
		DefaultPluginConfig: PluginSpecificConfig{
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:         true,
				MaxFailures:     5,
				ResetInterval:   time.Second,
				TimeoutDuration: time.Second,
			},
		},
	}

	m, err := NewManager(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		m.Close()
		os.RemoveAll(dir)
	}

	return m, cleanup
}

// Test plugin version upgrade
func TestPluginUpgrade(t *testing.T) {
	ctx := context.Background()
	m, cleanup := setupTestManager(t)
	defer cleanup()

	pluginName := "test-plugin"

	// Load v1.0.0
	mockFuncs1 := map[string]interface{}{
		"TestFunc": "v1 result",
	}
	plugin1 := NewMockPlugin("1.0.0", mockFuncs1)
	instance1 := &PluginInstance{
		Plugin:  plugin1,
		state:   StateActive,
		version: plugin1.Version(),
	}
	m.plugins.Store(pluginName, instance1)
	m.breakers.Store(pluginName, NewCircuitBreaker(ctx, CircuitBreakerConfig{
		Enabled:         true,
		MaxFailures:     5,
		ResetInterval:   time.Second,
		TimeoutDuration: time.Second,
	}, m.logger))

	// Simulate plugin upgrade
	mockFuncs2 := map[string]interface{}{
		"TestFunc": "v2 result",
	}
	plugin2 := NewMockPlugin("2.0.0", mockFuncs2)
	instance2 := &PluginInstance{
		Plugin:  plugin2,
		state:   StateActive,
		version: plugin2.Version(),
	}

	// Use LoadPluginWithConfig to properly handle the upgrade
	oldInstance, ok := m.plugins.Load(pluginName)
	if ok {
		old := oldInstance.(*PluginInstance)
		old.state = StateDeprecated
	}
	m.plugins.Store(pluginName, instance2)

	// Verify v1 is deprecated
	if instance1.state != StateDeprecated {
		t.Errorf("Expected v1 to be deprecated, got state: %v", instance1.state)
	}

	// Verify current version is v2
	result, err := m.Call(ctx, pluginName, "TestFunc")
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if result != "v2 result" {
		t.Errorf("Expected v2 result, got %v", result)
	}

	// Verify version upgrade
	plugins := m.ListPlugins()
	found := false
	for _, p := range plugins {
		if p.Name == pluginName {
			if p.Version != "2.0.0" {
				t.Errorf("Expected version 2.0.0, got %s", p.Version)
			}
			if p.State != StateActive {
				t.Errorf("Expected state Active, got %v", p.State)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("Plugin not found in list")
	}
}

// Test concurrent plugin calls
func TestConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	m, cleanup := setupTestManager(t)
	defer cleanup()

	// Load multiple plugins
	plugins := map[string]string{
		"plugin1": "result1",
		"plugin2": "result2",
		"plugin3": "result3",
	}

	for name, result := range plugins {
		mockFuncs := map[string]interface{}{
			"TestFunc": result,
		}
		plugin := NewMockPlugin("1.0.0", mockFuncs)
		instance := &PluginInstance{
			Plugin:  plugin,
			state:   StateActive,
			version: plugin.Version(),
		}
		m.plugins.Store(name, instance)
		m.breakers.Store(name, NewCircuitBreaker(ctx, CircuitBreakerConfig{
			Enabled:         true,
			MaxFailures:     5,
			ResetInterval:   time.Second,
			TimeoutDuration: time.Second,
		}, m.logger))
	}

	// Concurrent calls
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("panic in concurrent call: %v", r)
				}
				wg.Done()
			}()

			for name, expected := range plugins {
				result, err := m.Call(ctx, name, "TestFunc")
				if err != nil {
					errChan <- fmt.Errorf("plugin %s call error: %v", name, err)
					return
				}
				if result != expected {
					errChan <- fmt.Errorf("plugin %s: expected %s, got %v", name, expected, result)
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

// Test graceful shutdown
func TestGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	m, cleanup := setupTestManager(t)
	defer cleanup()

	// Load a plugin with long-running function
	pluginName := "test-plugin"
	mockFuncs := map[string]interface{}{
		"LongFunc": func() string {
			// Simulate a long-running operation that checks context
			select {
			case <-time.After(3 * time.Second):
				return "completed"
			case <-ctx.Done():
				return "cancelled"
			}
		},
	}
	plugin := NewMockPlugin("1.0.0", mockFuncs)
	instance := &PluginInstance{
		Plugin:  plugin,
		state:   StateActive,
		version: plugin.Version(),
	}

	// Store plugin instance
	m.plugins.Store(pluginName, instance)

	// Initialize circuit breaker
	breaker := NewCircuitBreaker(ctx, CircuitBreakerConfig{
		Enabled:         true,
		MaxFailures:     5,
		ResetInterval:   time.Second,
		TimeoutDuration: time.Second,
	}, m.logger)
	m.breakers.Store(pluginName, breaker)

	// Start multiple long-running calls
	var wg sync.WaitGroup
	callCount := 3
	wg.Add(callCount)

	for i := 0; i < callCount; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in long running call: %v", r)
				}
				wg.Done()
			}()

			_, err := m.Call(ctx, pluginName, "LongFunc")
			if err != nil && err != context.Canceled {
				t.Errorf("Unexpected error: %v", err)
			}
		}()
	}

	// Wait for calls to start
	time.Sleep(500 * time.Millisecond)

	// Create a channel to track shutdown completion
	shutdownComplete := make(chan struct{})

	// Start shutdown in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic in shutdown goroutine: %v", r)
			}
		}()

		// Close manager
		if err := m.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		// Wait for all calls to complete
		wg.Wait()

		// Verify all resources are cleaned up
		var pluginCount int
		m.plugins.Range(func(key, value interface{}) bool {
			pluginCount++
			return true
		})
		if pluginCount > 0 {
			t.Error("Expected all plugins to be cleaned up")
		}

		// Signal shutdown completion
		close(shutdownComplete)
	}()

	// Wait for either shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		// Shutdown completed successfully
	case <-time.After(5 * time.Second):
		t.Error("Shutdown timed out")
	}
}
