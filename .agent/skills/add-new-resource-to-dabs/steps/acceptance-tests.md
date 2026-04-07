# Acceptance tests

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
