# metricviews

Go serialization/deserialization for
[Databricks metric-view](https://docs.databricks.com/aws/en/metric-views/)
YAML definitions (versions 0.1, 1.0, and 1.1; single- and multi-source). It mirrors the YAML
shapes accepted by the metric-view serde implementation in the backend.

## Scope

Pure serde. No semantic validation (such as uniqueness, trailing defaults, or
wildcard rules); the backend validates these when a
`CREATE OR REPLACE VIEW ... LANGUAGE YAML` statement runs. Models the YAML
shape only. Exported types carry `yaml` and `json` tags so callers can diff
parsed values with `libs/structs/structdiff`.

## Fixtures

`testdata/fixtures/*.yml` exercise every feature of the format. They use
generic placeholder identifiers (e.g. `main.sales.orders`); only the YAML
structure is significant.

## Conformance

`conformance_test.go` asserts every fixture parses and that re-serialization is
idempotent (`marshal(parse(x))` is a fixed point). The package tracks a pinned
upstream revision of the serde format; that provenance is maintained privately,
outside this repository.
