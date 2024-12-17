default: build

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

INTEGRATION = gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./integration/..." -- -parallel 4 -timeout=2h

integration:
	$(INTEGRATION)

integration-short:
	$(INTEGRATION) -short

.PHONY: lint lintcheck test testonly coverage build snapshot vendor integration integration-short
