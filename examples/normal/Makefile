.PHONY: build-generator build-plugin build-host run-example

build-generator:
	@echo "Building generator..."
	@cd ../../cmd/chameleon && go build -o ../../bin/chameleon

build-plugin: build-generator
	@echo "Building normal plugins..."
	@mkdir -p plugins
	@echo "Generating plugin wrapper..."
	@../../bin/chameleon generate plugin
	@echo "Building plugin..."
	@cd plugin && go build -buildmode=plugin -o ../plugins/example-plugin.so

build-host:
	@echo "Building host..."
	@cd host && go build -o host

run-example: build-plugin build-host
	@echo "Running example..."
	@cd host && ./host

run-plugin: build-plugin
	@echo "Running example..."
	@cd host && ./host

clean:
	@rm -f ../../bin/chameleon
	@rm -f host/host
	@rm -f plugins/*.so
	@rm -f plugin/plugin_wrapper.go