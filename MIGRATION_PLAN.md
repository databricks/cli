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
