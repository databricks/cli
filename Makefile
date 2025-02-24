default: vendor fmt lint tidy

PACKAGES=./acceptance/... ./libs/... ./internal/... ./cmd/... ./bundle/... .

GOTESTSUM_FORMAT ?= pkgname-and-test-fails
GOTESTSUM_CMD ?= gotestsum --format ${GOTESTSUM_FORMAT} --no-summary=skipped


lint:
	golangci-lint run --fix

tidy:
	@# not part of golangci-lint, apparently
	go mod tidy

lintcheck:
	golangci-lint run ./...

# Note 'make lint' will do formatting as well. However, if there are compilation errors,
# formatting/goimports will not be applied by 'make lint'. However, it will be applied by 'make fmt'.
# If you need to ensure that formatting & imports are always fixed, do "make fmt lint"
fmt:
	ruff format -q
	golangci-lint run --enable-only="gofmt,gofumpt,goimports" --fix ./...

test:
	${GOTESTSUM_CMD} -- ${PACKAGES}

cover:
	rm -fr ./acceptance/build/cover/
	VERBOSE_TEST=1 CLI_GOCOVERDIR=build/cover ${GOTESTSUM_CMD} -- -coverprofile=coverage.txt ${PACKAGES}
	rm -fr ./acceptance/build/cover-merged/
	mkdir -p acceptance/build/cover-merged/
	go tool covdata merge -i $$(printf '%s,' acceptance/build/cover/* | sed 's/,$$//') -o acceptance/build/cover-merged/
	go tool covdata textfmt -i acceptance/build/cover-merged -o coverage-acceptance.txt

showcover:
	go tool cover -html=coverage.txt

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

docs:
	go run ./bundle/docsgen ./bundle/internal/schema ./bundle/docsgen

INTEGRATION = gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./acceptance ./integration/..." -- -parallel 4 -timeout=2h

integration: vendor
	$(INTEGRATION)

integration-short: vendor
	VERBOSE_TEST=1 $(INTEGRATION) -short

.PHONY: lint tidy lintcheck fmt test cover showcover build snapshot vendor schema integration integration-short acc-cover acc-showcover docs
