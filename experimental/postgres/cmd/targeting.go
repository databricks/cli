package postgrescmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/lakebase/target"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// resolvedTarget carries everything the query command needs to dial Postgres:
// the endpoint host (resolved through the SDK) and a short-lived OAuth token.
// `kind` records whether we resolved an autoscaling endpoint or a provisioned
// instance, so the caller can pick the right default database name and emit
// kind-appropriate logging.
type resolvedTarget struct {
	Kind     targetKind
	Host     string
	Username string
	Token    string
	// Display strings used only for human-readable logs / errors.
	DisplayName string
}

type targetKind int

const (
	kindAutoscaling targetKind = iota
	kindProvisioned
)

// targetingFlags is the user-supplied targeting input. Exactly one of:
//   - target (full path or instance name)
//   - project (with optional branch and endpoint)
//
// must be set. Validated by validateTargeting before any SDK call.
type targetingFlags struct {
	target   string
	project  string
	branch   string
	endpoint string
}

func (f targetingFlags) hasGranular() bool {
	return f.project != "" || f.branch != "" || f.endpoint != ""
}

// validateTargeting enforces "exactly one targeting form" before any SDK call.
// Returns a typed error so the JSON envelope renderer (added in a later PR)
// can surface a structured error.
//
// We require --branch when --endpoint is set: this command is non-interactive
// and scriptable, and the alternative (auto-select-then-look-up-endpoint)
// produces confusing errors when the resolved branch does not contain the
// requested endpoint. Asking the user to be explicit is friendlier.
func validateTargeting(f targetingFlags) error {
	switch {
	case f.target == "" && !f.hasGranular():
		return errors.New("must specify --target or --project")
	case f.target != "" && f.hasGranular():
		return errors.New("--target is mutually exclusive with --project, --branch, --endpoint")
	case f.target == "" && f.project == "" && (f.branch != "" || f.endpoint != ""):
		return errors.New("--project is required when using --branch or --endpoint")
	case f.endpoint != "" && f.branch == "":
		return errors.New("--branch is required when using --endpoint")
	}
	return nil
}

// resolveTarget translates the validated flags into a resolvedTarget.
//
// --target accepts either an autoscaling resource path (starts with "projects/")
// or a provisioned instance name (everything else). Granular flags
// (--project, --branch, --endpoint) target autoscaling only.
func resolveTarget(ctx context.Context, f targetingFlags) (*resolvedTarget, error) {
	w := cmdctx.WorkspaceClient(ctx)

	switch {
	case f.target != "" && target.IsAutoscalingPath(f.target):
		spec, err := target.ParseAutoscalingPath(f.target)
		if err != nil {
			return nil, err
		}
		return resolveAutoscaling(ctx, w, spec)

	case f.target != "":
		return resolveProvisioned(ctx, w, f.target)

	default:
		spec := target.AutoscalingSpec{
			ProjectID:  f.project,
			BranchID:   f.branch,
			EndpointID: f.endpoint,
		}
		return resolveAutoscaling(ctx, w, spec)
	}
}

// resolveProvisioned looks up a provisioned instance and issues a token. The
// instance must be in the AVAILABLE state; transitional states return an
// error pointing the user at the lifecycle they are waiting on.
func resolveProvisioned(ctx context.Context, w *databricks.WorkspaceClient, instanceName string) (*resolvedTarget, error) {
	instance, err := target.GetProvisioned(ctx, w, instanceName)
	if err != nil {
		return nil, err
	}

	if instance.State != database.DatabaseInstanceStateAvailable {
		return nil, fmt.Errorf("database instance %q is not ready for accepting connections (state: %s)", instance.Name, instance.State)
	}
	if instance.ReadWriteDns == "" {
		return nil, fmt.Errorf("database instance %q has no read/write DNS yet", instance.Name)
	}

	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	token, err := target.ProvisionedCredential(ctx, w, instance.Name)
	if err != nil {
		return nil, err
	}

	return &resolvedTarget{
		Kind:        kindProvisioned,
		Host:        instance.ReadWriteDns,
		Username:    user.UserName,
		Token:       token,
		DisplayName: instance.Name,
	}, nil
}

// resolveAutoscaling expands a partial spec into a fully-resolved endpoint and
// issues a short-lived OAuth token. Missing branch/endpoint IDs are
// auto-selected when exactly one candidate exists; ambiguity propagates as an
// AmbiguousError with the list of choices.
func resolveAutoscaling(ctx context.Context, w *databricks.WorkspaceClient, spec target.AutoscalingSpec) (*resolvedTarget, error) {
	if spec.ProjectID == "" {
		var err error
		spec.ProjectID, err = target.AutoSelectProject(ctx, w)
		if err != nil {
			return nil, err
		}
	}

	project, err := target.GetProject(ctx, w, spec.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if spec.BranchID == "" {
		spec.BranchID, err = target.AutoSelectBranch(ctx, w, project.Name)
		if err != nil {
			return nil, err
		}
	}

	if spec.EndpointID == "" {
		branchName := project.Name + "/branches/" + spec.BranchID
		spec.EndpointID, err = target.AutoSelectEndpoint(ctx, w, branchName)
		if err != nil {
			return nil, err
		}
	}

	endpoint, err := target.GetEndpoint(ctx, w, spec.ProjectID, spec.BranchID, spec.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	if err := checkEndpointReady(endpoint); err != nil {
		return nil, err
	}

	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	token, err := target.AutoscalingCredential(ctx, w, endpoint.Name)
	if err != nil {
		return nil, err
	}

	return &resolvedTarget{
		Kind:        kindAutoscaling,
		Host:        endpoint.Status.Hosts.Host,
		Username:    user.UserName,
		Token:       token,
		DisplayName: endpoint.Name,
	}, nil
}

// checkEndpointReady returns an error if the endpoint is not in a connectable
// state. Idle endpoints are considered connectable (Lakebase wakes them on
// dial); the connect retry loop handles the wake-up window.
func checkEndpointReady(endpoint *postgres.Endpoint) error {
	if endpoint.Status == nil {
		return errors.New("endpoint status is not available")
	}
	if endpoint.Status.Hosts == nil || endpoint.Status.Hosts.Host == "" {
		return errors.New("endpoint host information is not available")
	}
	switch endpoint.Status.CurrentState {
	case postgres.EndpointStatusStateActive, postgres.EndpointStatusStateIdle:
		return nil
	default:
		return fmt.Errorf("endpoint is not ready for accepting connections (state: %s)", endpoint.Status.CurrentState)
	}
}
