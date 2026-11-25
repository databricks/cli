package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

// servingEndpointsPath is the serving endpoints API. Dedicated endpoints are created
// and configured under dedicatedEndpointsPath.
const (
	servingEndpointsPath   = "/api/2.0/serving-endpoints"
	dedicatedEndpointsPath = servingEndpointsPath + "/pt"
)

// workloadTypeCPU is the workload type of a replica without a GPU, the API's
// default.
const workloadTypeCPU = "CPU"

// servingCompute is the serving compute of a version: the workload type of each
// replica, and the provisioned concurrency the endpoint scales between. ScaleToZero
// is set when the minimum is 0.
type servingCompute struct {
	WorkloadType   string
	MinConcurrency int
	MaxConcurrency int
	ScaleToZero    bool
}

// dedicatedCompute is the dedicated compute a dedicated served entity gets.
type dedicatedCompute struct {
	ComputeProfile        string
	ProvisionedModelUnits int64
}

// servedEntity is the one entity air serve serves on the endpoint: a version of the
// model on serving compute, or on dedicated compute when Dedicated is set.
type servedEntity struct {
	EntityName    string
	EntityVersion string
	// Serving is the serving compute.
	Serving servingCompute
	// Dedicated is the dedicated compute, nil for serving compute.
	Dedicated *dedicatedCompute
}

// servingInput is the entity as the serving endpoints API takes it: its scale as a
// range of provisioned concurrency, which the API takes instead of a workload size.
// scale_to_zero_enabled and a minimum of 0 are sent even though they are zero values,
// since the server's defaults aren't known to be zero.
func (e servedEntity) servingInput() serving.ServedEntityInput {
	return serving.ServedEntityInput{
		EntityName:                e.EntityName,
		EntityVersion:             e.EntityVersion,
		WorkloadType:              serving.ServingModelWorkloadType(e.Serving.WorkloadType),
		MinProvisionedConcurrency: e.Serving.MinConcurrency,
		MaxProvisionedConcurrency: e.Serving.MaxConcurrency,
		ScaleToZeroEnabled:        e.Serving.ScaleToZero,
		ForceSendFields:           []string{"MinProvisionedConcurrency", "ScaleToZeroEnabled"},
	}
}

// dedicatedServedEntity is a dedicated served entity as the dedicated endpoints API
// takes it. The SDK has no compute_profile, so these requests are sent as JSON. The
// entity_version is empty in a dry run, before the registration produces one.
type dedicatedServedEntity struct {
	EntityName            string `json:"entity_name"`
	EntityVersion         string `json:"entity_version,omitempty"`
	ProvisionedModelUnits int64  `json:"provisioned_model_units"`
	ComputeProfile        string `json:"compute_profile"`
}

// dedicatedEndpointConfig is the config of a dedicated endpoint.
type dedicatedEndpointConfig struct {
	ServedEntities []dedicatedServedEntity `json:"served_entities"`
}

// dedicatedEndpointUpdate replaces the config of a dedicated endpoint.
type dedicatedEndpointUpdate struct {
	Config dedicatedEndpointConfig `json:"config"`
}

// dedicatedEndpointRequest creates a dedicated endpoint.
type dedicatedEndpointRequest struct {
	Name   string                  `json:"name"`
	Config dedicatedEndpointConfig `json:"config"`
}

// dedicatedJSON is the entity as the dedicated endpoints API takes it.
func (e servedEntity) dedicatedJSON() dedicatedServedEntity {
	return dedicatedServedEntity{
		EntityName:            e.EntityName,
		EntityVersion:         e.EntityVersion,
		ProvisionedModelUnits: e.Dedicated.ProvisionedModelUnits,
		ComputeProfile:        e.Dedicated.ComputeProfile,
	}
}

// servingEndpoint is a serving endpoint as GET /api/2.0/serving-endpoints/{name}
// reports it, with the fields air serve reads. It replaces the SDK's type, which has
// no compute_profile for a dedicated served entity.
type servingEndpoint struct {
	Name          string                 `json:"name"`
	State         *serving.EndpointState `json:"state,omitempty"`
	Config        *endpointConfig        `json:"config,omitempty"`
	PendingConfig *endpointConfig        `json:"pending_config,omitempty"`
}

// endpointConfig is the served entities of an endpoint's current or pending config.
type endpointConfig struct {
	ServedEntities []observedEntity `json:"served_entities,omitempty"`
}

