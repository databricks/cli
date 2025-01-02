default: build

PACKAGES=./libs/... ./internal/... ./cmd/... ./bundle/... .

lint:
	@echo "✓ Linting source code with https://golangci-lint.run/ (with --fix)..."
	@./lint.sh ./...

lintcheck:
	@echo "✓ Linting source code with https://golangci-lint.run/ ..."
	@golangci-lint run ./...

test:
	@echo "✓ Running tests ..."
	@gotestsum --format pkgname-and-test-fails --no-summary=skipped -- ${PACKAGES}

cover:
	@echo "✓ Running tests with coverage..."
	@gotestsum --format pkgname-and-test-fails --no-summary=skipped -- -coverprofile=coverage.txt ${PACKAGES}

showcover:
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
  
schema:
	@echo "✓ Generating json-schema ..."
	@go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema.json

INTEGRATION = gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./integration/..." -- -parallel 4 -timeout=2h

integration:
	$(INTEGRATION)

integration-short:
	$(INTEGRATION) -short

.PHONY: lint lintcheck test cover showcover build snapshot vendor schema integration integration-short
