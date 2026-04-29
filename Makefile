.PHONY: default
default: checks fmt lint

# Default packages to test (all)
TEST_PACKAGES = ./acceptance/internal ./libs/... ./internal/... ./cmd/... ./bundle/... ./experimental/ssh/... .

# Default acceptance test filter (all)
ACCEPTANCE_TEST_FILTER = ""

GO_TOOL ?= go tool -modfile=tools/go.mod
GOTESTSUM_FORMAT ?= pkgname-and-test-fails
GOTESTSUM_CMD ?= ${GO_TOOL} gotestsum --format ${GOTESTSUM_FORMAT} --no-summary=skipped --jsonfile test-output.json --rerun-fails
LOCAL_TIMEOUT ?= 30m


.PHONY: lintfull
lintfull: ./tools/golangci-lint
	./tools/golangci-lint run --fix

.PHONY: lint
lint: ./tools/golangci-lint
	./tools/lintdiff.py ./tools/golangci-lint run --fix

.PHONY: tidy
tidy:
	@# not part of golangci-lint, apparently
	go mod tidy

.PHONY: lintcheck
lintcheck: ./tools/golangci-lint
	./tools/golangci-lint run ./...

.PHONY: fmtfull
fmtfull: ./tools/golangci-lint ./tools/yamlfmt
	ruff format -n
	./tools/golangci-lint fmt
	./tools/yamlfmt .

.PHONY: fmt
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

.PHONY: ws
ws:
	./tools/validate_whitespace.py

.PHONY: wsfix
wsfix:
	./tools/validate_whitespace.py --fix

.PHONY: links
links:
	./tools/update_github_links.py

.PHONY: deadcode
deadcode:
	./tools/check_deadcode.py

# Checks other than 'fmt' and 'lint'; these are fast, so can be run first
.PHONY: checks
checks: tidy ws links deadcode


.PHONY: install-pythons
install-pythons:
	uv python install 3.9 3.10 3.11 3.12 3.13

.PHONY: test
test: test-unit test-acc

.PHONY: test-unit
test-unit:
	${GOTESTSUM_CMD} --packages "${TEST_PACKAGES}" -- -timeout=${LOCAL_TIMEOUT}

.PHONY: test-acc
test-acc:
	${GOTESTSUM_CMD} --packages ./acceptance/... -- -timeout=${LOCAL_TIMEOUT} -run ${ACCEPTANCE_TEST_FILTER}

# Updates acceptance test output (local tests)
.PHONY: test-update
test-update:
	-go test ./acceptance -run '^TestAccept$$' -update -timeout=${LOCAL_TIMEOUT}

# Updates acceptance test output for template tests only
.PHONY: test-update-templates
test-update-templates:
	-go test ./acceptance -run '^TestAccept/bundle/templates' -update -timeout=${LOCAL_TIMEOUT}

# Regenerate out.test.toml files without running tests
.PHONY: generate-out-test-toml
generate-out-test-toml:
	go test ./acceptance -run '^TestAccept$$' -only-out-test-toml -timeout=${LOCAL_TIMEOUT}

# Updates acceptance test output (integration tests, requires access)
.PHONY: test-update-aws
test-update-aws:
	deco env run -i -n aws-prod-ucws -- env DATABRICKS_TEST_SKIPLOCAL=1 go test ./acceptance -run ^TestAccept$$ -update -timeout=1h -v

.PHONY: test-update-all
test-update-all: test-update test-update-aws

.PHONY: slowest
slowest:
	${GO_TOOL} gotestsum tool slowest --jsonfile test-output.json --threshold 1s --num 50

