default: tidy fmt lint ws

PACKAGES=./acceptance/... ./libs/... ./internal/... ./cmd/... ./bundle/... .

GOTESTSUM_FORMAT ?= pkgname-and-test-fails
GOTESTSUM_CMD ?= go tool gotestsum --format ${GOTESTSUM_FORMAT} --no-summary=skipped --jsonfile test-output.json


lint:
	golangci-lint run --fix

tidy:
	@# not part of golangci-lint, apparently
	go mod tidy

lintcheck:
	golangci-lint run ./...

fmt:
	ruff format -qn
	golangci-lint fmt

ws:
	./tools/validate_whitespace.py

test:
	${GOTESTSUM_CMD} -- ${PACKAGES}

slowest:
	go tool gotestsum tool slowest --jsonfile test-output.json --threshold 1s --num 50

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

build: tidy
	go build

snapshot:
	go build -o .databricks/databricks

schema:
	go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema.json

docs:
	go run ./bundle/docsgen ./bundle/internal/schema ./bundle/docsgen

INTEGRATION = go tool gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./acceptance ./integration/..." -- -parallel 4 -timeout=2h

integration:
	$(INTEGRATION)

integration-short:
	VERBOSE_TEST=1 $(INTEGRATION) -short

generate:
	genkit update-sdk
	[ ! -f tagging.py ] || mv tagging.py internal/genkit/tagging.py
# tagging.yml is automatically synced by update-sdk command and contains a reference to tagging.py in root
# since we move tagging.py to different folder, we need to update this reference here as well
	[ ! -f .github/workflows/tagging.yml ] || sed -i '' 's/python tagging.py/python internal\/genkit\/tagging.py/g' .github/workflows/tagging.yml
	[ ! -f .github/workflows/next-changelog.yml ] || rm .github/workflows/next-changelog.yml
	pushd experimental/python && make codegen

.PHONY: lint tidy lintcheck fmt test cover showcover build snapshot schema integration integration-short acc-cover acc-showcover docs ws