// observedEntity is a served entity as the endpoint reports it: a dedicated one by
// its units and profile, and any other by its workload type and concurrency range.
type observedEntity struct {
	EntityName                string            `json:"entity_name"`
	EntityVersion             string            `json:"entity_version"`
	WorkloadType              string            `json:"workload_type"`
	MinProvisionedConcurrency int               `json:"min_provisioned_concurrency"`
	MaxProvisionedConcurrency int               `json:"max_provisioned_concurrency"`
	ScaleToZeroEnabled        bool              `json:"scale_to_zero_enabled"`
	EnvironmentVars           map[string]string `json:"environment_vars"`
	ProvisionedModelUnits     int64             `json:"provisioned_model_units"`
	ComputeProfile            string            `json:"compute_profile"`
}

// endpointAPI calls the serving endpoints API for air serve. Endpoints are read as
// JSON, and dedicated endpoints written so; serving compute is written through the SDK.
type endpointAPI struct {
	w       *databricks.WorkspaceClient
	api     *client.DatabricksClient
	headers map[string]string
}

func newEndpointAPI(w *databricks.WorkspaceClient) (*endpointAPI, error) {
	api, err := client.New(w.Config)
	if err != nil {
		return nil, withCode(codeInternalError, false, fmt.Errorf("failed to create API client: %w", err))
	}
	return &endpointAPI{w: w, api: api, headers: auth.WorkspaceIDHeaders(w.Config)}, nil
}

// read returns the endpoint named name, and the error the API gives when it doesn't
// exist. A transient failure is repeated.
func (e *endpointAPI) read(ctx context.Context, name string) (*servingEndpoint, error) {
	return retry.With(ctx, retry.IsTransientRead, func() (*servingEndpoint, error) {
		endpoint := &servingEndpoint{}
		err := e.api.Do(ctx, http.MethodGet, servingEndpointsPath+"/"+name, e.headers, nil, nil, endpoint)
		return endpoint, err
	})
}

// get returns the endpoint named name, or nil if it doesn't exist.
func (e *endpointAPI) get(ctx context.Context, name string) (*servingEndpoint, error) {
	endpoint, err := e.read(ctx, name)
	if errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return nil, nil
	}
	return endpoint, err
}

// create creates the endpoint named name with entity. It returns once the API accepts
// the request, not once the endpoint is up.
func (e *endpointAPI) create(ctx context.Context, name string, entity servedEntity) error {
	if entity.Dedicated != nil {
		request := dedicatedEndpointRequest{
			Name:   name,
			Config: dedicatedEndpointConfig{ServedEntities: []dedicatedServedEntity{entity.dedicatedJSON()}},
		}
		return e.api.Do(ctx, http.MethodPost, dedicatedEndpointsPath, e.headers, nil, request, nil)
	}
	_, err := e.w.ServingEndpoints.Create(ctx, serving.CreateServingEndpoint{
		Name:   name,
		Config: &serving.EndpointCoreConfigInput{ServedEntities: []serving.ServedEntityInput{entity.servingInput()}},
	})
	return err
}

// update replaces the endpoint named name's served entity with entity. It returns once
// the API accepts the update.
func (e *endpointAPI) update(ctx context.Context, name string, entity servedEntity) error {
	if entity.Dedicated != nil {
		update := dedicatedEndpointUpdate{Config: dedicatedEndpointConfig{ServedEntities: []dedicatedServedEntity{entity.dedicatedJSON()}}}
		return e.api.Do(ctx, http.MethodPut, dedicatedEndpointsPath+"/"+name+"/config", e.headers, nil, update, nil)
	}
	_, err := e.w.ServingEndpoints.UpdateConfig(ctx, serving.EndpointCoreConfigInput{
		Name:           name,
		ServedEntities: []serving.ServedEntityInput{entity.servingInput()},
	})
	return err
}

