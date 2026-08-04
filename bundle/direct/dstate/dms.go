package dstate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// RecordedState is what the CLI serializes into the DMS Operation.State field.
//
// It is an envelope rather than the bare resource config, because depends_on has
// to survive the round trip: DMS has no field for dependency edges, and they
// cannot be recomputed from the config once it is recorded (references are
// resolved to literals before serialization). Nesting depends_on inside the
// config instead would collide with resource fields of the same name, e.g.
// jobs.Task.depends_on.
//
// The shape deliberately matches the local ResourceEntry so both sides of the
// state round trip look the same.
type RecordedState struct {
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

// readDMSState replaces the file-derived resource state with the state recorded
// in DMS. Recording is only enabled for net-new deployments, so once a
// deployment exists DMS owns its resource set outright - including when that set
// is empty, which is a successful deploy of nothing rather than missing data.
// The caller holds db.mu.
func (db *DeploymentState) readDMSState(ctx context.Context, src *DMSSource) error {
	resources, err := fetchDeploymentResources(ctx, src.Client, src.DeploymentID)
	if err != nil {
		// The deployment's record is created by its first version, so a node can
		// resolve to an ID that has none yet: a deploy that registered the deployment
		// and then failed before recording a version. There is nothing to read, and
		// the file's resources are still empty, so carry on and let this deploy record
		// the first version.
		if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
			log.Debugf(ctx, "No deployment record for %s yet; keeping local state", src.DeploymentID)
			return nil
		}
		return err
	}

	db.Data.State = resources
	db.stateIDs = make(map[string]string, len(resources))
	for key, entry := range resources {
		db.stateIDs[key] = entry.ID
	}
	return nil
}

// fetchDeploymentResources lists every resource recorded for the deployment in
// DMS and maps them into state entries keyed by the fully-qualified resource key.
func fetchDeploymentResources(ctx context.Context, client bundledeployments.BundleDeploymentsInterface, deploymentID string) (map[string]ResourceEntry, error) {
	it := client.ListResources(ctx, bundledeployments.ListResourcesRequest{
		Parent: "deployments/" + deploymentID,
	})

	out := make(map[string]ResourceEntry)
	for it.HasNext(ctx) {
		res, err := it.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing resources from deployment metadata service: %w", err)
		}

		// DMS reports resource keys without the "resources." prefix (e.g.
		// "jobs.foo"), but the state DB keys are fully qualified
		// ("resources.jobs.foo"), so prepend it here.
		key := "resources." + res.ResourceKey

		var recorded RecordedState
		if res.State != "" {
			if err := json.Unmarshal([]byte(res.State), &recorded); err != nil {
				return nil, fmt.Errorf("interpreting state recorded for %s: %w", key, err)
			}
		}

		// Normalize integral doubles ("1.0") back to integers before the state
		// reaches the typed resource structs, which unmarshal fields like
		// jobs.JobSettings.MaxConcurrentRuns and num_workers as int and reject
		// the fractional form. DMS now stores state as an opaque string it never
		// parses (see the Operation.state SDK docs), so freshly recorded state no
		// longer picks up doubles; this still guards state written by an older
		// server that round-tripped it through a protobuf Struct.
		state, err := normalizeIntegralNumbers(recorded.State)
		if err != nil {
			return nil, fmt.Errorf("interpreting state recorded for %s: %w", key, err)
		}

		out[key] = ResourceEntry{
			ID:        res.ResourceId,
			State:     state,
			DependsOn: recorded.DependsOn,
		}
	}
	return out, nil
}

// normalizeIntegralNumbers rewrites JSON numbers that have no fractional part
// (e.g. "1.0") as integers ("1"). DMS round-trips state through a protobuf
// Struct whose only numeric type is double, so every integer it stores comes
// back fractional; the typed resource structs unmarshal integer fields as int
// and reject that form. Genuinely fractional numbers are left untouched.
//
// A nil or empty input is returned unchanged so an unrecorded resource keeps
// its nil state rather than becoming "null".
func normalizeIntegralNumbers(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}

	return json.Marshal(normalizeValue(v))
}

// normalizeValue walks a decoded JSON value (with numbers as json.Number) and
// converts every integral number to an int64, recursing into objects and
// arrays. Non-numeric leaves are returned as-is.
func normalizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			t[k] = normalizeValue(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = normalizeValue(val)
		}
		return t
	case json.Number:
		// An integer already parses as int64; keep it. Otherwise the value is a
		// double, and only its integral form needs rewriting - a real fraction
		// must survive untouched (e.g. a float-typed config field).
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil && f == math.Trunc(f) && !math.IsInf(f, 0) {
			return int64(f)
		}
		return t
	default:
		return v
	}
}
