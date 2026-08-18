// Package dms records bundle deployment history with the deployment metadata service.
//
// One deploy or destroy is one version under one deployment. CreateVersion stages an
// operation for every resource the plan touches, in OPERATION_STATUS_PENDING at sequence
// id 0, and the CLI fills each one in with UpdateOperation as the resource is applied.
// The set is fixed at CreateVersion: the service has no call to add one later, and it caps
// how many a version may stage.
//
// An update is taken literally. The field mask decides what changes, and a field left out
// keeps the value it had. The service mirrors state and resource_id onto the
// deployment-level resource, which is what the next plan reads, so a mask that names state
// with no value is what removes a resource - and a failure that must not disturb the state
// an earlier write left names only error_message and status.
//
// Every update carries the sequence id the previous one returned, starting from the 0 that
// staging leaves, so a write from a stale deploy is rejected rather than applied.
//
// Two calls are written by hand rather than taken from the SDK, which is generated from an
// OpenAPI spec that trails the service; see client.go.
package dms
