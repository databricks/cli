default: build

fmt:
	@echo "✓ Formatting source code with goimports ..."
	@goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
	@echo "✓ Formatting source code with gofmt ..."
	@gofmt -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

lint: vendor
	@echo "✓ Linting source code with https://golangci-lint.run/ (with --fix)..."
	@golangci-lint run --fix ./...

lintcheck: vendor
	@echo "✓ Linting source code with https://golangci-lint.run/ ..."
	@golangci-lint run ./...

test: lint testonly

testonly:
	@echo "✓ Running tests ..."
	@gotestsum --format pkgname-and-test-fails --no-summary=skipped --raw-command go test -v -json -short -coverprofile=coverage.txt ./...

coverage: test
	@echo "✓ Opening coverage for unit tests ..."
	@go tool cover -html=coverage.txt

build: vendor
	@echo "✓ Building source code with go build ..."
	@go build -mod vendor

snapshot:
	@echo "✓ Building dev snapshot"
	@go build -o .databricks/databricks

vendor:
	@echo "✓ Filling vendor folder with library code ..."
	@go mod vendor

integration:
	gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./integration/..." -- -parallel 4 -timeout=2h

schema:
	@echo "✓ Generating json-schema ..."
	@go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema.json

.PHONY: fmt lint lintcheck test testonly coverage build snapshot vendor integration schema
