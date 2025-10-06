Use this tool to generate equivalent Go types from Terraform provider schema.

## Usage

The entry point for this tool is `.`.

It uses `./tmp` a temporary data directory and `../schema` as output directory.

It automatically installs the Terraform binary as well as the Databricks Terraform provider.

Run with:

```go
go run .
```

How to regenerate Go structs from an updated terraform provider?
1. Bump version in ./schema/version.go
2. Delete `./tmp` if it exists
3. Run `go run .`
4. Run `gofmt -s -w ../schema`
