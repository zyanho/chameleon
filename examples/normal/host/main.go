package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/zyanho/chameleon/pkg/plugin"
)

func main() {
	ctx := context.Background()

	// Get absolute path for plugin directory
	pluginDir := filepath.Join("..", "plugins")
	absPath, err := filepath.Abs(pluginDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loading plugins from: %s\n", absPath)

	// Create configuration
	config := plugin.DefaultConfig()
	config.PluginDir = absPath
	config.AllowHotReload = true
	config.LogLevel = plugin.LogLevelDebug

	// Set default plugin configuration
	config.DefaultPluginConfig = plugin.PluginSpecificConfig{
		InitArgs:           []interface{}{"init-arg1", "init-arg2"},
		PluginTimeout:      30 * time.Second,
		MaxConcurrentCalls: 100,
		CircuitBreaker: plugin.CircuitBreakerConfig{
			Enabled:         true,
			MaxFailures:     5,
			ResetInterval:   60 * time.Second,
			TimeoutDuration: 5 * time.Second,
		},
		Options: make(map[string]interface{}),
	}

	// Create plugin manager
	manager, err := plugin.NewManager(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Enable performance statistics
	manager.EnableMetrics()

	// List all loaded plugins
	fmt.Println("\nLoaded plugins:")
	plugins := manager.ListPlugins()
	for _, info := range plugins {
		fmt.Printf("- %s v%s (state: %v)\n", info.Name, info.Version, info.State)

		// Print plugin functions
		funcs, err := manager.GetPluginFunctions(info.Name)
		if err == nil {
			fmt.Println("  Functions:")
			for _, fn := range funcs {
				fmt.Printf("    - %s\n", fn)
			}
		}
	}

	// Call plugin functions
	result, err := manager.Call(ctx, "example-plugin", "Add", 1, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nAdd Result: %v\n", result)

	result, err = manager.Call(ctx, "example-plugin", "Some1111", 1, 2, "2222")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Some1111 Result: %v\n", result)

	result, err = manager.Call(ctx, "example-plugin", "Some1111", 1, 2, "1111")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Some1111 Result: %v\n", result)

	// Print detailed plugin information
	printPluginInfo(manager, "Current State")

	// Print performance statistics
	printMetrics(manager)
}

// printPluginInfo prints detailed information about loaded plugins
func printPluginInfo(manager *plugin.Manager, title string) {
	fmt.Printf("\n=== %s ===\n", title)
	plugins := manager.ListPlugins()
	for _, p := range plugins {
		fmt.Printf("Plugin: %s\n", p.Name)
		fmt.Printf("  Version: %s\n", p.Version)
		fmt.Printf("  State: %s\n", stateToString(p.State))
		fmt.Printf("  RefCount: %d\n", p.RefCount)
		fmt.Printf("  Path: %s\n", p.Path)

		// Print registered functions
		funcs, err := manager.GetPluginFunctions(p.Name)
		if err == nil {
			fmt.Printf("  Functions:\n")
			for _, fn := range funcs {
				fmt.Printf("    - %s\n", fn)
			}
		}
	}
}

// printMetrics prints performance metrics for all plugins
func printMetrics(manager *plugin.Manager) {
	fmt.Printf("\n=== Performance Metrics ===\n")
	plugins := manager.ListPlugins()

	for _, p := range plugins {
		metrics, err := manager.GetMetrics(p.Name)
		if err != nil {
			fmt.Printf("Error getting metrics for plugin %s: %v\n", p.Name, err)
			continue
		}

		fmt.Printf("\nPlugin: %s\n", p.Name)
		fmt.Printf("Methods:\n")

		// use Range to iterate over sync.Map
		metrics.Methods.Range(func(key, value interface{}) bool {
			methodName := key.(string)
			methodMetrics := value.(*plugin.MethodMetrics)

			fmt.Printf("  %s:\n", methodName)
			fmt.Printf("    Call Count: %d\n", methodMetrics.Count.Load())
			fmt.Printf("    Total Time: %v\n", time.Duration(methodMetrics.TotalTime.Load()))
			fmt.Printf("    Min Time: %v\n", time.Duration(methodMetrics.MinTime.Load()))
			fmt.Printf("    Max Time: %v\n", time.Duration(methodMetrics.MaxTime.Load()))

			count := methodMetrics.Count.Load()
			if count > 0 {
				avgTime := time.Duration(methodMetrics.TotalTime.Load()) / time.Duration(count)
				fmt.Printf("    Avg Time: %v\n", avgTime)
			}
			return true
		})

		// Print circuit breaker status
		breakerStatus := "Closed"
		if manager.GetBreakerStatus(p.Name) {
			breakerStatus = "Open"
		}
		fmt.Printf("\n  Circuit Breaker Status: %s\n", breakerStatus)
	}
}

// stateToString converts plugin state to string representation
func stateToString(state plugin.PluginState) string {
	switch state {
	case plugin.StateActive:
		return "Active"
	case plugin.StateDeprecated:
		return "Deprecated"
	default:
		return "Unknown"
	}
}
