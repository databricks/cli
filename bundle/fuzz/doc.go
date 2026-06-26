// Package fuzz compares how the terraform and direct deploy engines translate the
// same bundle resource into an API create payload, catching divergences during the
// migration off terraform. Generators are seeded so any divergence reproduces from
// the printed seed. Jobs only for now (DECO-25361).
//
// Everything lives in _test.go files: the package is test-only and nothing in the
// product imports it. This file exists only to carry the package doc.
package fuzz
