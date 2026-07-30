package localenv

import (
	"context"
	"errors"
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
	// JobTaskEnvironment resolves a single job task's compute to an environment.
	//
	// A job can bind multiple tasks to different environment versions, so the
	// target is a specific task, not the whole job. taskKey selects it; an empty
	// taskKey is a request to enumerate — the method returns ErrTaskKeyRequired
	// wrapping the job's task keys so the caller can prompt for one. An unknown
	// taskKey returns an error naming the available keys.
	//
	// For a serverless task, isServerless is true and version is the task's
	// recorded serverless environment version (read directly — no fallback). For
	// a classic task, isServerless is false and sparkVersion is the task cluster's
	// runtime.
	JobTaskEnvironment(ctx context.Context, jobID, taskKey string) (sparkVersion string, isServerless bool, version string, err error)
}

// ErrTaskKeyRequired is returned by JobTaskEnvironment when --job-task names a
// job but no task, so ResolveCompute can classify it as E_USAGE (the user must
// pick a task) rather than E_RESOLVE. It carries the job's task keys so the
// error message can list them.
type ErrTaskKeyRequired struct {
	JobID    string
	TaskKeys []string
}

func (e *ErrTaskKeyRequired) Error() string {
	return fmt.Sprintf("job %s has multiple tasks; specify one: --job-task %s.<task-key> (available: %s)",
		e.JobID, e.JobID, strings.Join(e.TaskKeys, ", "))
}

// ComputeFlags holds the mutually-exclusive compute target flags from the CLI.
type ComputeFlags struct {
	Cluster     string
	ClusterName string
	Serverless  string
	// JobTask is "<job-id>.<task-key>" (or a bare "<job-id>", which resolves to
	// an E_USAGE listing the job's task keys). A job can bind tasks to different
	// environment versions, so the target is a specific task, not the whole job.
	JobTask string
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
const noTargetMessage = "No compute target is selected. Select a cluster or serverless target, or pass --cluster-id / --cluster-name / --serverless-version / --job-task"

// ValidateComputeFlags returns an error if more than one of the target flags is set.
// Cobra marks them mutually exclusive too; this guards the library path.
func ValidateComputeFlags(f ComputeFlags) error {
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
	if f.JobTask != "" {
		set = append(set, "--job-task")
	}
	if len(set) > 1 {
		return fmt.Errorf("flags %s are mutually exclusive; specify at most one", strings.Join(set, " and "))
	}
	return nil
}

// ResolveCompute resolves the compute target using ordered precedence:
// --cluster-id → --cluster-name → --serverless-version → --job-task → bundle target.
// PythonVersion is left empty; it is filled later from constraint data.
//
// Incompatible flags are rejected up front: without this a library caller that
// bypasses Cobra (which also marks the flags mutually exclusive) and passes more
// than one target flag would have all but the first precedence branch silently
// ignored, resolving a different target than requested.
func ResolveCompute(ctx context.Context, f ComputeFlags, c ComputeClient, bt BundleTarget) (*ComputeInfo, error) {
	if err := ValidateComputeFlags(f); err != nil {
		return nil, NewError(ErrResolve, err, "invalid compute target flags")
	}

	if f.Cluster != "" {
		v, err := c.GetClusterSparkVersion(ctx, f.Cluster)
		if err != nil {
			return nil, NewError(ErrResolve, err, "resolving cluster %s", f.Cluster)
		}
		return &ComputeInfo{
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
		return &ComputeInfo{
			Source:       "cluster",
			ClusterID:    id,
			SparkVersion: v,
			EnvKey:       EnvKeyForSparkVersion(v),
		}, nil
	}

	if f.Serverless != "" {
		if !ValidServerlessVersion(f.Serverless) {
			return nil, NewError(ErrResolve, nil, "invalid --serverless-version %q: expected a version number like 5", f.Serverless)
		}
		return &ComputeInfo{
			Source:            "serverless",
			ServerlessVersion: NormalizeServerless(f.Serverless),
			EnvKey:            EnvKeyForServerless(f.Serverless),
		}, nil
	}

	if f.JobTask != "" {
		// Split "<job-id>.<task-key>" on the FIRST dot: task keys may themselves
		// contain dots, so everything after the first separator is the task key.
		// A bare "<job-id>" (no dot) leaves taskKey empty, which JobTaskEnvironment
		// treats as an enumerate request and reports back via ErrTaskKeyRequired.
		jobID, taskKey, _ := strings.Cut(f.JobTask, ".")
		sparkVersion, isServerless, version, err := c.JobTaskEnvironment(ctx, jobID, taskKey)
		if err != nil {
			// A missing task key is a usage error (the user must pick a task),
			// distinct from a genuine resolve failure (unknown key, API error).
			if _, ok := errors.AsType[*ErrTaskKeyRequired](err); ok {
				return nil, NewError(ErrUsage, err, "specify a job task")
			}
			return nil, NewError(ErrResolve, err, "resolving job task %s", f.JobTask)
		}
		if isServerless {
			// The task's serverless environment version is read directly from its
			// bound environment, so there is no version to guess: unlike the bundle
			// path, the job-task path never applies the serverless-vN fallback.
			return &ComputeInfo{
				Source:            "job",
				ServerlessVersion: NormalizeServerless(version),
				EnvKey:            EnvKeyForServerless(version),
			}, nil
		}
		// Classic compute: sparkVersion is the task cluster's runtime.
		return &ComputeInfo{
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
		return &ComputeInfo{
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
		return &ComputeInfo{
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
