default: build

PACKAGES=./libs/... ./internal/... ./cmd/... ./bundle/... .

lint:
	./lint.sh ./...

lintcheck:
	golangci-lint run ./...

test:
	gotestsum --format pkgname-and-test-fails --no-summary=skipped -- ${PACKAGES}

cover:
	gotestsum --format pkgname-and-test-fails --no-summary=skipped -- -coverprofile=coverage.txt ${PACKAGES}

showcover:
	go tool cover -html=coverage.txt

build: vendor
	go build -mod vendor

snapshot:
	go build -o .databricks/databricks

vendor:
	go mod vendor
  
schema:
	go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema.json

INTEGRATION = gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./integration/..." -- -parallel 4 -timeout=2h

integration:
	$(INTEGRATION)

integration-short:
	$(INTEGRATION) -short

.PHONY: lint lintcheck test cover showcover build snapshot vendor schema integration integration-short
