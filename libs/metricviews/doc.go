// Package metricviews parses and serializes Databricks metric-view YAML
// definitions (versions 0.1, 1.0, and 1.1; single- and multi-source).
//
// It is a pure serialization/deserialization library: it models the YAML
// shapes accepted by the metric-view serde implementation in the backend, but
// performs no semantic validation (uniqueness, trailing-defaults, wildcard
// rules) — the backend validates those when a CREATE OR REPLACE statement runs.
//
// The exported types carry both yaml and json struct tags so callers can
// compare parsed values with libs/structs/structdiff for drift detection.
package metricviews
