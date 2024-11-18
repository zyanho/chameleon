# Chameleon 插件系统

🚀 高性能：无锁原子操作，高效并发访问
🛡️ 安全性：内置熔断器，panic 恢复，优雅关闭
🔌 灵活性：支持热重载，版本控制，独立插件依赖
📊 可观测：内置指标统计，方法级性能监控一个高性能、安全且灵活的 Go 语言插件系统。

[English](README.md)

## 特性

### 高性能

- 使用无锁原子操作进行状态管理
- 基于 sync.Map 的高效并发映射实现
- 极低的插件调用开销
- 零拷贝的插件加载和卸载
- 优化的内存使用和资源清理

### 安全性和稳定性

- 内置熔断器模式，实现故障隔离
- 支持优雅关闭
- 全面的 goroutine panic 恢复机制
- 完善的资源清理
- 全面的错误处理

### 插件管理

- 支持静态热更新机制
- 版本控制和升级管理
- 插件状态监控
- 可配置的超时和重试策略
- 独立的插件依赖管理

### 指标监控

- 内置性能指标收集
- 方法级别的计时统计
- 可自定义的日志系统
- 熔断器状态监控

## 安装

```bash
go get -u github.com/zyanho/chameleon
```

## 快速开始

1. 创建插件：

```go
// plugin/hello/main.go
package main

type HelloPlugin struct{}
func (p HelloPlugin) Name() string { return "hello" }
func (p HelloPlugin) Version() string { return "1.0.0" }
func (p HelloPlugin) Init() error { return nil }
func (p HelloPlugin) Free() error { return nil }
func (p HelloPlugin) Greet(name string) string {
  return "你好, " + name
}
var Export = &HelloPlugin{}
```

2. 构建插件：

```bash
go install github.com/zyanho/chameleon/cmd/chameleon@latest
chameleon build ./plugin/hello
```

3. 使用插件:

```go

package main

import "github.com/zyanho/chameleon/pkg/plugin"

func main() {
  // 初始化插件管理器
  config := plugin.DefaultConfig()
  config.PluginDir = "./plugins"
  manager, err := plugin.NewManager(config)
  if err != nil {
    panic(err)
  }
  defer manager.Close()
  // 调用插件函数
  result, err := manager.Call(ctx, "hello", "Greet", "世界")
  if err != nil {
    panic(err)
  }
  fmt.Println(result) // 输出: 你好, 世界
}
```

## 高级特性

### 熔断器

内置熔断器模式保护系统免受级联故障的影响：

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

### 动态加载

支持带版本控制的插件动态加载：

```go
if err := manager.LoadPlugin("./plugins/hello.so"); err != nil {
  log.Printf("插件重载失败: %v", err)
}
```

### 指标收集

内置性能指标收集：

```go
metrics, err := manager.GetMetrics("hello")
if err != nil {
  log.Printf("获取指标失败: %v", err)
}
```

### 可配置的日志系统

支持自定义日志实现：

```go
type CustomLogger struct {
// 自定义字段
}
func (l CustomLogger) Debug(msg string, args ...interface{}) {
// 自定义实现
}
// 设置自定义日志器
config.Logger = &CustomLogger{}
```

## 性能

### 高效的资源管理

- 使用原子操作实现近乎零开销的插件调用
- 基于 sync.Map 的无锁并发访问
- 最小化内存分配和垃圾回收
- 快速的插件加载和卸载，确保资源及时清理
- 使用原子操作优化状态管理

### 性能指标

- 插件调用延迟：< 1ms
- 每个插件的内存开销：约 100KB
- 并发调用能力：单插件 10k+ QPS
- 动态加载时间：< 100ms

## 系统要求

- Go 1.21 或更高版本
- Linux/macOS（Go原生插件系统不支持Windows）

## 许可证

Apache-2.0 许可证

## 贡献

欢迎提交 Pull Request 来帮助改进项目！
