.PHONY: lint build-tools docs-update docs-update-dry-run test-docs-update

build-tools:
	go build -o tools/bin/unexported tools/unexported/main.go

lint: build-tools
	golangci-lint run
	@echo "Running custom unexported analyzer..."
	@./tools/bin/unexported --allowlist "main,candidate_alocator/back/allocation,candidate_alocator/back/db,candidate_alocator/back/logic,candidate_alocator/back/type,candidate_alocator/back/workflow"

docs-update:
	@./scripts/update-docs

docs-update-dry-run:
	@./scripts/update-docs --dry-run

test-docs-update:
	@./scripts/update-docs.test.sh
