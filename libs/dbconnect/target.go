package dbconnect

import (
	"context"
	"fmt"
	"strings"
)

// ComputeClient is a narrow seam over the SDK so tests can stub it.
// The real adapter is wired in Task 9.
type ComputeClient interface {
	// GetClusterSparkVersion returns the Spark version string for a cluster.
	GetClusterSparkVersion(ctx context.Context, clusterID string) (string, error)
	// GetJobSparkVersion returns either a Spark version (isServerless=false) or a
	// serverless marker (isServerless=true) for a job, plus a recorded version string.
	GetJobSparkVersion(ctx context.Context, jobID string) (sparkVersion string, isServerless bool, version string, err error)
}

// TargetFlags holds the mutually-exclusive compute target flags from the CLI.
type TargetFlags struct {
	Cluster    string
	Serverless string
	Job        string
}

// BundleTarget is the three-state result of reading the bundle's configured
// target. Selected=false means nothing was configured.
type BundleTarget struct {
	ClusterID  string
	Serverless bool
	Selected   bool
}

// ValidateTargetFlags returns an error if more than one of the three flags is set.
// Cobra marks them mutually exclusive too; this guards the library path.
func ValidateTargetFlags(f TargetFlags) error {
	var set []string
	if f.Cluster != "" {
		set = append(set, "--cluster")
	}
	if f.Serverless != "" {
		set = append(set, "--serverless")
	}
	if f.Job != "" {
		set = append(set, "--job")
	}
	if len(set) > 1 {
		return fmt.Errorf("flags %s are mutually exclusive; specify at most one", strings.Join(set, " and "))
	}
	return nil
}

// ResolveTarget resolves the compute target using ordered precedence:
// --cluster flag → --serverless flag → --job flag → bundle target.
// PythonVersion is left empty; it is filled later from constraint data.
func ResolveTarget(ctx context.Context, f TargetFlags, c ComputeClient, bt BundleTarget) (*TargetInfo, error) {
	if f.Cluster != "" {
		v, err := c.GetClusterSparkVersion(ctx, f.Cluster)
		if err != nil {
			return nil, fmt.Errorf("resolving cluster %s: %w", f.Cluster, err)
		}
		return &TargetInfo{
			Kind:         "cluster",
			ClusterID:    f.Cluster,
			SparkVersion: v,
			EnvKey:       EnvKeyForSparkVersion(v),
		}, nil
	}

	if f.Serverless != "" {
		return &TargetInfo{
			Kind:   "serverless",
			EnvKey: EnvKeyForServerless(f.Serverless),
		}, nil
	}

	if f.Job != "" {
		_, isServerless, version, err := c.GetJobSparkVersion(ctx, f.Job)
		if err != nil {
			return nil, fmt.Errorf("resolving job %s: %w", f.Job, err)
		}
		if isServerless {
			// Default to v4 when the job is serverless; the serverless env version
			// is not recorded in the bundle/project (documented stand-in from the
			// original script).
			v := version
			if v == "" {
				v = "v4"
			}
			return &TargetInfo{
				Kind:   "serverless",
				EnvKey: EnvKeyForServerless(v),
			}, nil
		}
		return &TargetInfo{
			Kind:         "cluster",
			SparkVersion: version,
			EnvKey:       EnvKeyForSparkVersion(version),
		}, nil
	}

	// Fall back to bundle target.
	if !bt.Selected {
		return nil, NewError(ErrNoTargetSelected, nil,
			"No compute target is selected. Select a cluster or serverless target, or pass --cluster/--serverless/--job")
	}

	if bt.Serverless {
		// Default to serverless-v4: the serverless env version is not recorded
		// in the bundle/project (documented stand-in from the original script).
		return &TargetInfo{
			Kind:   "serverless",
			EnvKey: EnvKeyForServerless("v4"),
		}, nil
	}

	if bt.ClusterID != "" {
		v, err := c.GetClusterSparkVersion(ctx, bt.ClusterID)
		if err != nil {
			return nil, fmt.Errorf("resolving bundle cluster %s: %w", bt.ClusterID, err)
		}
		return &TargetInfo{
			Kind:         "cluster",
			ClusterID:    bt.ClusterID,
			SparkVersion: v,
			EnvKey:       EnvKeyForSparkVersion(v),
		}, nil
	}

	// Bundle target is selected but has neither serverless nor a cluster ID —
	// treat this the same as nothing selected so the user gets a clear message.
	return nil, NewError(ErrNoTargetSelected, nil,
		"No compute target is selected. Select a cluster or serverless target, or pass --cluster/--serverless/--job.")
}
