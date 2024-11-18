module example-logger

go 1.23.3

require (
	github.com/zyanho/chameleon v0.0.0-20241116174048-9d04c40ebaf1
	go.uber.org/zap v1.27.0
)

require (
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
)

replace github.com/zyanho/chameleon => ../..
