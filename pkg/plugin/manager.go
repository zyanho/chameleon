package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sync/errgroup"
)

// PluginState represents the state of a plugin
type PluginState int

const (
	StateActive PluginState = iota
	StateDeprecated
)

// PluginInstance wraps a plugin with additional metadata
type PluginInstance struct {
	*Plugin
	state   PluginState
	version string
}

// GetFunctions returns a list of available functions
func (pi *PluginInstance) GetFunctions() []string {
	return pi.Plugin.GetFunctions()
}

// Manager handles plugin lifecycle and operations
type Manager struct {
	plugins     sync.Map // map[string]*PluginInstance
	pluginPaths sync.Map // map[string]string
	watcher     *fsnotify.Watcher
	ctx         context.Context
	cancel      context.CancelFunc
	config      *Config
	logger      Logger
	metrics     *PluginMetrics
	breakers    sync.Map // map[string]*CircuitBreaker
	eg          *errgroup.Group
}

// ManagerOption defines a function type for configuring Manager
type ManagerOption func(*Manager)

// WithLogger sets an external logger implementation
func WithLogger(logger Logger) ManagerOption {
	return func(m *Manager) {
		if logger != nil {
			m.logger = logger
		}
	}
}

// NewManager creates a new plugin manager
func NewManager(ctx context.Context, config *Config, opts ...ManagerOption) (*Manager, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	m := &Manager{
		plugins:     sync.Map{},
		pluginPaths: sync.Map{},
		watcher:     watcher,
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
		logger:      NewDefaultLogger(config.LogLevel),
		metrics:     NewPluginMetrics(config.EnableMetrics),
		breakers:    sync.Map{},
		eg:          eg,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Start plugin directory watcher if enabled
	if config.AllowHotReload && config.PluginDir != "" {
		m.eg.Go(func() error {
			return m.watchPlugins(config.PluginDir)
		})
	}

	// Load plugins from directory if specified
	if config.PluginDir != "" {
		if err := m.loadPluginsFromDir(config.PluginDir); err != nil {
			m.Close()
			return nil, fmt.Errorf("failed to load plugins: %w", err)
		}
	}

	return m, nil
}

// LoadPlugin loads a plugin from the specified path
func (m *Manager) LoadPlugin(path string) error {
	return m.LoadPluginWithConfig(path, nil)
}

// LoadPluginWithConfig loads a plugin with specific configuration
func (m *Manager) LoadPluginWithConfig(path string, config *PluginSpecificConfig) error {
	pluginName := getPluginNameFromPath(path)

	// if no specific config is provided, use default config
	if config == nil {
		defaultConfig := m.config.DefaultPluginConfig
		config = &defaultConfig
	}

	// use Loader to load plugin first to get version
	loader := NewLoader(m)
	plugin, err := loader.Load(m.ctx, path)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Check for existing plugin
	if oldVal, exists := m.plugins.Load(pluginName); exists {
		oldInstance := oldVal.(*PluginInstance)
		// If new version is not higher, skip loading
		if !isHigherVersion(plugin.Version(), oldInstance.version) {
			plugin.Free()
			return nil
		}
		// Mark old version as deprecated
		oldInstance.state = StateDeprecated
	}

	// initialize plugin
	if err := plugin.Init(config.InitArgs...); err != nil {
		plugin.Free()
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// create circuit breaker
	breaker := NewCircuitBreaker(m.ctx, config.CircuitBreaker, m.logger)

	instance := &PluginInstance{
		Plugin:  plugin,
		state:   StateActive,
		version: plugin.Version(), // Use version from plugin
	}

	m.plugins.Store(pluginName, instance)
	m.pluginPaths.Store(pluginName, path)
	m.breakers.Store(pluginName, breaker)

	return nil
}

// Call invokes a plugin function with the given arguments
func (m *Manager) Call(ctx context.Context, pluginName, funcName string, args ...interface{}) (interface{}, error) {
	// get plugin instance
	instanceVal, exists := m.plugins.Load(pluginName)
	if !exists {
		return nil, &ErrPluginNotFound{Name: pluginName}
	}
	instance := instanceVal.(*PluginInstance)

	// get circuit breaker
	breakerVal, _ := m.breakers.Load(pluginName)
	breaker := breakerVal.(*CircuitBreaker)

	if breaker != nil && !breaker.Allow() {
		return nil, &ErrCircuitBreakerOpen{Name: pluginName}
	}

	start := time.Now()
	result, err := instance.Call(ctx, funcName, args...)
	duration := time.Since(start)

	if err != nil {
		if breaker != nil {
			breaker.RecordFailure()
		}
		return nil, err
	}

	if breaker != nil {
		breaker.RecordSuccess()
	}

	if m.metrics.IsEnabled() {
		m.metrics.RecordMetric(pluginName, funcName, duration)
	}

	return result, nil
}

// IsCircuitBreakerOpen checks if the circuit breaker is open for a plugin
func (m *Manager) IsCircuitBreakerOpen(pluginName string) bool {
	breakerVal, _ := m.breakers.Load(pluginName)
	breaker := breakerVal.(*CircuitBreaker)

	if breaker == nil {
		return false
	}
	return !breaker.Allow()
}

// ListPlugins returns a list of all loaded plugins
func (m *Manager) ListPlugins() []PluginInfo {
	var plugins []PluginInfo
	m.plugins.Range(func(key, value interface{}) bool {
		name := key.(string)
		instance := value.(*PluginInstance)
		plugins = append(plugins, PluginInfo{
			Name:    name,
			Version: instance.version,
			State:   instance.state,
		})
		return true
	})
	return plugins
}

// Close gracefully shuts down the manager and all plugins
func (m *Manager) Close() error {
	// Cancel context to signal shutdown
	m.cancel()

	// Wait for all background tasks to complete
	if err := m.eg.Wait(); err != nil {
		m.logger.Error("Error waiting for background tasks", "error", err)
	}

	// Close watcher
	if m.watcher != nil {
		if err := m.watcher.Close(); err != nil {
			m.logger.Error("Error closing watcher", "error", err)
		}
	}

	// Close circuit breakers
	m.breakers.Range(func(key, value interface{}) bool {
		name := key.(string)
		breaker := value.(*CircuitBreaker)
		if breaker != nil {
			breaker.Close()
			m.logger.Debug("Circuit breaker closed", "plugin", name)
		}
		return true
	})

	// Wait a bit for ongoing calls to complete
	time.Sleep(100 * time.Millisecond)

	// Clean up plugins
	var errs []error
	m.plugins.Range(func(key, value interface{}) bool {
		name := key.(string)
		instance := value.(*PluginInstance)
		if err := instance.Free(); err != nil {
			errs = append(errs, &ErrPluginFree{Name: name, Err: err})
		}
		m.plugins.Delete(key) // Explicitly remove the plugin
		m.logger.Debug("Plugin freed", "name", name)
		return true
	})

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}
	return nil
}

// Internal methods
func (m *Manager) watchPlugins(dir string) error {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("Panic in watchPlugins", "error", r)
		}
	}()

	if err := m.watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create == fsnotify.Create && strings.HasSuffix(event.Name, ".so") {
				m.handleNewPlugin(event.Name)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return nil
			}
			m.logger.Error("Watcher error", "error", err)
		case <-m.ctx.Done():
			return nil
		}
	}
}

