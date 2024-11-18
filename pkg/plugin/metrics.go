package plugin

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MethodMetrics stores metrics for a single method using atomic operations
type MethodMetrics struct {
	Count     atomic.Int64
	TotalTime atomic.Int64 // save nanoseconds
	MinTime   atomic.Int64 // save nanoseconds
	MaxTime   atomic.Int64 // save nanoseconds
}

// PluginMethodMetrics stores metrics for plugin methods
type PluginMethodMetrics struct {
	Methods sync.Map // map[string]*MethodMetrics
}

// PluginMetrics stores metrics for plugin calls
type PluginMetrics struct {
	plugins sync.Map // map[string]*PluginMethodMetrics
	enabled atomic.Bool
}

// NewPluginMetrics creates a new plugin metrics collector
func NewPluginMetrics(enabled bool) *PluginMetrics {
	m := &PluginMetrics{}
	m.enabled.Store(enabled)
	return m
}

// SetEnabled sets the enabled state
func (m *PluginMetrics) SetEnabled(enabled bool) {
	m.enabled.Store(enabled)
}

// IsEnabled returns the enabled state
func (m *PluginMetrics) IsEnabled() bool {
	return m.enabled.Load()
}

// AddPlugin adds a new plugin metrics record
func (m *PluginMetrics) AddPlugin(pluginName string) {
	if !m.enabled.Load() {
		return
	}
	m.plugins.LoadOrStore(pluginName, &PluginMethodMetrics{})
}

// RecordMetric records a single method call
func (m *PluginMetrics) RecordMetric(pluginName, funcName string, duration time.Duration) {
	if !m.enabled.Load() {
		return
	}

	// Get or create plugin metrics
	pluginMetrics, _ := m.plugins.LoadOrStore(pluginName, &PluginMethodMetrics{})
	pMetrics := pluginMetrics.(*PluginMethodMetrics)

	// Get or create method metrics
	methodMetricsIface, _ := pMetrics.Methods.LoadOrStore(funcName, &MethodMetrics{})
	metrics := methodMetricsIface.(*MethodMetrics)

	durationNanos := duration.Nanoseconds()

	// Update count and total time
	metrics.Count.Add(1)
	metrics.TotalTime.Add(durationNanos)

	// Update min time using CAS loop
	for {
		current := metrics.MinTime.Load()
		if current == 0 || durationNanos < current {
			if metrics.MinTime.CompareAndSwap(current, durationNanos) {
				break
			}
		} else {
			break
		}
	}

	// Update max time using CAS loop
	for {
		current := metrics.MaxTime.Load()
		if durationNanos > current {
			if metrics.MaxTime.CompareAndSwap(current, durationNanos) {
				break
			}
		} else {
			break
		}
	}
}

// GetPluginMetrics returns metrics for a specific plugin
func (m *PluginMetrics) GetPluginMetrics(pluginName string) (*PluginMethodMetrics, error) {
	if !m.enabled.Load() {
		return nil, fmt.Errorf("metrics are disabled")
	}

	pluginMetricsIface, exists := m.plugins.Load(pluginName)
	if !exists {
		return nil, fmt.Errorf("no metrics found for plugin: %s", pluginName)
	}

	pMetrics := pluginMetricsIface.(*PluginMethodMetrics)

	// Create a snapshot
	snapshot := &PluginMethodMetrics{
		Methods: sync.Map{},
	}

	// use Range to iterate over sync.Map
	pMetrics.Methods.Range(func(key, value interface{}) bool {
		methodName := key.(string)
		metrics := value.(*MethodMetrics)

		// Create method snapshot
		methodSnapshot := &MethodMetrics{}
		methodSnapshot.Count.Store(metrics.Count.Load())
		methodSnapshot.TotalTime.Store(metrics.TotalTime.Load())
		methodSnapshot.MinTime.Store(metrics.MinTime.Load())
		methodSnapshot.MaxTime.Store(metrics.MaxTime.Load())

		snapshot.Methods.Store(methodName, methodSnapshot)
		return true
	})

	return snapshot, nil
}
