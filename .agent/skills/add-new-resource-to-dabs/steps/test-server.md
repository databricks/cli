# Add resource to the test server

Checklist:
- Add resource state to `libs/testserver/fake_workspace.go`.
- Add one resource-focused implementation file under `libs/testserver/` for CRUD behavior.
- Wire endpoints in `libs/testserver/handlers.go` to match API paths and methods.
- Cover any split update endpoints (`PATCH` variants) exposed by the API.

Keep this step implementation-focused. Canonical resource behavior and constraints live in:
- `bundle/direct/dresources/README.md`
