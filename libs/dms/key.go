package dms

import "strings"

// statePrefix is what a bundle state key carries and a DMS resource key does not: state
// calls a job "resources.jobs.foo", DMS calls it "jobs.foo".
const statePrefix = "resources."

// ResourceKey is how DMS names one resource. Its own type is the point: a state key sent
// as a DMS key records the operation under a name nothing reads back.
type ResourceKey string

// KeyFromState converts a bundle state key to the key DMS knows the resource by.
func KeyFromState(stateKey string) ResourceKey {
	return ResourceKey(strings.TrimPrefix(stateKey, statePrefix))
}

// StateKey converts back to the bundle state key.
func (k ResourceKey) StateKey() string {
	return statePrefix + string(k)
}

func (k ResourceKey) String() string {
	return string(k)
}
