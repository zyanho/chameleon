package plugin

import (
	"fmt"
	"time"
)

// LogLevel defines the severity level for logging
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// CircuitBreakerConfig defines configuration for the circuit breaker
type CircuitBreakerConfig struct {
	Enabled         bool
	MaxFailures     int
	ResetInterval   time.Duration
	TimeoutDuration time.Duration
}

// PluginSpecificConfig defines configuration for a specific plugin
type PluginSpecificConfig struct {
	InitArgs           []interface{}
	CircuitBreaker     CircuitBreakerConfig
	MaxConcurrentCalls int
	PluginTimeout      time.Duration
	Options            map[string]interface{}
}

// Config defines the configuration for plugin manager
type Config struct {
	PluginDir           string
	AllowHotReload      bool
	LogLevel            LogLevel
	EnableMetrics       bool
	DefaultPluginConfig PluginSpecificConfig
	PluginConfigs       map[string]PluginSpecificConfig
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:         true,
		MaxFailures:     5,
		ResetInterval:   60 * time.Second,
		TimeoutDuration: 5 * time.Second,
	}
}

// DefaultPluginSpecificConfig returns the default plugin specific configuration
func DefaultPluginSpecificConfig() PluginSpecificConfig {
	return PluginSpecificConfig{
		InitArgs:           []interface{}{},
		CircuitBreaker:     DefaultCircuitBreakerConfig(),
		MaxConcurrentCalls: 100,
		PluginTimeout:      30 * time.Second,
		Options:            make(map[string]interface{}),
	}
}

// DefaultConfig returns the default plugin manager configuration
func DefaultConfig() *Config {
	return &Config{
		PluginDir:           "",
		AllowHotReload:      true,
		LogLevel:            LogLevelInfo,
		EnableMetrics:       true,
		DefaultPluginConfig: DefaultPluginSpecificConfig(),
		PluginConfigs:       make(map[string]PluginSpecificConfig),
	}
}

// GetPluginConfig gets the plugin configuration, returning the default configuration if no specific configuration is provided
func (c *Config) GetPluginConfig(pluginName string) PluginSpecificConfig {
	if config, exists := c.PluginConfigs[pluginName]; exists {
		return mergeConfig(c.DefaultPluginConfig, config)
	}
	return c.DefaultPluginConfig
}

// mergeConfig merges two configurations, using the specific configuration to override the default configuration
func mergeConfig(defaultConfig, specificConfig PluginSpecificConfig) PluginSpecificConfig {
	merged := defaultConfig

	// If the specific configuration provides initialization arguments, use the arguments from the specific configuration
	if len(specificConfig.InitArgs) > 0 {
		merged.InitArgs = specificConfig.InitArgs
	}

	// If the specific configuration provides a circuit breaker, use the circuit breaker from the specific configuration
	if specificConfig.CircuitBreaker.Enabled {
		merged.CircuitBreaker = specificConfig.CircuitBreaker
	}

	// If the specific configuration provides a maximum number of concurrent calls, use the value from the specific configuration
	if specificConfig.MaxConcurrentCalls > 0 {
		merged.MaxConcurrentCalls = specificConfig.MaxConcurrentCalls
	}
	if specificConfig.PluginTimeout > 0 {
		merged.PluginTimeout = specificConfig.PluginTimeout
	}

	// If the specific configuration provides options, use the options from the specific configuration
	for k, v := range specificConfig.Options {
		merged.Options[k] = v
	}

	return merged
}

// ValidateConfig validates the configuration to ensure it is valid
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate the default configuration
	if err := validatePluginSpecificConfig(config.DefaultPluginConfig); err != nil {
		return fmt.Errorf("invalid default plugin config: %w", err)
	}

	// Validate the specific plugin configurations
	for name, pluginConfig := range config.PluginConfigs {
		if err := validatePluginSpecificConfig(pluginConfig); err != nil {
			return fmt.Errorf("invalid config for plugin %s: %w", name, err)
		}
	}

	return nil
}

// validatePluginSpecificConfig validates the plugin specific configuration to ensure it is valid
func validatePluginSpecificConfig(config PluginSpecificConfig) error {
	if config.MaxConcurrentCalls < 0 {
		return fmt.Errorf("MaxConcurrentCalls cannot be negative")
	}
	if config.PluginTimeout < 0 {
		return fmt.Errorf("PluginTimeout cannot be negative")
	}
	if config.CircuitBreaker.Enabled {
		if config.CircuitBreaker.MaxFailures <= 0 {
			return fmt.Errorf("CircuitBreaker MaxFailures must be positive")
		}
		if config.CircuitBreaker.ResetInterval <= 0 {
			return fmt.Errorf("CircuitBreaker ResetInterval must be positive")
		}
		if config.CircuitBreaker.TimeoutDuration <= 0 {
			return fmt.Errorf("CircuitBreaker TimeoutDuration must be positive")
		}
	}
	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := &Config{
		PluginDir:           c.PluginDir,
		AllowHotReload:      c.AllowHotReload,
		LogLevel:            c.LogLevel,
		EnableMetrics:       c.EnableMetrics,
		DefaultPluginConfig: clonePluginSpecificConfig(c.DefaultPluginConfig),
		PluginConfigs:       make(map[string]PluginSpecificConfig),
	}

	for name, config := range c.PluginConfigs {
		clone.PluginConfigs[name] = clonePluginSpecificConfig(config)
	}

	return clone
}

// clonePluginSpecificConfig creates a deep copy of the plugin specific configuration
func clonePluginSpecificConfig(config PluginSpecificConfig) PluginSpecificConfig {
	clone := PluginSpecificConfig{
		InitArgs:           make([]interface{}, len(config.InitArgs)),
		CircuitBreaker:     config.CircuitBreaker,
		MaxConcurrentCalls: config.MaxConcurrentCalls,
		PluginTimeout:      config.PluginTimeout,
		Options:            make(map[string]interface{}),
	}

	copy(clone.InitArgs, config.InitArgs)
	for k, v := range config.Options {
		clone.Options[k] = v
	}

	return clone
}
