# Chameleon Plugin System

🚀 High Performance: Lock-free atomic operations, efficient concurrent access
🛡️ Safety: Built-in circuit breaker, panic recovery, graceful shutdown
🔌 Flexible: Hot reload support, version control, independent plugin dependency
📊 Observable: Built-in metrics, method-level timing statistics

[中文文档](README_zh.md)

## Features

### High Performance

- Lock-free atomic operations for state management
- Efficient concurrent map implementations using sync.Map
- Minimal overhead for plugin calls
- Zero-copy plugin loading and unloading
- Optimized memory usage with proper resource cleanup

### Safety & Stability

- Built-in circuit breaker pattern for fault tolerance
- Graceful shutdown support
- Panic recovery in all goroutines
- Proper resource cleanup
- Comprehensive error handling

### Plugin Management

- Hot reload support with static swap mechanism
- Version control and upgrade management
- Plugin state monitoring
- Configurable timeout and retry policies
- Independent plugin dependency management

### Metrics & Monitoring

- Built-in performance metrics collection
- Method-level timing statistics
- Customizable logging system
- Circuit breaker state monitoring

## Installation

```bash
go get -u github.com/zyanho/chameleon
```

## Quick Start

1. Create a plugin:

```go
// plugin/hello/main.go
package main

type HelloPlugin struct{}
func (p HelloPlugin) Name() string { return "hello" }
func (p HelloPlugin) Version() string { return "1.0.0" }
func (p HelloPlugin) Init() error { return nil }
func (p HelloPlugin) Free() error { return nil }
func (p HelloPlugin) Greet(name string) string {
  return "Hello, " + name
}
var Export = &HelloPlugin{}
```

2. Build the plugin:

```bash
go install github.com/zyanho/chameleon/cmd/chameleon@latest
chameleon build ./plugin/hello
```

3. Use the plugin:

```go

package main

import "github.com/zyanho/chameleon/pkg/plugin"

func main() {
  // Initialize plugin manager
  config := plugin.DefaultConfig()
  config.PluginDir = "./plugins"
  manager, err := plugin.NewManager(config)
  if err != nil {
    panic(err)
  }
  defer manager.Close()
  // Call plugin function
  result, err := manager.Call(ctx, "hello", "Greet", "World")
  if err != nil {
    panic(err)
  }
  fmt.Println(result) // Output: Hello, World
}
```

## Advanced Features

### Circuit Breaker

Built-in circuit breaker pattern protects your system from cascading failures:

```go
config.PluginConfigs["hello"] = plugin.PluginSpecificConfig{
  CircuitBreaker: plugin.CircuitBreakerConfig{
    Enabled: true,
    MaxFailures: 5,
    ResetInterval: 60 time.Second,
    TimeoutDuration: 5 time.Second,
  },
}
```

### Hot Reload

Supports plugin hot reloading with version control:

```go
if err := manager.LoadPlugin("./plugins/hello.so"); err != nil {
  log.Printf("Failed to reload plugin: %v", err)
}
```

### Metrics Collection

Built-in performance metrics:

```go
metrics, err := manager.GetMetrics("hello")
if err != nil {
  log.Printf("Failed to get metrics: %v", err)
}
```

### Configurable Logging System

Support for custom logger implementation:

```go
type CustomLogger struct {
// Custom fields
}
func (l CustomLogger) Debug(msg string, args ...interface{}) {
// Custom implementation
}
// Set custom logger
config.Logger = &CustomLogger{}
```

## Performance

### Efficient Resource Management

- Near-zero overhead for plugin calls using atomic operations
- Lock-free concurrent access with sync.Map
- Minimal memory allocation and garbage collection
- Fast plugin loading and unloading with proper cleanup
- Optimized state management with atomic operations

### Benchmarks

- Plugin call latency: < 1ms
- Memory overhead per plugin: ~100KB
- Concurrent calls: 10k+ QPS per plugin
- Hot reload time: < 100ms

## Requirements

- Go 1.21 or higher
- Linux/macOS (Go's plugin doesn't support Windows)

## License

Apache-2.0 license

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
