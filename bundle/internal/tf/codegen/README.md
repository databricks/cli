Use this tool to generate equivalent Go types from Terraform provider schema.

## Usage

The entry point for this tool is `.`.

It uses `./tmp` a temporary data directory and `../schema` as output directory.

It automatically installs the Terraform binary as well as the Databricks Terraform provider.

Run with:

```go
go run .
```
