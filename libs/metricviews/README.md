# metricviews

Go serialization/deserialization for YAML definitions of [Databricks metric views](https://docs.databricks.com/aws/en/metric-views/) (versions 0.1, 1.0, and 1.1; single- and multi-source).
It mirrors the YAML shapes accepted by the backend.

## Scope

`Parse` decodes YAML without applying structural rules, so persisted definitions remain readable when those rules change. Use `ParseAndValidate` for user input, or call `Validate` separately when only the structural checks are needed. Semantic validation (such as uniqueness, trailing defaults, or wildcard rules) is outside this package. Exported types carry `yaml` and `json` tags so callers can diff parsed values with `libs/structs/structdiff`.

## Fixtures

`testdata/fixtures/*.yml` exercise every feature of the format. They use
generic placeholder identifiers (e.g. `main.sales.orders`); only the YAML
structure is significant.

## Conformance

`conformance_test.go` asserts every fixture validates and parses, and that re-serialization is idempotent (`marshal(parse(x))` is a fixed point). The package tracks a pinned upstream revision of the serde format; that provenance is maintained privately, outside this repository.