// checkKind fails when the endpoint serves on dedicated compute and dedicated is
// false, or the reverse: the server converts neither kind of endpoint to the other.
// An endpoint that reports no served entity is taken as either kind.
func checkKind(endpoint *servingEndpoint, dedicated bool) error {
	served := append(currentEntities(endpoint), pendingEntities(endpoint)...)
	if len(served) == 0 {
		return nil
	}
	isDedicated := slices.ContainsFunc(served, func(e observedEntity) bool { return e.ComputeProfile != "" })
	if isDedicated == dedicated {
		return nil
	}
	kind, want := "serving compute", "dedicated compute"
	if isDedicated {
		kind, want = want, kind
	}
	return withCode(codeInvalidArgs, false, fmt.Errorf("endpoint %s serves on %s, and this version is served on %s; an endpoint can't change between the two, so set endpoint_name to another name", stripControl(endpoint.Name), kind, want))
}

// checkReplaceable fails unless the endpoint serves at most one entity, a version
// of model, in its current and pending configs. Replacing anything else would
// remove another model or a traffic split the user didn't ask to change. The
// endpoint lists every served model under served_entities too (served_models
// mirrors it), so served_entities covers both.
func checkReplaceable(endpoint *servingEndpoint, model string) error {
	for _, served := range [][]observedEntity{currentEntities(endpoint), pendingEntities(endpoint)} {
		if len(served) > 1 || (len(served) == 1 && !strings.EqualFold(served[0].EntityName, model)) {
			return withCode(codeInvalidArgs, false, fmt.Errorf("endpoint %s serves %s, and air serve replaces only a single version of %s; set endpoint_name to another name, or change this endpoint with databricks serving-endpoints update-config", stripControl(endpoint.Name), describeEntities(served), model))
		}
	}
	return nil
}

// serveOnEndpoint makes the endpoint named name serve only entity, calling progress
// on every poll. It creates the endpoint if it doesn't exist. An endpoint that
// already serves entity isn't updated. With watch, it waits until the endpoint is
// ready, including for an update to entity that an earlier air serve started;
// without it, it returns once the endpoint accepts the update. It reports whether
// the endpoint serves entity and is ready.
func serveOnEndpoint(ctx context.Context, w *databricks.WorkspaceClient, name string, entity servedEntity, watch bool, progress func(*servingEndpoint)) (bool, error) {
	e, err := newEndpointAPI(w)
	if err != nil {
		return false, err
	}
	endpoint, err := e.get(ctx, name)
	if err != nil {
		return false, err
	}
	if endpoint == nil {
		if err := e.create(ctx, name, entity); err != nil {
			return false, err
		}
		return watchServes(ctx, e, name, entity, watch, progress)
	}
	if updating(endpoint) && servesOnly(pendingEntities(endpoint), entity) {
		return watchServes(ctx, e, name, entity, watch, progress)
	}
	// The checks read the pending config too, so an endpoint air serve won't change is
	// refused before waiting behind its update.
	if err := checkEndpoint(endpoint, entity); err != nil {
		return false, err
	}
	if updating(endpoint) {
		// The endpoint takes no other update until this one finishes, so this waits
		// even without watch. How that update ends doesn't matter: either way the
		// endpoint then takes this one.
		endpoint, err = waitNotUpdating(ctx, e, name, progress, false)
		if err != nil {
			return false, err
		}
		if err := checkEndpoint(endpoint, entity); err != nil {
			return false, err
		}
	}
	if servesOnly(currentEntities(endpoint), entity) {
		return checkServes(endpoint, entity)
	}
	if err := e.update(ctx, name, entity); err != nil {
		return false, err
	}
	return watchServes(ctx, e, name, entity, watch, progress)
}

// checkEndpoint fails when air serve won't make endpoint serve entity: it serves
// something else, or is of the other kind.
func checkEndpoint(endpoint *servingEndpoint, entity servedEntity) error {
	if err := checkReplaceable(endpoint, entity.EntityName); err != nil {
		return err
	}
	return checkKind(endpoint, entity.Dedicated != nil)
}

// watchServes waits for the endpoint's update to entity to finish when watch is set.
func watchServes(ctx context.Context, e *endpointAPI, name string, entity servedEntity, watch bool, progress func(*servingEndpoint)) (bool, error) {
	if !watch {
		return false, nil
	}
	return waitServes(ctx, e, name, entity, progress)
}

