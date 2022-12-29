default: build

fmt:
	@echo "✓ Formatting source code with goimports ..."
	@goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
	@echo "✓ Formatting source code with gofmt ..."
	@gofmt -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

lint: vendor
	@echo "✓ Linting source code with https://staticcheck.io/ ..."
	@staticcheck ./...

test: lint
	@echo "✓ Running tests ..."
	@gotestsum --format pkgname-and-test-fails --no-summary=skipped --raw-command go test -v -json -short -coverprofile=coverage.txt ./...

coverage: test
	@echo "✓ Opening coverage for unit tests ..."
	@go tool cover -html=coverage.txt

build: vendor
	@echo "✓ Building source code with go build ..."
	@go build -mod vendor

snapshot:
	# ln -fs $PWD/dist/bricks_darwin_arm64/bricks_v0.0.15-devel ~/bin/bricks
	@echo "✓ Building dev snapshot"
	@goreleaser build --snapshot --rm-dist --single-target

vendor:
	@echo "✓ Filling vendor folder with library code ..."
	@go mod vendor

.PHONY: build vendor coverage test lint fmt