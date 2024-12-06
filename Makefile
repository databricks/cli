default: build

fmt:
	@echo "✓ Formatting source code with goimports ..."
	@goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
	@echo "✓ Formatting source code with gofmt ..."
	@gofmt -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

lint: vendor
	@echo "✓ Linting source code with https://golangci-lint.run/ ..."
	@golangci-lint run ./...

lintfix: vendor
	@echo "✓ Linting source code with 'golangci-lint run --fix' ..."
	@golangci-lint run --fix ./...

test: lint testonly

testonly:
	@echo "✓ Running tests ..."
	@gotestsum --format pkgname-and-test-fails --no-summary=skipped --raw-command go test -v -json -short -coverprofile=coverage.txt ./...

viewchanges: changecalc/changecalc
	@changecalc/changecalc

testchanges: changecalc/changecalc
	@echo "✓ Running tests based on changes relative to main..."
	changecalc/changecalc > changed-packages.txt || echo "./..." > changed-packages.txt
	gotestsum --format pkgname-and-test-fails --no-summary=skipped --raw-command go test -v -json -short -coverprofile=coverage.txt $(shell cat changed-packages.txt)

changecalc/changecalc: changecalc/*.go
	@go build -o changecalc/changecalc changecalc/main.go

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


.PHONY: build vendor coverage test lint fmt viewchanges testchanges
