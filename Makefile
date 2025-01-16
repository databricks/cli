default: vendor fmt lint

PACKAGES=./acceptance/... ./libs/... ./internal/... ./cmd/... ./bundle/... .

GOTESTSUM_FORMAT ?= pkgname-and-test-fails

lint:
	golangci-lint run --fix

lintcheck:
	golangci-lint run ./...

# Note 'make lint' will do formatting as well. However, if there are compilation errors,
# formatting/goimports will not be applied by 'make lint'. However, it will be applied by 'make fmt'.
# If you need to ensure that formatting & imports are always fixed, do "make fmt lint"
fmt:
	golangci-lint run --enable-only="gofmt,gofumpt,goimports" --fix ./...

test:
	gotestsum --format ${GOTESTSUM_FORMAT} --no-summary=skipped -- ${PACKAGES}

cover:
	gotestsum --format ${GOTESTSUM_FORMAT} --no-summary=skipped -- -coverprofile=coverage.txt ${PACKAGES}

showcover:
	go tool cover -html=coverage.txt

acc-cover:
	rm -fr ./acceptance/build/cover/
	CLI_GOCOVERDIR=build/cover go test ./acceptance
	rm -fr ./acceptance/build/cover-merged/
	mkdir -p acceptance/build/cover-merged/
	go tool covdata merge -i $$(printf '%s,' acceptance/build/cover/* | sed 's/,$$//') -o acceptance/build/cover-merged/
	go tool covdata textfmt -i acceptance/build/cover-merged -o coverage-acceptance.txt

acc-showcover:
	go tool cover -html=coverage-acceptance.txt

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

.PHONY: lint lintcheck fmt test cover showcover build snapshot vendor schema integration integration-short acc-cover acc-showcover
