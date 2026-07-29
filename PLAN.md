# Bundle bitmap

## Context

CLI telemetry (`go/dabs-telemetry`) tracks DABs feature usage. Today each tracked feature needs
custom code: add a key constant in `bundle/metrics/metrics.go` and call `b.Metrics.SetBoolValue(...)`
from a mutator (see `bundle/config/mutator/collect_escape_telemetry.go`, PR #2949). This means only
hand-picked features are tracked, only after someone wires them up.

Features almost always map to *config fields*. So instead of tracking fields one-by-one, we encode the
presence/absence of **every** bundle config field as one bit in a compact bitmap. The bitmap's schema
is an ordered list of field paths derived by reflecting over `config.Root`; it is recorded in-repo and
embedded into the binary. It is append-only, so a newer binary can decode an older bitmap and vice
versa (old binary decodes a newer bitmap but lacks names for the trailing new bits).

This PR delivers the hidden `databricks bundle bitmap` command group + the schema infrastructure +
tests. Wiring the encoded bitmap into the deploy telemetry payload (`BundleDeployExperimental`) is a
**follow-up PR** (keeps this one focused, per repo rules).

## Key mechanic (validated against the code)

Two existing walkers in `libs/structs/structwalk` are the whole basis:

- `WalkType(reflect.TypeFor[config.Root](), ...)` → visits every field of the *type*, giving JSON
  paths with wildcards: `resources.jobs.*.name`, `variables.*.default`, `bundle.git.branch`. This
  ordered list **is the bitmap schema** (bit *i* ↔ line *i*). Paths render via `structpath.PatternNode`
  (`.*` for map values, `[*]` for slice elements).
- `Walk(loadedRoot, ...)` → visits scalar **leaf values** of an actual config, giving concrete paths:
  `resources.jobs['my_job'].name`, honoring `omitempty` (a zero+omitempty field is not emitted = "not
  set"). Concrete paths use `structpath.PathNode` (`['key']` for map keys, `[N]` for indices).

A concrete path maps to its schema pattern by rebuilding the node chain: dot-string→`NewPatternDotString`,
map-key(bracket-string)→`NewPatternDotStar`, index→`NewPatternBracketStar`, then `.String()`. The result
is byte-identical to what `WalkType` emitted, so setting a bit is an O(1) lookup in a
`map[patternString]int`. Setting a leaf bit also sets all ancestor-prefix bits (so intermediate nodes
like `resources.jobs.*` read as set when any descendant is present).

Known v1 gap (documented in code): `any`-typed subtrees (`variables.*.default`, `definitions`) are
walked by neither walker, so those bits stay 0. Acceptable.

## New package: `bundle/bitmap/`

- `schema.txt` — committed, embedded schema (ordered pattern list, one per line). ~thousands of lines,
  like `acceptance/bundle/refschema/out.fields.txt`.
- `embed.go` — `//go:embed schema.txt` → `var schemaBytes []byte` (auto-picked up by
  `tools/list_embeds.py` / `EMBED_SOURCES`, so edits invalidate build/test caches).
- `bitmap.go` — core, no cobra/bundle-loading deps:
  - `EmbeddedSchema() []string` — split embedded `schema.txt`.
  - `WalkSchema() ([]string, error)` — fresh `WalkType` over `config.Root`, pruning noisy/duplicated
    subtrees `targets`, `environments`, `__locations` (targets/environments are nilled after
    `SelectTarget`; `__locations` is output-only internal). Mirrors refschema's permissions/grants prune.
  - `Merge(old, fresh []string) (merged, added []string)` — append-only: keep `old` order, append
    patterns in `fresh` not already present. Removed fields stay (bit stability).
  - `Bits(cfg config.Root, schema []string) ([]bool, error)` — `Walk` the value, normalize each leaf +
    its prefixes, set matching bits.
  - `normalize(*structpath.PathNode) string` — the rebuild described above.
  - `Encode(bits []bool, context uint16) (string, error)` — build the byte format below, DEFLATE
    (`compress/flate`, deterministic — no gzip mtime), base64-encode.
  - constants: `magic = "DBTB"` (0x44424254; avoids gzip 0x1f8b / zlib 0x78), `contextFullBundle uint16 = 0`.

### Encoded format (uncompressed, before DEFLATE + base64)

| field        | bytes        | value |
|--------------|--------------|-------|
| Magic        | 4            | `DBTB` |
| Version/Size | 4            | uint32 big-endian = N (bit count = len(schema)); doubles as version since N only grows |
| Context      | 2            | uint16 big-endian = `contextFullBundle` |
| Bitmap       | ceil(N/8)    | bit *i* = schema line *i*, MSB-first within each byte |

Magic is inside the payload (chosen not to collide with compression magic) so a decoder can tell a raw
payload from a compressed one.

## New command group: `cmd/bundle/bitmap/`

Hidden parent `bitmap` under `bundle`, registered in `cmd/bundle/bundle.go` via
`cmd.AddCommand(bitmap.NewBitmapCommand())`. Subcommands (names per spec):

- `schema` — prints `EmbeddedSchema()`, one pattern per line. No bundle load. (`Args: root.NoArgs`.)
- `update-schema [--validate]` — no bundle load. Computes `WalkSchema()`, `Merge` with embedded.
  - default: prints merged schema to stdout (Taskfile redirects into `schema.txt`).
  - `--validate`: if `added` is non-empty, print the new fields and return an error (drift guard);
    else print nothing, exit 0.
- `bitmap-text` — loads bundle via `utils.ProcessBundle(cmd, utils.ProcessOptions{Validate: true})`
  (like `cmd/bundle/validate.go`), computes `Bits` against the embedded schema, prints `0/1 <pattern>`
  per schema line. Primary golden artifact.
- `bitmap` — same load, prints `Encode(...)` (base64). Deterministic.

## Tests

New acceptance dir `acceptance/bundle/bitmap/`:
- `databricks.yml` — a fixture named `test-bundle` exercising several field kinds (a job with tags,
  a variable, workspace/presets fields).
- `test.toml` — `Local = true`, `Cloud = false`.
- `script`:
  ```
  trace $CLI bundle bitmap schema > out.schema.txt
  trace $CLI bundle bitmap bitmap-text > out.bitmap-text.txt
  trace $CLI bundle bitmap bitmap
  trace $CLI bundle bitmap update-schema --validate
  ```
- Golden `output.txt` + generated `out.schema.txt`, `out.bitmap-text.txt`. `--validate` asserts the
  committed schema is not stale.

`no_drift` invariant (`acceptance/bundle/invariant/no_drift/script`): add
`$CLI bundle bitmap bitmap > LOG.bitmap` — `LOG`-prefixed files are logged, not diffed
(`acceptance_test.go`), so it runs the encoder over all ~55 invariant configs "just for fun".

Unit tests in `bundle/bitmap/bitmap_test.go`: `Merge` append-only behavior, `normalize` on
struct/map/slice paths, `Encode` header bytes + round-trip decode.

## Taskfile wiring (`Taskfile.yml`)

- Add `generate-bitmap-schema`: `go run . bundle bitmap update-schema > bundle/bitmap/schema.txt`
  (sources: `bundle/**/*.go`, `go.mod`, `go.sum`, `{{.EMBED_SOURCES}}`; generates
  `bundle/bitmap/schema.txt`). Model on `generate-refschema` / `generate-schema`.
- Add it to the `generate` aggregator and to `generate-check` (CI drift guard via `git diff --exit-code`).
- Per spec ("before we run acceptance tests, update the schema"): make `test-acc` (and `test-update`)
  depend on `generate-bitmap-schema` so the embedded schema is current before the acceptance suite runs.

## Files touched

- new: `bundle/bitmap/{schema.txt,embed.go,bitmap.go,bitmap_test.go}`
- new: `cmd/bundle/bitmap/bitmap.go`
- new: `acceptance/bundle/bitmap/{databricks.yml,test.toml,script,output.txt,out.*.txt}`
- edit: `cmd/bundle/bundle.go` (register group)
- edit: `acceptance/bundle/invariant/no_drift/script` (LOG.bitmap line)
- edit: `Taskfile.yml` (generate-bitmap-schema + wiring)
- new: `.nextchanges/cli/<slug>.md` (changelog fragment — user-visible hidden command)

## Verification

1. `./task build`
2. `./task generate-bitmap-schema` → writes `bundle/bitmap/schema.txt`; inspect it looks like a
   superset of `out.fields.txt` covering the whole Bundle.
3. `go test ./bundle/bitmap/...`
4. `go test ./acceptance -run TestAccept/bundle/bitmap -update` then re-run without `-update` (clean).
5. `go test ./acceptance -run TestAccept/bundle/invariant/no_drift -tail -test.v` — confirm
   `LOG.bitmap` is produced and no panic.
6. `./task lint-q`
