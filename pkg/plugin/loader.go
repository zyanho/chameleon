package plugin

import (
	"context"
	"fmt"
	"plugin"
	"sync"
)

// Loader handles plugin loading and validation
type Loader struct {
	manager *Manager
	cache   sync.Map
	logger  Logger
}

// NewLoader creates a new plugin loader
func NewLoader(manager *Manager) *Loader {
	return &Loader{
		manager: manager,
		logger:  manager.logger,
	}
}

// Load loads a plugin from the specified path
func (l *Loader) Load(ctx context.Context, path string) (*Plugin, error) {
	if cached, ok := l.cache.Load(path); ok {
		l.logger.Debug("Using cached plugin", "path", path)
		return cached.(*Plugin), nil
	}

	pluginConfig := l.manager.config.DefaultPluginConfig
	timeoutCtx, cancel := context.WithTimeout(ctx, pluginConfig.PluginTimeout)
	defer cancel()

	done := make(chan struct{})
	var plug *plugin.Plugin
	var err error

	go func() {
		plug, err = plugin.Open(path)
		close(done)
	}()

	select {
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("plugin load timeout: %w", timeoutCtx.Err())
	case <-done:
		if err != nil {
			return nil, fmt.Errorf("failed to open plugin: %w", err)
		}
	}

	p, err := l.validateAndCreatePlugin(plug)
	if err != nil {
		return nil, err
	}

	l.cache.Store(path, p)
	return p, nil
}

func (l *Loader) validateAndCreatePlugin(plug *plugin.Plugin) (*Plugin, error) {
	// find the Export symbol
	sym, err := plug.Lookup("Export")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export 'Export' symbol: %w", err)
	}

	l.logger.Debug("Found Export symbol", "type", fmt.Sprintf("%T", sym))

	// validate and convert to Bureau interface
	bureau, ok := sym.(*Bureau)
	if !ok {
		return nil, fmt.Errorf("exported symbol is not a *Bureau: got type %T", sym)
	}

	// create plugin instance
	p := NewPlugin(*bureau)

	// find and validate the Functions symbol
	funcsSym, err := plug.Lookup("Functions")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export 'Functions' symbol: %w", err)
	}

	l.logger.Debug("Found Functions symbol", "type", fmt.Sprintf("%T", funcsSym))

	// validate and convert to map[string]InvokeFunc
	funcsMap, ok := funcsSym.(*map[string]InvokeFunc)
	if !ok {
		return nil, fmt.Errorf("Functions is not a *map[string]InvokeFunc: got type %T", funcsSym)
	}

	// register functions
	for name, fn := range *funcsMap {
		if err := l.validateFunc(name, fn); err != nil {
			return nil, fmt.Errorf("invalid function %s: %w", name, err)
		}
		p.RegisterFunc(name, fn)
	}

	return p, nil
}

func (l *Loader) validateFunc(name string, fn InvokeFunc) error {
	if name == "" {
		return fmt.Errorf("empty function name")
	}
	if fn == nil {
		return fmt.Errorf("nil function")
	}
	return nil
}
