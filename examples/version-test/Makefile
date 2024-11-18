.PHONY: build-generator build-plugin build-host run-example

build-generator:
	@echo "Building generator..."
	@cd ../../cmd/chameleon && go build -o ../../bin/chameleon

build-plugin: build-generator
	@echo "Building version test plugins..."
	@mkdir -p plugins
	@mkdir -p plugins/v1
	@mkdir -p plugins/v2
	@echo "Generating plugin wrapper for v1..."
	@../../bin/chameleon generate plugin
	@echo "Building plugin v1..."
	@cd plugin && \
		go build -buildmode=plugin -o ../plugins/v1/version-test-plugin.so *.go
	@echo "Generating plugin wrapper for v2..."
	@../../bin/chameleon generate pluginv2
	@echo "Building plugin v2..."
	@cd pluginv2 && \
		go build -buildmode=plugin -o ../plugins/v2/version-test-plugin.so *.go

build-host:
	@echo "Building version test host..."
	@cd host && go build -o host

run-example: build-plugin build-host
	@cd host && ./host

run-plugin: build-plugin
	@echo "Running version..."
	@cd host && ./host

clean:
	@rm -f host/host
	@rm -f plugins/v1/*.so
	@rm -f plugins/v2/*.so
	@rm -f plugin/plugin_wrapper.go
	@rm -f pluginv2/plugin_wrapper.go