default: checks fmt lint

PACKAGES=./acceptance/... ./libs/... ./internal/... ./cmd/... ./bundle/... ./experimental/ssh/... .

GO_TOOL ?= go tool -modfile=tools/go.mod
GOTESTSUM_FORMAT ?= pkgname-and-test-fails
GOTESTSUM_CMD ?= ${GO_TOOL} gotestsum --format ${GOTESTSUM_FORMAT} --no-summary=skipped --jsonfile test-output.json
LOCAL_TIMEOUT ?= 30m


lintfull: ./tools/golangci-lint
	./tools/golangci-lint run --fix

lint: ./tools/golangci-lint
	./tools/lintdiff.py ./tools/golangci-lint run --fix

tidy:
	@# not part of golangci-lint, apparently
	go mod tidy

lintcheck: ./tools/golangci-lint
	./tools/golangci-lint run ./...

fmtfull: ./tools/golangci-lint ./tools/yamlfmt
	ruff format -n
	./tools/golangci-lint fmt
	./tools/yamlfmt .

fmt: ./tools/golangci-lint ./tools/yamlfmt
	ruff format -n
	./tools/lintdiff.py ./tools/golangci-lint fmt
	./tools/yamlfmt .

# pre-building yamlfmt because it is invoked from tests and scripts
tools/yamlfmt: tools/go.mod tools/go.sum
	go build -modfile=tools/go.mod -o tools/yamlfmt github.com/google/yamlfmt/cmd/yamlfmt

tools/yamlfmt.exe: tools/go.mod tools/go.sum
	go build -modfile=tools/go.mod -o tools/yamlfmt.exe github.com/google/yamlfmt/cmd/yamlfmt

# pre-building golangci-lint because it's faster to run pre-built version
tools/golangci-lint: tools/go.mod tools/go.sum
	go build -modfile=tools/go.mod -o tools/golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint

ws:
	./tools/validate_whitespace.py

links:
	./tools/update_github_links.py

# Checks other than 'fmt' and 'lint'; these are fast, so can be run first
checks: tidy ws links

test:
	${GOTESTSUM_CMD} -- ${PACKAGES} -timeout=${LOCAL_TIMEOUT} -short

test-slow:
	${GOTESTSUM_CMD} -- ${PACKAGES} -timeout=${LOCAL_TIMEOUT}

# Updates acceptance test output (local tests)
test-update:
	-go test ./acceptance -run '^TestAccept$$' -update -timeout=${LOCAL_TIMEOUT}
	@# at the moment second pass is required because some tests show diff against output of another test for easier review
	-go test ./acceptance -run '^TestAccept$$' -update -timeout=${LOCAL_TIMEOUT}

# Updates acceptance test output (integration tests, requires access)
test-update-aws:
	deco env run -i -n aws-prod-ucws -- go test ./acceptance -run ^TestAccept$$ -update -timeout=1h -skiplocal -v

test-update-all: test-update test-update-aws

slowest:
	${GO_TOOL} gotestsum tool slowest --jsonfile test-output.json --threshold 1s --num 50

cover:
	rm -fr ./acceptance/build/cover/
	VERBOSE_TEST=1 CLI_GOCOVERDIR=build/cover ${GOTESTSUM_CMD} -- -coverprofile=coverage.txt ${PACKAGES} -timeout=${LOCAL_TIMEOUT}
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

# builds the binary in a VM environment (such as Parallels Desktop) where your files are mirrored from the host os
build-vm: tidy
	go build -buildvcs=false

snapshot:
	go build -o .databricks/databricks

# Produce release binaries and archives in the dist folder without uploading them anywhere.
# Useful for "databricks ssh" development, as it needs to upload linux releases to the /Workspace.
snapshot-release:
	goreleaser release --clean --skip docker --snapshot

schema:
	go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema.json

docs:
	go run ./bundle/docsgen ./bundle/internal/schema ./bundle/docsgen

INTEGRATION = ${GO_TOOL} gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./acceptance ./integration/..." -- -parallel 4 -timeout=2h

integration:
	$(INTEGRATION)

integration-short:
	VERBOSE_TEST=1 $(INTEGRATION) -short

generate-validation:
	go run ./bundle/internal/validation/.
	gofmt -w -s ./bundle/internal/validation/generated

# Rule to generate the CLI from a new version of the OpenAPI spec.
# I recommend running this rule from Arca because of faster build times
# because of better caching and beefier machines, but it should also work
# fine from your local mac.
#
# By default, this rule will use the universe directory in your home
# directory. You can override this by setting the UNIVERSE_DIR
# environment variable.
#
# Example:
# UNIVERSE_DIR=/Users/shreyas.goenka/universe make generate
UNIVERSE_DIR ?= $(HOME)/universe
GENKIT_BINARY := $(UNIVERSE_DIR)/bazel-bin/openapi/genkit/genkit_/genkit

generate:
	@echo "Checking out universe at SHA: $$(cat .codegen/_openapi_sha)"
	cd $(UNIVERSE_DIR) && git fetch origin master && git checkout $$(cat $(PWD)/.codegen/_openapi_sha)
	@echo "Building genkit..."
	cd $(UNIVERSE_DIR) && bazel build //openapi/genkit
	@echo "Generating CLI code..."
	$(GENKIT_BINARY) update-sdk


.PHONY: lint lintfull tidy lintcheck fmt fmtfull test cover showcover build snapshot snapshot-release schema integration integration-short acc-cover acc-showcover docs ws links checks test-update test-update-aws test-update-all generate-validation
