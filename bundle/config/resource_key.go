package config

import "strings"

// ResourceKey wraps a resource key (e.g. "resources.jobs.my_job" or
// "resources.jobs.my_job.permissions") for use as an error argument. It formats
// as the full key, so error messages are unchanged, but it reports only its
// resource type to telemetry — the resource name is user-authored and therefore
// PII, while the type is a value the CLI itself defines.
//
// Pass it wherever a resource key is interpolated into a safeerr error:
//
//	safeerr.Errorf("%s: SaveState: %w", config.ResourceKey(node), err)
//	  message:  resources.jobs.my_job: SaveState: ...
//	  template: jobs.*: SaveState: %w
type ResourceKey string

func (k ResourceKey) String() string {
	return string(k)
}

// SafeString implements safeerr.SafeStringer, standing in for the key with its
// name replaced by "*". A key this package cannot parse reports nothing beyond
// the redaction marker, since an unrecognized shape may be anything at all.
func (k ResourceKey) SafeString() string {
	resourceType := GetResourceTypeFromKey(string(k))
	if resourceType == "" {
		return "*"
	}

	// GetResourceTypeFromKey collapses a sub-resource into "<group>.<kind>"
	// (e.g. "jobs.permissions"), but in the key itself the kind trails the
	// name. Rebuild the key's own shape so the stand-in reads like the value.
	// The "resources." prefix is dropped: every key carries it, so it is noise.
	group, kind, hasKind := strings.Cut(resourceType, ".")
	if hasKind {
		return group + ".*." + kind
	}
	return group + ".*"
}
