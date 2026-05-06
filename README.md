# docgen

This branch is the publish target for `bundle/schema/jsonschema_for_docs.json`.

It is updated automatically by the `update-schema-docs` workflow on every `v*`
release tag push: the workflow regenerates the schema (so newly-released fields
get their `x-since-version` annotation) and commits the result here.

Do not edit this branch by hand.
