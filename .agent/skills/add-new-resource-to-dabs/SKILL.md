---
name: Add a new resource to DABs
description: Use this skill if tasked with adding a new resource
---

# Rules for adding a new resource

ALL FILE PATHS AND COMMANDS ARE REFERENCED FROM/TO BE EXECUTED FROM THE `databricks/cli` REPO ROOT.

DO NOT EDIT AUTO-GENERATED FILES! RUN THE APPROPRIATE `make` COMMAND INSTEAD!

## Test server

- Add the resource to `FakeWorkspace` in [fake_workspace.go](./libs/testserver/fake_workspace.go)
- Provide CRUD methods for the resource to the workspace
    - Create a new file in [libs/testserver/](./libs/testserver/) for the resource (one file per resource, even if resources are linked)
- Add the APIs to [handlers.go](./libs/testserver/handlers.go) based on the API spec for this resource
    - Usually CRUD is `POST`, `GET`, `PATCH`, `DELETE`
    - Make sure the URL paths are as expected from the API spec
    - It is possible the resource has separate `PATCH` endpoints, handle them all.

## Include resource in auto-generation

- Add the resource to [resources.go](./bundle/config/resources.go)
- Implement the resource in [bundle/config/resources/](./bundle/config/resources/)
    - Use one file per resource
    - URL should be the one from the Databricks Workspace
    - TODO: don't add unit tests as we prefer testserver over mocks
- Trigger auto-generations
    - `make schema`
        - You first should download the open API spec with `genkit get $(cat .codegen/_openapi_sha)`
            - If `genkit` isn't available, fall back to downloading `https://openapi.dev.databricks.com/OPENAPI_SHA/specs/all-internal.json`
        - Then run `DATABRICKS_OPENAPI_SPEC=/path/to/spec/file make schema`
    - `make schema-for-docs`
    - `make generate-validation`
    - `make generate-direct`
    - `make generate`?
- Add overrides to specs
    - TODO: on what conditions should we modify the below files?
    - annotations_openapi_overrides.yml
    - annotations.yml

## Terraform

Going forward, Terraform engine will be deprecated and new resources should not be added there.

For new resources that don't include terraform, make sure to add it to exclude lists when TF-related unit tests are failing (or seem suitable due to having a list of unuspported TF resources).

There's also logic in [validate_direct_only_resources.go](./bundle/config/mutator/validate_direct_only_resources.go) which needs to be edited.

## Direct Deployment

[all.go](./bundle/direct/dresources/all.go)

Then implement all methods (one file per resource).
- `DoUpdate` should be left unimplemented if there is no update API

You have to take care to wire fields together correctly.
- Ideally the reponses contain the initial config (even if echo'ed under a different sub-field)
- Parameters that aren't echoed (e.g. `req.foo` vs `resp.effective_foo`) should not be mapped.
    - Either ignore remote, or file a ticket to get the original request value in the reponse
- If needed, edit [resources.yml](./bundle/direct/dresources/resources.yml) (unless auto-gen'ed files already have these specs)
    - Mark fields as needed to get the correct treatment
- General rules (TODO read from code):
    - `DoUpdate` API allows for property changes
    - `DoUpdateWithID` allows for rename, add `update_id_on_changes` to resources.yml
    - `recreate_on_changes` for immutable fields
    - `ignore_remote_changes` in case any fields don't have any direct response (e.g. `input_only`)

## Acceptance tests

Fix all unit tests. Adding a new resource currently requires adding dummy stuff all over the place.

First, do auto-generated stuff:
- `make test-update`
    - [out.fields.txt](./acceptance/bundle/refschema/out.fields.txt) should get updated

Add acceptance tests:
- One folder per resource under [acceptance/bundle/resources/](./acceptance/bundle/resources/)
- Create folders for `basic` and other test cases that cover specific behaviour

Add invariant tests:
- Add resource to the test cases in [acceptance/bundle/invariant/](./acceptance/bundle/invariant/)

If multiple resources have complex interaction, that should be tested in acceptance tests too (but do still test resources in isolation as much as possible)!

## Guardrails

If the resource is non-ephemeral in nature, deletion should trigger a warning + confirmation. Implement this in [deploy.go](./bundle/phases/deploy.go).

Likewise, if deleting + recreating the resource causes significant downtime (e.g. vector search index requiring re-indexing post-deploy), this should trigger a warning + confirmation, too.


# INDEX

- [./acceptance/README.md](./acceptance/README.md)
- [./acceptance/bundle/invariant/README.md](./acceptance/bundle/invariant/README.md)
- [./bundle/tests/README.md](./bundle/tests/README.md)
- [./bundle/direct/dresources/README.md](./bundle/direct/dresources/README.md)

# GENERAL RULES

- Use the `ForceSendFields: utils.FilterFields` pattern
