module version-test-plugin-v2

go 1.23.3

require github.com/zyanho/chameleon v0.0.0-00010101000000-000000000000

require (
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
)

replace github.com/zyanho/chameleon => ../../..
