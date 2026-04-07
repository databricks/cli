# Add resource to the test server

- Add the resource to `FakeWorkspace` in [fake_workspace.go](./libs/testserver/fake_workspace.go)
- Provide CRUD methods for the resource to the workspace
    - Create a new file in [libs/testserver/](./libs/testserver/) for the resource (one file per resource, even if resources are linked)
- Add the APIs to [handlers.go](./libs/testserver/handlers.go) based on the API spec for this resource
    - Usually CRUD is `POST`, `GET`, `PATCH`, `DELETE`
    - Make sure the URL paths are as expected from the API spec
    - It is possible the resource has separate `PATCH` endpoints, handle them all.
