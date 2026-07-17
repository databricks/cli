package localenv

import (
	"context"
	"fmt"
	"strings"
)

// ComputeClient is a narrow seam over the SDK so tests can stub it.
type ComputeClient interface {
	// GetClusterSparkVersion returns the Spark version string for a cluster.
	GetClusterSparkVersion(ctx context.Context, clusterID string) (string, error)
	// GetClusterByName resolves a cluster name to its ID and Spark version. It
	// errors when the name is unknown or ambiguous (more than one cluster shares
	// the name), so the caller can surface an actionable E_RESOLVE.
	GetClusterByName(ctx context.Context, name string) (clusterID, sparkVersion string, err error)
	// GetJobSparkVersion returns either a Spark version (isServerless=false) or a
	// serverless marker (isServerless=true) for a job, plus a recorded version string.
	GetJobSparkVersion(ctx context.Context, jobID string) (sparkVersion string, isServerless bool, version string, err error)
}

// TargetFlags holds the mutually-exclusive compute target flags from the CLI.
type TargetFlags struct {
	Cluster     string
	ClusterName string
	Serverless  string
	Job         string
}

// BundleTarget is the three-state result of reading the bundle's configured
// target. Selected=false means nothing was configured.
type BundleTarget struct {
	ClusterID  string
	Serverless bool
	Selected   bool
}

// noTargetMessage is the actionable message shown when no target is selected,
// matching spec §2.3.
const noTargetMessage = "No compute target is selected. Select a cluster or serverless target, or pass --cluster-id / --cluster-name / --serverless-version / --job-id"

// ValidateTargetFlags returns an error if more than one of the target flags is set.
// Cobra marks them mutually exclusive too; this guards the library path.
func ValidateTargetFlags(f TargetFlags) error {
	var set []string
	if f.Cluster != "" {
		set = append(set, "--cluster-id")
	}
	if f.ClusterName != "" {
		set = append(set, "--cluster-name")
	}
	if f.Serverless != "" {
		set = append(set, "--serverless-version")
	}
	if f.Job != "" {
		set = append(set, "--job-id")
	}
	if len(set) > 1 {
		return fmt.Errorf("flags %s are mutually exclusive; specify at most one", strings.Join(set, " and "))
	}
	return nil
}

// ResolveTarget resolves the compute target using ordered precedence:
// --cluster-id → --cluster-name → --serverless-version → --job-id → bundle target.
// PythonVersion is left empty; it is filled later from constraint data.
//
// Incompatible flags are rejected up front: without this a library caller that
// bypasses Cobra (which also marks the flags mutually exclusive) and passes more
// than one target flag would have all but the first precedence branch silently
// ignored, resolving a different target than requested.
func ResolveTarget(ctx context.Context, f TargetFlags, c ComputeClient, bt BundleTarget) (*TargetInfo, error) {
	if err := ValidateTargetFlags(f); err != nil {
		return nil, NewError(ErrResolve, err, "invalid compute target flags")
	}

	if f.Cluster != "" {
		v, err := c.GetClusterSparkVersion(ctx, f.Cluster)
		if err != nil {
			return nil, NewError(ErrResolve, err, "resolving cluster %s", f.Cluster)
		}
		return &TargetInfo{
			Source:       "cluster",
			ClusterID:    f.Cluster,
			SparkVersion: v,
			EnvKey:       EnvKeyForSparkVersion(v),
		}, nil
	}

	if f.ClusterName != "" {
		// Resolve the name to an ID via the Clusters API; from there it is
		// identical to --cluster-id. An unknown or ambiguous name (two clusters
		// sharing it) yields an actionable E_RESOLVE.
		id, v, err := c.GetClusterByName(ctx, f.ClusterName)
		if err != nil {
			return nil, NewError(ErrResolve, err, "resolving cluster name %q", f.ClusterName)
		}
		return &TargetInfo{
			Source:       "cluster",
			ClusterID:    id,
			SparkVersion: v,
			EnvKey:       EnvKeyForSparkVersion(v),
		}, nil
	}

	if f.Serverless != "" {
		return &TargetInfo{
			Source:            "serverless",
			ServerlessVersion: NormalizeServerless(f.Serverless),
			EnvKey:            EnvKeyForServerless(f.Serverless),
		}, nil
	}

	if f.Job != "" {
		sparkVersion, isServerless, version, err := c.GetJobSparkVersion(ctx, f.Job)
		if err != nil {
			return nil, NewError(ErrResolve, err, "resolving job %s", f.Job)
		}
		if isServerless {
			// Use the job's recorded serverless environment version when present;
			// fall back to the default when the job did not pin one (documented
			// stand-in, spec §4.3).
			v := version
			if v == "" {
				v = defaultServerlessVersion
			}
			return &TargetInfo{
				Source:            "job",
				ServerlessVersion: NormalizeServerless(v),
				EnvKey:            EnvKeyForServerless(v),
			}, nil
		}
		// Classic compute: the Spark version is the first return per the
		// GetJobSparkVersion contract, not the recorded-version third return.
		return &TargetInfo{
			Source:       "job",
			SparkVersion: sparkVersion,
			EnvKey:       EnvKeyForSparkVersion(sparkVersion),
		}, nil
	}

	// Fall back to bundle target.
	if !bt.Selected {
		return nil, NewError(ErrNoTarget, nil, "%s", noTargetMessage)
	}

	if bt.Serverless {
		// The bundle records that the target is serverless but not which version,
		// so use the default stand-in (spec §4.3).
		return &TargetInfo{
			Source:            "bundle",
			ServerlessVersion: NormalizeServerless(defaultServerlessVersion),
			EnvKey:            EnvKeyForServerless(defaultServerlessVersion),
		}, nil
	}

	if bt.ClusterID != "" {
		v, err := c.GetClusterSparkVersion(ctx, bt.ClusterID)
		if err != nil {
			return nil, NewError(ErrResolve, err, "resolving bundle cluster %s", bt.ClusterID)
		}
		return &TargetInfo{
			Source:       "bundle",
			ClusterID:    bt.ClusterID,
			SparkVersion: v,
			EnvKey:       EnvKeyForSparkVersion(v),
		}, nil
	}

	// Bundle target is selected but has neither serverless nor a cluster ID —
	// treat this the same as nothing selected so the user gets a clear message.
	return nil, NewError(ErrNoTarget, nil, "%s", noTargetMessage)
}
