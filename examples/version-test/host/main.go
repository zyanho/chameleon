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
	pluginDir := filepath.Join("..", "plugins")

	// Create configuration
	config := plugin.DefaultConfig()
	config.PluginDir = ""
	config.AllowHotReload = true

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

	// Set specific plugin configuration
	config.PluginConfigs["version-test-plugin"] = plugin.PluginSpecificConfig{
		InitArgs:      []interface{}{"init-arg1", "init-arg2"},
		PluginTimeout: 30 * time.Second,
		CircuitBreaker: plugin.CircuitBreakerConfig{
			Enabled:         true,
			MaxFailures:     3,
			ResetInterval:   30 * time.Second,
			TimeoutDuration: 5 * time.Second,
		},
	}

	// Create plugin manager
	manager, err := plugin.NewManager(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Print initial state
	printPluginInfo(manager, "Initial state")

	// Enable performance statistics
	manager.EnableMetrics()

	// 1. Load the first version of the plugin
	fmt.Println("\nLoading version 1.0.0...")
	err = manager.LoadPlugin(filepath.Join(pluginDir, "v1/version-test-plugin.so"))
	if err != nil {
		log.Fatal(err)
	}

	// Print current plugin information
	printPluginInfo(manager, "After loading v1")

	// Start a long-running operation
	fmt.Println("\nStarting long running operation...")
	go func() {
		longResult, err := manager.Call(ctx, "version-test-plugin", "LongRunning", 10)
		if err != nil {
			log.Printf("Long running operation error: %v\n", err)
		} else {
			log.Printf("Long running result: %v\n", longResult)
		}
	}()

	// Wait for a while to let the long-running operation start
	time.Sleep(2 * time.Second)

	// Get current version
	result, err := manager.Call(ctx, "version-test-plugin", "GetVersion", []interface{}{}...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nCurrent version: %v\n", result)

	// 2. Load new version plugin
	fmt.Println("\nLoading version 2.0.0...")
	err = manager.LoadPlugin(filepath.Join(pluginDir, "v2/version-test-plugin.so"))
	if err != nil {
		log.Fatal(err)
	}

	// Print updated plugin information
	printPluginInfo(manager, "After loading v2")

	// 3. Start new call to verify if the new version is used
	result, err = manager.Call(ctx, "version-test-plugin", "GetVersion", []interface{}{}...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nNew version: %v\n", result)

	// 4. Wait for the old version operation to complete and unload
	fmt.Println("\nWaiting for old version to be freed...")
	time.Sleep(12 * time.Second)

	// Final state
	printPluginInfo(manager, "Final state")

	// Print performance statistics
	printMetrics(manager)
}

// printPluginInfo prints current plugin information
func printPluginInfo(manager *plugin.Manager, title string) {
	fmt.Printf("\n=== %s ===\n", title)
	plugins := manager.ListPlugins()
	for _, p := range plugins {
		fmt.Printf("Plugin: %s\n", p.Name)
		fmt.Printf("  Version: %s\n", p.Version)
		fmt.Printf("  State: %v\n", p.State)
		fmt.Printf("  RefCount: %d\n", p.RefCount)
		fmt.Printf("  Path: %s\n", p.Path)
	}
}

// printMetrics prints performance metrics for all plugins
func printMetrics(manager *plugin.Manager) {
	fmt.Printf("\n=== Performance Metrics ===\n")
	plugins := manager.ListPlugins()

	for _, p := range plugins {
		fmt.Printf("\nPlugin: %s\n", p.Name)
		metrics, err := manager.GetMetrics(p.Name)
		if err != nil {
			fmt.Printf("Error getting metrics for plugin %s: %v\n", p.Name, err)
			continue
		}

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
