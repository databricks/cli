// Package fuzz provides randomized generators and harnesses that compare how the
// terraform and direct deploy engines translate the same bundle resource into an
// API create payload. See DECO-25361.
//
// The first technique implemented here generates a random resource config and
// checks for differences in the create payload between the terraform and direct
// engines. Generators are seeded so that any divergence found by the fuzz driver
// can be reproduced from the printed seed.
//
// Only jobs are covered for now. Extending the harness to other resource kinds
// (pipelines, apps, ...) is tracked as follow-up work under DECO-25361.
//
// Everything else in the package lives in _test.go files: the package is a
// test-only utility and nothing in the product imports it, so keeping the logic
// out of the regular build avoids shipping dead code. This file exists only to
// carry the package documentation in a non-test file.
package fuzz
