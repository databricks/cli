# CLI spec shards

Genkit's `cli_v2` mode writes the checked-in, uncompressed CLI spec in `.codegen/`.
Command generation, bundle schema generation, and the direct-engine Python tools
consume this directory without a Universe checkout or network access.

- `index.json` versions the format, lists every schema package and service with
  its SHA-256 digest, preserves service order, and contains command metadata.
- `schemas/<package>.json` holds every schema in that package, including enums
  and unreferenced types.
- `commands/<package>/<service>.json` holds one resolved service.

Every JSON file is smaller than 1,000,000 bytes. Go's `ReadDir`/`Parse` and Python's
`bundle/direct/tools/clijson.py` validate the complete inventory, digests, paths,
and identities before consumers generate output. Missing files, deleted schema
keys, unexpected files, and unsupported format versions fail rather than silently
producing a reduced API surface. Digests detect accidental changes; the reviewed
manifest is the source of truth, not a cryptographic signature.

The producer validates and stages the entire spec before publishing, replaces
owned directories, and publishes the index last. Publication errors roll back
the previous files. Readers during publication may fail validation and should
retry after generation completes; this is not an atomic multi-directory swap.
A killed producer can leave `.cli-shards.lock` and `.cli-shards-*` recovery
directories. Inspect/recover the previous contents before removing a stale lock
and regenerating. Other `.codegen` metadata is preserved.

## Regeneration and rollout

Run `./task generate-cligen generate-schema generate-direct` for ordinary offline
regeneration. `./task test-clijson-python` and `go test ./internal/clijson` test the
readers using the same small fixture emitted by the upstream producer.

`./task generate-clijson` builds genkit and OpenAPI at `.codegen/_openapi_sha`.
The upstream `cli_v2` change must land before moving that pin to an SDK-aligned
Universe commit containing it. The existing pin predates `cli_v2`; it cannot
refresh this format. Do not change the pin independently of the Go SDK or add a
downstream splitter as a compatibility fallback. The initial shard migration
uses the existing logical spec and the new upstream writer, with no API changes.
