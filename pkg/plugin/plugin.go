package plugin

import (
	"context"
	"sync"
	"sync/atomic"
)

// Bureau defines the interface that all plugins must implement
type Bureau interface {
	Name() string
	Version() string
	Init(...interface{}) error
	Free() error
}

// InvokeFunc represents a plugin function with a context as its first parameter
type InvokeFunc func(ctx context.Context, args ...interface{}) (interface{}, error)

// Plugin wraps a plugin instance
type Plugin struct {
	sync.RWMutex
	bureau Bureau
	funcs  map[string]InvokeFunc
	refs   int32
}

func NewPlugin(b Bureau) *Plugin {
	return &Plugin{
		bureau: b,
		funcs:  make(map[string]InvokeFunc),
	}
}

func (p *Plugin) Name() string {
	return p.bureau.Name()
}

func (p *Plugin) Version() string {
	return p.bureau.Version()
}

func (p *Plugin) Init(args ...interface{}) error {
	return p.bureau.Init(args...)
}

func (p *Plugin) Free() error {
	return p.bureau.Free()
}

func (p *Plugin) RegisterFunc(name string, fn InvokeFunc) {
	p.funcs[name] = fn
}

// AddRef increases the reference count
func (p *Plugin) AddRef() {
	atomic.AddInt32(&p.refs, 1)
}

// DecRef decreases the reference count and returns whether it is 0
func (p *Plugin) DecRef() bool {
	return atomic.AddInt32(&p.refs, -1) == 0
}

// GetRefs gets the current reference count
func (p *Plugin) GetRefs() int32 {
	return atomic.LoadInt32(&p.refs)
}

// Call calls the plugin function
func (p *Plugin) Call(ctx context.Context, name string, args ...interface{}) (interface{}, error) {
	p.RLock()
	fn, ok := p.funcs[name]
	p.RUnlock()

	if !ok {
		return nil, ErrFuncNotFound{Name: name}
	}

	result, err := fn(ctx, args...)
	return result, err
}

// GetFunctions returns a list of available functions
func (p *Plugin) GetFunctions() []string {
	funcs := make([]string, 0, len(p.funcs))
	for name := range p.funcs {
		funcs = append(funcs, name)
	}
	return funcs
}
