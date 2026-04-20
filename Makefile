.PHONY: lint build-tools

build-tools:
	go build -o tools/bin/unexported tools/unexported/main.go

lint: build-tools
	golangci-lint run
	@echo "Running custom unexported analyzer..."
	@./tools/bin/unexported --allowlist "main"
