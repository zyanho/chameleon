package plugin

// PluginInfo contains basic information about a loaded plugin
type PluginInfo struct {
	Name     string
	Version  string
	State    PluginState
	RefCount int32
	Path     string
}