func (m *Manager) handleNewPlugin(path string) {
	pluginName := getPluginNameFromPath(path)
	if config, exists := m.config.PluginConfigs[pluginName]; exists {
		if err := m.LoadPluginWithConfig(path, &config); err != nil {
			m.logger.Error("Failed to load new plugin", "path", path, "error", err)
		}
	} else {
		if err := m.LoadPlugin(path); err != nil {
			m.logger.Error("Failed to load new plugin", "path", path, "error", err)
		}
	}
}

func (m *Manager) loadPluginsFromDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".so") {
			pluginName := getPluginNameFromPath(path)
			if config, exists := m.config.PluginConfigs[pluginName]; exists {
				return m.LoadPluginWithConfig(path, &config)
			}
			return m.LoadPlugin(path)
		}
		return nil
	})
}

// Helper functions
func getPluginNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func isHigherVersion(new, current string) bool {
	v1 := strings.Split(strings.TrimPrefix(new, "v"), ".")
	v2 := strings.Split(strings.TrimPrefix(current, "v"), ".")

	// Ensure both version numbers have the same number of parts
	for len(v1) < 3 {
		v1 = append(v1, "0")
	}
	for len(v2) < 3 {
		v2 = append(v2, "0")
	}

	// Compare each part
	for i := 0; i < len(v1); i++ {
		n1, _ := strconv.Atoi(v1[i])
		n2, _ := strconv.Atoi(v2[i])
		if n1 > n2 {
			return true
		}
		if n1 < n2 {
			return false
		}
	}
	return false
}

// EnableMetrics enables metrics collection
func (m *Manager) EnableMetrics() {
	m.metrics.SetEnabled(true)
}

// DisableMetrics disables metrics collection
func (m *Manager) DisableMetrics() {
	m.metrics.SetEnabled(false)
}

// IsMetricsEnabled returns whether metrics collection is enabled
func (m *Manager) IsMetricsEnabled() bool {
	return m.metrics.IsEnabled()
}

// GetMetrics returns metrics for a specific plugin
func (m *Manager) GetMetrics(pluginName string) (*PluginMethodMetrics, error) {
	return m.metrics.GetPluginMetrics(pluginName)
}

// ResetMetrics resets all metrics
func (m *Manager) ResetMetrics() {
	m.metrics.plugins.Range(func(key, value interface{}) bool {
		m.metrics.plugins.Delete(key)
		return true
	})
}

func (m *Manager) GetBreakerStatus(pluginName string) bool {
	breakerVal, _ := m.breakers.Load(pluginName)
	breaker := breakerVal.(*CircuitBreaker)
	if breaker == nil {
		return false
	}
	return !breaker.Allow()
}

// GetPluginPath returns the path of a loaded plugin
func (m *Manager) GetPluginPath(name string) (string, bool) {
	if val, ok := m.pluginPaths.Load(name); ok {
		if path, ok := val.(string); ok {
			return path, true
		}
	}
	return "", false
}

// GetPluginFunctions returns a list of available functions for a plugin
func (m *Manager) GetPluginFunctions(pluginName string) ([]string, error) {
	val, ok := m.plugins.Load(pluginName)
	if !ok {
		return nil, ErrPluginNotFound{Name: pluginName}
	}

	instance := val.(*PluginInstance)
	return instance.GetFunctions(), nil
}