// waitNotUpdating waits until the endpoint has no update in progress, calling
// progress on every poll. When own is set, the update is air serve's and its failure
// is an error; another update's failure only frees the endpoint. Each poll repeats a
// transient failure, which ends the SDK's own wait: an endpoint can take many minutes
// to start. Creates and config updates aren't repeated this way, since a repeat after
// an applied write would fail.
func waitNotUpdating(ctx context.Context, e *endpointAPI, name string, progress func(*servingEndpoint), own bool) (*servingEndpoint, error) {
	return retries.Poll(ctx, untilDeadline(ctx), func() (*servingEndpoint, *retries.Err) {
		endpoint, err := e.read(ctx, name)
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
			if !own {
				return endpoint, nil
			}
			return nil, retries.Halt(fmt.Errorf("the endpoint's update ended in %s; check its events and logs in the Serving UI", state))
		default:
			return nil, retries.Continues(string(state))
		}
	})
}

// waitServes waits until the endpoint's update finishes, then checks that it
// serves entity.
func waitServes(ctx context.Context, e *endpointAPI, name string, entity servedEntity, progress func(*servingEndpoint)) (bool, error) {
	endpoint, err := waitNotUpdating(ctx, e, name, progress, true)
	if err != nil {
		return false, err
	}
	return checkServes(endpoint, entity)
}

// errEndpointNotReady reports an endpoint that serves the version but isn't ready.
// Running air serve again doesn't make it ready; it reports success once the
// endpoint recovers, so the error isn't marked retryable.
var errEndpointNotReady = errors.New("endpoint isn't ready")

// checkServes reports whether endpoint serves only entity and is ready to serve,
// failing when it doesn't.
func checkServes(endpoint *servingEndpoint, entity servedEntity) (bool, error) {
	name := stripControl(endpoint.Name)
	if !servesOnly(currentEntities(endpoint), entity) {
		return false, fmt.Errorf("endpoint %s was changed to serve %s while air serve waited for it", name, describeEntities(currentEntities(endpoint)))
	}
	if endpoint.State == nil || endpoint.State.Ready != serving.EndpointStateReadyReady {
		return false, withCode(codeEndpointNotReady, false, fmt.Errorf("%w: endpoint %s serves version %s of %s but isn't ready; check its events and logs in the Serving UI", errEndpointNotReady, name, entity.EntityVersion, entity.EntityName))
	}
	return true, nil
}

func updating(endpoint *servingEndpoint) bool {
	return endpoint.State != nil && endpoint.State.ConfigUpdate == serving.EndpointStateConfigUpdateInProgress
}

func currentEntities(endpoint *servingEndpoint) []observedEntity {
	if endpoint.Config == nil {
		return nil
	}
	return endpoint.Config.ServedEntities
}

func pendingEntities(endpoint *servingEndpoint) []observedEntity {
	if endpoint.PendingConfig == nil {
		return nil
	}
	return endpoint.PendingConfig.ServedEntities
}

// describeEntities names served entities for an error, e.g. "version 2 of a.b.c".
func describeEntities(served []observedEntity) string {
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

// servesOnly reports whether served is exactly entity. A dedicated entity must match
// its units and compute profile; any other must match its workload type, concurrency
// range and scale-to-zero setting, and have no environment variables, since air serve
// sets none.
func servesOnly(served []observedEntity, entity servedEntity) bool {
	if len(served) != 1 {
		return false
	}
	s := served[0]
	// Unity Catalog names are case-insensitive, and the endpoint reports them as
	// Unity Catalog stores them.
	if !strings.EqualFold(s.EntityName, entity.EntityName) || s.EntityVersion != entity.EntityVersion {
		return false
	}
	if entity.Dedicated != nil {
		return s.ProvisionedModelUnits == entity.Dedicated.ProvisionedModelUnits && s.ComputeProfile == entity.Dedicated.ComputeProfile
	}
	// Where GPU APIs are off, the endpoint reports a CPU entity without its workload
	// type, the API's default.
	workloadType := s.WorkloadType
	if workloadType == "" {
		workloadType = workloadTypeCPU
	}
	return workloadType == entity.Serving.WorkloadType &&
		s.MinProvisionedConcurrency == entity.Serving.MinConcurrency &&
		s.MaxProvisionedConcurrency == entity.Serving.MaxConcurrency &&
		s.ScaleToZeroEnabled == entity.Serving.ScaleToZero &&
		len(s.EnvironmentVars) == 0
}

// untilDeadline is the time left before ctx's deadline, which air serve always
// sets from --timeout.
func untilDeadline(ctx context.Context) time.Duration {
	deadline, _ := ctx.Deadline()
	return time.Until(deadline)
}
