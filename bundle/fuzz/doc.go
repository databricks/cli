// Package fuzz deploys randomly generated bundle resources through the direct
// engine and asserts invariants that any valid config's API create payload must
// satisfy (e.g. task keys are preserved, references resolve, a new_cluster is
// sized by autoscale or num_workers but not both). Unlike a terraform/direct
// payload comparison, an invariant has no legitimate reason to fail, so a failure
// is a real bug. Generators are seeded so any failure reproduces from the printed
// seed. Jobs only for now.
//
// Everything lives in _test.go files: the package is test-only and nothing in the
// product imports it. This file exists only to carry the package doc.
package fuzz