.PHONY: cover
cover:
	rm -fr ./acceptance/build/cover/
	VERBOSE_TEST=1 ${GOTESTSUM_CMD} --packages "${TEST_PACKAGES}" -- -coverprofile=coverage.txt -timeout=${LOCAL_TIMEOUT}
	VERBOSE_TEST=1 CLI_GOCOVERDIR=build/cover ${GOTESTSUM_CMD} --packages ./acceptance/... -- -timeout=${LOCAL_TIMEOUT} -run ${ACCEPTANCE_TEST_FILTER}
	rm -fr ./acceptance/build/cover-merged/
	mkdir -p acceptance/build/cover-merged/
	go tool covdata merge -i $$(printf '%s,' acceptance/build/cover/* | sed 's/,$$//') -o acceptance/build/cover-merged/
	go tool covdata textfmt -i acceptance/build/cover-merged -o coverage-acceptance.txt

.PHONY: showcover
showcover:
	go tool cover -html=coverage.txt

.PHONY: acc-showcover
acc-showcover:
	go tool cover -html=coverage-acceptance.txt

.PHONY: fetch-compat-manifest
fetch-compat-manifest:
	@curl -sfL https://raw.githubusercontent.com/databricks/appkit/main/cli-compat.json \
		-o libs/apps/compat/cli-compat.json \
		&& echo "Fetched latest cli-compat.json" \
		|| echo "Warning: failed to fetch cli-compat.json, using existing copy"

.PHONY: build
build: tidy
	go build

# builds the binary in a VM environment (such as Parallels Desktop) where your files are mirrored from the host os
.PHONY: build-vm
build-vm: tidy
	go build -buildvcs=false

.PHONY: snapshot
snapshot:
	go build -o .databricks/databricks

# Produce release binaries and archives in the dist folder without uploading them anywhere.
# Useful for "databricks ssh" development, as it needs to upload linux releases to the /Workspace.
.PHONY: snapshot-release
snapshot-release:
	goreleaser release --clean --skip docker --snapshot

.PHONY: schema
schema:
	go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema.json

.PHONY: schema-for-docs
schema-for-docs:
	go run ./bundle/internal/schema ./bundle/internal/schema ./bundle/schema/jsonschema_for_docs.json --docs

.PHONY: docs
docs:
	go run ./bundle/docsgen ./bundle/internal/schema ./bundle/docsgen

INTEGRATION = go run -modfile=tools/go.mod ./tools/testrunner/main.go ${GO_TOOL} gotestsum --format github-actions --rerun-fails --jsonfile output.json --packages "./acceptance ./integration/..." -- -parallel 4 -timeout=2h

.PHONY: integration
integration: install-pythons
	$(INTEGRATION)

.PHONY: integration-short
integration-short: install-pythons
	DATABRICKS_TEST_SKIPLOCAL=1 VERBOSE_TEST=1 $(INTEGRATION) -short

.PHONY: dbr-integration
dbr-integration: install-pythons
	DBR_ENABLED=true go test -v -timeout 4h -run TestDbrAcceptance$$ ./acceptance

# DBR acceptance tests - run on Databricks Runtime using serverless compute
# These require deco env run for authentication
# Set DBR_TEST_VERBOSE=1 for detailed output (e.g., DBR_TEST_VERBOSE=1 make dbr-test)
.PHONY: dbr-test
dbr-test:
	deco env run -i -n aws-prod-ucws -- make dbr-integration

.PHONY: generate-validation
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

.PHONY: generate
generate:
	@if [ -z "$$UNIVERSE_SKIP_CHECKOUT" ]; then \
		if ! git -C $(UNIVERSE_DIR) diff --quiet || ! git -C $(UNIVERSE_DIR) diff --cached --quiet; then \
			echo "Error: universe repo at $(UNIVERSE_DIR) has uncommitted changes; commit or stash them, or set UNIVERSE_SKIP_CHECKOUT=1 to skip checkout"; \
			exit 1; \
		fi; \
		echo "Checking out universe at SHA: $$(cat .codegen/_openapi_sha)"; \
		cd $(UNIVERSE_DIR) && (git cat-file -e $$(cat $(PWD)/.codegen/_openapi_sha) 2>/dev/null || (git fetch --filter=blob:none origin master && git checkout $$(cat $(PWD)/.codegen/_openapi_sha))); \
	else \
		echo "UNIVERSE_SKIP_CHECKOUT set; using current $(UNIVERSE_DIR) HEAD"; \
	fi
	@echo "Building genkit..."
	cd $(UNIVERSE_DIR) && bazel build //openapi/genkit
	@echo "Generating CLI code..."
	$(GENKIT_BINARY) update-sdk
	cat .gitattributes.manual .gitattributes > .gitattributes.tmp && mv .gitattributes.tmp .gitattributes
	-go test ./acceptance -run TestAccept/bundle/refschema -update &> /dev/null
	@echo "Updating direct engine config..."
	make generate-direct
	go test ./bundle/internal/schema

.codegen/openapi.json: .codegen/_openapi_sha
	wget -O $@.tmp "https://openapi.dev.databricks.com/$$(cat $<)/specs/all-internal.json" && mv $@.tmp $@ && touch $@

.PHONY: generate-direct
generate-direct: generate-direct-apitypes generate-direct-resources

.PHONY: generate-direct-apitypes
generate-direct-apitypes: bundle/direct/dresources/apitypes.generated.yml

.PHONY: generate-direct-resources
generate-direct-resources: bundle/direct/dresources/resources.generated.yml

.PHONY: generate-direct-clean
generate-direct-clean:
	rm -f bundle/direct/dresources/apitypes.generated.yml bundle/direct/dresources/resources.generated.yml

bundle/direct/dresources/apitypes.generated.yml: ./bundle/direct/tools/generate_apitypes.py .codegen/openapi.json acceptance/bundle/refschema/out.fields.txt
	python3 $^ > $@

bundle/direct/dresources/resources.generated.yml: ./bundle/direct/tools/generate_resources.py .codegen/openapi.json bundle/direct/dresources/apitypes.generated.yml bundle/direct/dresources/apitypes.yml acceptance/bundle/refschema/out.fields.txt
	python3 $^ > $@

.PHONY: test-exp-aitools
test-exp-aitools:
	make test TEST_PACKAGES="./experimental/aitools/..." ACCEPTANCE_TEST_FILTER="TestAccept/apps"

.PHONY: test-exp-ssh
test-exp-ssh:
	make test TEST_PACKAGES="./experimental/ssh/..." ACCEPTANCE_TEST_FILTER="TestAccept/ssh"

.PHONY: test-pipelines
test-pipelines:
	make test TEST_PACKAGES="./cmd/pipelines/..." ACCEPTANCE_TEST_FILTER="TestAccept/pipelines"


# Benchmarks:

.PHONY: bench1k
bench1k:
	BENCHMARK_PARAMS="--jobs 1000" go test ./acceptance -v -tail -run TestAccept/bundle/benchmarks -timeout=120m

.PHONY: bench100
bench100:
	BENCHMARK_PARAMS="--jobs 100" go test ./acceptance -v -tail -run TestAccept/bundle/benchmarks -timeout=120m

# small benchmark to quickly test benchmark-related code
.PHONY: bench10
bench10:
	BENCHMARK_PARAMS="--jobs 10" go test ./acceptance -v -tail -run TestAccept/bundle/benchmarks -timeout=120m

bench1k.log:
	make bench1k | tee $@

bench100.log:
	make bench100 | tee $@

bench10.log:
	make bench10 | tee $@

.PHONY: bench1k_summary
bench1k_summary: bench1k.log
	./tools/bench_parse.py $<

.PHONY: bench100_summary
bench100_summary: bench100.log
	./tools/bench_parse.py $<

.PHONY: bench10_summary
bench10_summary: bench10.log
	./tools/bench_parse.py $<
