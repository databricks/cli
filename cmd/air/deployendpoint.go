package aircmd

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

// servedEntity is the one served entity air deploy puts on the endpoint.
func (c *deployConfig) servedEntity(version string) serving.ServedEntityInput {
	return serving.ServedEntityInput{
		EntityName:         c.RegisteredModel,
		EntityVersion:      version,
		WorkloadType:       serving.ServingModelWorkloadType(c.Compute.WorkloadType),
		WorkloadSize:       c.Compute.WorkloadSize,
		ScaleToZeroEnabled: c.Compute.ScaleToZeroEnabled,
		EnvironmentVars:    c.servedEnvironmentVars(),
		// scale_to_zero_enabled: false is the user's choice, so send it rather than leave it
		// to the server's default.
		ForceSendFields: []string{"ScaleToZeroEnabled"},
	}
}

// getEndpoint returns the endpoint named name, or nil if it doesn't exist.
func getEndpoint(ctx context.Context, w *databricks.WorkspaceClient, name string) (*serving.ServingEndpointDetailed, error) {
	endpoint, err := retry.With(ctx, retry.IsTransientRead, func() (*serving.ServingEndpointDetailed, error) {
		return w.ServingEndpoints.GetByName(ctx, name)
	})
	if errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return nil, nil
	}
	return endpoint, err
}

// waitNotUpdating waits until the endpoint has no update in progress, calling
// progress on every poll. It polls as the SDK's WaitGetServingEndpointNotUpdating
// does, except that each poll repeats a transient failure, which ends the SDK's
// wait: an endpoint can take many minutes to start. Creates and config updates
// aren't repeated this way, since a repeat after an applied write would fail.
func waitNotUpdating(ctx context.Context, w *databricks.WorkspaceClient, name string, progress func(*serving.ServingEndpointDetailed)) (*serving.ServingEndpointDetailed, error) {
	return retries.Poll(ctx, untilDeadline(ctx), func() (*serving.ServingEndpointDetailed, *retries.Err) {
		endpoint, err := retry.With(ctx, retry.IsTransientRead, func() (*serving.ServingEndpointDetailed, error) {
			return w.ServingEndpoints.GetByName(ctx, name)
		})
		if err != nil {
			return nil, retries.Halt(err)
		}
		if progress != nil {
			progress(endpoint)
		}
		var state serving.EndpointStateConfigUpdate
		if endpoint.State != nil {
			state = endpoint.State.ConfigUpdate
		}
		switch state {
		case serving.EndpointStateConfigUpdateNotUpdating:
			return endpoint, nil
		case serving.EndpointStateConfigUpdateUpdateFailed, serving.EndpointStateConfigUpdateUpdateCanceled:
			return nil, retries.Halt(fmt.Errorf("the endpoint's update ended in %s; check its events and logs in the Serving UI", state))
		default:
			return nil, retries.Continues(string(state))
		}
	})
}

// checkEndpoint fails when the endpoint named name, if it exists, serves
// something air deploy won't replace with a version of model. air deploy checks
// it before registering, so a wrong endpoint name fails before any weights are
// copied.
func checkEndpoint(ctx context.Context, w *databricks.WorkspaceClient, name, model string) error {
	endpoint, err := getEndpoint(ctx, w, name)
	if err != nil || endpoint == nil {
		return err
	}
	return checkReplaceable(endpoint, model)
}

// checkReplaceable fails unless the endpoint serves at most one entity, a version
// of model, in its current and pending configs. Replacing anything else would
// remove another model or a traffic split the user didn't ask to change. The
// endpoint lists every served model under served_entities too (served_models
// mirrors it), so served_entities covers both.
func checkReplaceable(endpoint *serving.ServingEndpointDetailed, model string) error {
	for _, served := range [][]serving.ServedEntityOutput{currentEntities(endpoint), pendingEntities(endpoint)} {
		if len(served) > 1 || (len(served) == 1 && !strings.EqualFold(served[0].EntityName, model)) {
			return withCode(codeInvalidArgs, false, fmt.Errorf("endpoint %s serves %s, and air deploy replaces only a single version of %s; set endpoint to another name, or change this endpoint with databricks serving-endpoints update-config", stripControl(endpoint.Name), describeEntities(served), model))
		}
	}
	return nil
}

