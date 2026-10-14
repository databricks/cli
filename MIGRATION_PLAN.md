# Migration: databricks-sdk-go -> sdk-go (temporary notes, delete before merge)

## Key facts about sdk-go
- One Go module per service (`github.com/databricks/sdk-go/<svc>`), versioned package `<svc>/vN`.
- Per-service clients: `jobs.NewClient(ctx, clientopts...)`, methods like `CreateJob(ctx, req)`.
- Model types have NO json tags and no MarshalJSON; optional fields are pointers; no ForceSendFields.
- Oneofs are protobuf style (unexported interface + `Parent_Field_Alt` wrapper structs).
- JSON lives in unexported `wire.go` (`xWire` structs with json tags + `xToWire`/`xFromWire`).
- Missing APIs in sdk-go: workspace import/export/list/mkdirs/delete/get-status, bundle deployments
  (`/api/2.0/bundle/...`), legacy SQL (`/api/2.0/preview/sql/...`), quality monitors v1, iamv2.
- Auth: sdk-go has only pat/m2m/u2m/oidc and itself depends on databricks-sdk-go (u2m).

## Design
1. `libs/sdktypes/<svc>/vN`: generated (internal/sdktypesgen) from sdk-go `wire.go`:
   JSON-tagged exported mirror types (wire shape, flattened oneofs), enum aliases + const re-exports,
   converters `XToSDK(*X) (*sdk.X, error)` and `XFromSDK(*sdk.X) (*X, error)`.
   Bundle config, direct engine state, testserver use these mirrors. API calls use sdk-go.
2. `libs/sdkclients` (or similar): builds sdk-go clients from the CLI's resolved auth
   (credentials adapter over resolved config, like libs/filer/files_client_auth.go).
3. Missing APIs: hand-written thin clients in the CLI.
4. Delete ForceSendFields support (libs/structs, libs/dyn/convert, etc.).
5. cmd/workspace, cmd/account: retarget internal/cligen templates to sdk-go.

## Progress
- [x] branch denik/migrate-sdk-go, go.mod has all sdk-go modules

## Conventions (all agents MUST follow)
- Shared working tree, other agents edit other directories concurrently. NEVER run git commit/stash/checkout/reset/restore,
  never revert files you did not change. Only edit files in your assigned area unless strictly necessary (then report it).
- Build/vet only your own packages (e.g. `go build ./libs/foo/... && go vet ./libs/foo/...`); errors in other areas are expected.
- Clients: `*sdkclients.Clients` (libs/sdkclients) replaces both `*databricks.WorkspaceClient` and `*databricks.AccountClient`.
  It has `.Config *config.Config` (old SDK config is KEPT for auth/profile resolution) and one field per sdk-go module,
  e.g. `w.Jobs` (*jobs.Client from github.com/databricks/sdk-go/jobs/v2), `w.Catalogs`, `w.Schemas`, `w.ServingEndpoints`,
  `w.Libraries` (clusterlibraries), `w.SecretsUc` (uc/secrets), `w.Alerts` (alerts/v2), `w.AlertsV1`. See libs/sdkclients/services.go.
  `cmdctx.WorkspaceClient(ctx)`, `cmdctx.AccountClient(ctx)`, `b.WorkspaceClient(ctx)` all return `*sdkclients.Clients`.
- sdk-go call style: `w.Jobs.GetJob(ctx, jobs.GetJobRequest{JobId: new(id)})`. Optional fields are pointers; Go 1.26+ `new(expr)`
  makes a pointer to a value. Iterators: `for x, err := range w.Jobs.ListJobsIter(ctx, req)`. LROs return waiters with `.Wait(ctx)`.
- JSON: sdk-go types have NO json tags. For anything serialized (bundle config, state, testserver, --json flags, output rendering)
  use the generated mirrors in libs/sdktypes/<module>/<vN> (same type names as sdk-go, json tags, flattened oneofs, enum aliases
  and constants re-exported) and convert with `jobs.JobSettingsToSDK(&m)` / `jobs.JobSettingsFromSDK(s)` (generated, return error).
  Import convention in mixed code: mirror as the plain name (`jobs`), sdk-go as `sdkjobs`.
- Raw REST calls: `libs/apiclient` (`apiclient.New(cfg)`, `c.Do(ctx, method, path, headers, query, request, response)`), same
  signature as the old `client.DatabricksClient.Do`.
- Errors: sdk-go returns `*apierr.APIError` from github.com/databricks/sdk-go/core/apierr (methods, not fields:
  `HTTPStatusCode()`, `Message()`). Use libs/apierrs for classification with old semantics:
  `apierrs.Is(err, apierrs.ErrNotFound)`, `apierrs.IsMissing(err)`, `apierrs.ErrorCode(err)`, `apierrs.StatusCode(err)`.
- No ForceSendFields anywhere: zero values that must be sent are expressed with pointers.
- No experimental/mocks (gomock) for SDK clients: rewrite such unit tests against libs/testserver or a small httptest server.
- Missing in sdk-go, hand-written in the CLI: workspace object API (libs/workspaceapi), bundle deployments
  (libs/bundledeployments), legacy quality monitors. Reach them via fields on sdkclients.Clients.
- Keep acceptance test outputs stable where reasonable; do not run `-update` on acceptance tests unless told to.
