default: build

fmt: $(GOPATH)/bin/goimports
	@echo "✓ Formatting source code with goimports ..."
	@goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
	@echo "✓ Formatting source code with gofmt ..."
	@gofmt -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

$(GOPATH)/bin/goimports:
	go install golang.org/x/tools/cmd/goimports@latest

lint: vendor $(GOPATH)/bin/staticcheck
	@echo "✓ Linting source code with https://staticcheck.io/ ..."
	@staticcheck ./...

$(GOPATH)/bin/staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

test: lint $(GOPATH)/bin/gotestsum
	@echo "✓ Running tests ..."
	@gotestsum --format pkgname-and-test-fails --no-summary=skipped --raw-command go test -v -json -short -coverprofile=coverage.txt ./...

$(GOPATH)/bin/gotestsum:
	go install gotest.tools/gotestsum@latest

coverage: test
	@echo "✓ Opening coverage for unit tests ..."
	@go tool cover -html=coverage.txt

build: vendor
	@echo "✓ Building source code with go build ..."
	@go build -mod vendor

snapshot: $(GOPATH)/bin/goreleaser
	@echo "✓ Building dev snapshot"
	@goreleaser build --snapshot --clean --single-target

$(GOPATH)/bin/goreleaser:
	go install github.com/goreleaser/goreleaser@latest

vendor:
	@echo "✓ Filling vendor folder with library code ..."
	@go mod vendor

.PHONY: build vendor coverage test lint fmt