// serveOnEndpoint makes the endpoint named name serve only entity and waits until
// it is ready, calling progress on every poll. It creates the endpoint if it
// doesn't exist. An endpoint that already serves entity isn't updated, and an
// update to entity that an interrupted air deploy started is waited for.
func serveOnEndpoint(ctx context.Context, w *databricks.WorkspaceClient, name string, entity serving.ServedEntityInput, progress func(*serving.ServingEndpointDetailed)) (*serving.ServingEndpointDetailed, error) {
	endpoint, err := getEndpoint(ctx, w, name)
	if err != nil {
		return nil, err
	}
	if endpoint == nil {
		_, err := w.ServingEndpoints.Create(ctx, serving.CreateServingEndpoint{
			Name:   name,
			Config: &serving.EndpointCoreConfigInput{ServedEntities: []serving.ServedEntityInput{entity}},
		})
		if err != nil {
			return nil, err
		}
		return waitServes(ctx, w, name, entity, progress)
	}
	if updating(endpoint) {
		if servesOnly(pendingEntities(endpoint), entity) {
			return waitServes(ctx, w, name, entity, progress)
		}
		// The endpoint takes no other update until this one finishes.
		endpoint, err = waitNotUpdating(ctx, w, name, progress)
		if err != nil {
			return nil, err
		}
	}
	if err := checkReplaceable(endpoint, entity.EntityName); err != nil {
		return nil, err
	}
	if servesOnly(currentEntities(endpoint), entity) {
		return checkServes(endpoint, entity)
	}
	_, err = w.ServingEndpoints.UpdateConfig(ctx, serving.EndpointCoreConfigInput{
		Name:           name,
		ServedEntities: []serving.ServedEntityInput{entity},
	})
	if err != nil {
		return nil, err
	}
	return waitServes(ctx, w, name, entity, progress)
}

// waitServes waits until the endpoint's update finishes, then checks that it
// serves entity.
func waitServes(ctx context.Context, w *databricks.WorkspaceClient, name string, entity serving.ServedEntityInput, progress func(*serving.ServingEndpointDetailed)) (*serving.ServingEndpointDetailed, error) {
	endpoint, err := waitNotUpdating(ctx, w, name, progress)
	if err != nil {
		return nil, err
	}
	return checkServes(endpoint, entity)
}

// errEndpointNotReady reports an endpoint that serves the version but isn't ready.
// Running air deploy again doesn't make it ready; it reports success once the
// endpoint recovers, so the error isn't marked retryable.
var errEndpointNotReady = errors.New("endpoint isn't ready")

// checkServes returns endpoint if it serves only entity and is ready to serve.
func checkServes(endpoint *serving.ServingEndpointDetailed, entity serving.ServedEntityInput) (*serving.ServingEndpointDetailed, error) {
	name := stripControl(endpoint.Name)
	if !servesOnly(currentEntities(endpoint), entity) {
		return nil, fmt.Errorf("endpoint %s was changed to serve %s while air deploy waited for it", name, describeEntities(currentEntities(endpoint)))
	}
	if endpoint.State == nil || endpoint.State.Ready != serving.EndpointStateReadyReady {
		return nil, withCode(codeEndpointNotReady, false, fmt.Errorf("%w: endpoint %s serves version %s of %s but isn't ready; check its events and logs in the Serving UI", errEndpointNotReady, name, entity.EntityVersion, entity.EntityName))
	}
	return endpoint, nil
}

func updating(endpoint *serving.ServingEndpointDetailed) bool {
	return endpoint.State != nil && endpoint.State.ConfigUpdate == serving.EndpointStateConfigUpdateInProgress
}

func currentEntities(endpoint *serving.ServingEndpointDetailed) []serving.ServedEntityOutput {
	if endpoint.Config == nil {
		return nil
	}
	return endpoint.Config.ServedEntities
}

func pendingEntities(endpoint *serving.ServingEndpointDetailed) []serving.ServedEntityOutput {
	if endpoint.PendingConfig == nil {
		return nil
	}
	return endpoint.PendingConfig.ServedEntities
}

// describeEntities names served entities for an error, e.g. "version 2 of a.b.c".
func describeEntities(served []serving.ServedEntityOutput) string {
	if len(served) == 0 {
		return "nothing"
	}
	names := make([]string, 0, len(served))
	for _, s := range served {
		name := stripControl(s.EntityName)
		if s.EntityVersion == "" {
			names = append(names, name)
			continue
		}
		names = append(names, fmt.Sprintf("version %s of %s", stripControl(s.EntityVersion), name))
	}
	return strings.Join(names, " and ")
}

// servesOnly reports whether served is exactly entity, with the same workload and
// environment variables.
func servesOnly(served []serving.ServedEntityOutput, entity serving.ServedEntityInput) bool {
	if len(served) != 1 {
		return false
	}
	s := served[0]
	// Unity Catalog names are case-insensitive, and the endpoint reports them as
	// Unity Catalog stores them.
	return strings.EqualFold(s.EntityName, entity.EntityName) &&
		s.EntityVersion == entity.EntityVersion &&
		s.WorkloadType == entity.WorkloadType &&
		s.WorkloadSize == entity.WorkloadSize &&
		s.ScaleToZeroEnabled == entity.ScaleToZeroEnabled &&
		maps.Equal(s.EnvironmentVars, entity.EnvironmentVars)
}

// untilDeadline is the time left before ctx's deadline, which air deploy always
// sets from --timeout.
func untilDeadline(ctx context.Context) time.Duration {
	deadline, _ := ctx.Deadline()
	return time.Until(deadline)
}
