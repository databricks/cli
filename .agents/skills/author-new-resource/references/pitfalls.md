# Pitfalls

Patterns that have cost resource PRs weeks of review. Each entry: what it looks like, why it is a
problem, what to do instead. Reviewers use the same list, so finding yourself here before you open
the PR saves a round trip.

## Design

**A list key built from several fields, or from a value that is not a field.** `KeyedSlices` returns
a field name and a value; the engine later looks elements up by that field. A key such as
`"member", "a=x|b=y"` pairs elements for the diff but can never be looked up, so every generated
rule under that list stops working and the resource grows plan overrides to compensate. What to do:
go back to the API readiness gate, section 2.2. The fix is a client-settable key field on the
element, in the API. Not a composite key, not an engine change.

**`OverrideChangeDesc` in a new resource.** It exists for the rare case the yml rules cannot
express. In a new resource it almost always means one of: the key problem above, an update endpoint
that supports only part of the resource, or a rule that belongs in the proto. Each of those is an API
conversation. If you still need it, the comment on the override must say which yml rule it replaces
and why that rule could not be written.

**`DoUpdate` for an endpoint that cannot update.** If the API's update can only remove list members,
or only touch two fields, do not implement `DoUpdate` and classify the rest; recreate on any change
like the resources that have no update at all. An update path that needs code to decide what is
updatable is a sign the API should mark the rest `IMMUTABLE`.

**Hand-written rules that mirror annotations.** `recreate_on_changes` for a field the proto should
mark `IMMUTABLE`; `ignore_remote_changes` for a field that should be `OUTPUT_ONLY` or `INPUT_ONLY`.
Acceptable only as a bridge: one comment per rule naming the API behaviour, and the proto change
filed before the PR opens.

**A mirror config struct instead of embedding the SDK type.** The bundle schema follows the API.
Embed the SDK resource type in the config type; add a wrapper only when the SDK hides identity
behind `json:"-"` fields, and keep the wrapper to identity plus the embedded body.

**Convenience syntax in the resource.** Friendlier duration units, shorthand that expands one config
entry into several API objects, defaults the API does not have. These are bundle-wide features that
belong in the config phase for every resource, in their own PR, after a design discussion. The
resource PR speaks the API's language. The one exception is a file reference for a payload the
server re-serializes, which the readiness gate (check 2.3) treats as part of drift detection.

## Adapter

**Waiting inline in `DoCreate` when the id is already known.** State is persisted after `DoCreate`
returns; a poll loop inside it means an interrupted deploy leaves an untracked resource. Return the
id and poll in `WaitAfterCreate`. If the API only yields the id at completion, say so in a comment
and in the API readiness note.

**Prefixing errors with the resource or id.** The engine already prints `cannot create
resources.<plural>.<key>: ...` and `updating id=...`. A second prefix repeats the same id twice in
one line. Return SDK errors as they are.

**Checking what the framework guarantees.** Receivers, the arguments to `DoCreate`, `DoUpdate` and
`DoDelete`, and the id passed in are never nil or empty; a check "just in case" is dead code. The
id you return from `DoCreate` is your responsibility: an empty one is saved as state and the next
plan creates the resource again.

**Logic in `RemapState`.** It is a field-by-field copy. Remapping a differently named field there
hides real drift. Put the remap in `DoRead` and make the field part of the remote type.

**A custom update mask.** Send `*` when the API supports it, otherwise a static list with a comment.
Never derive the mask from the planned changes.

## Fake server

**Modelling backend validation.** Rules about which combinations of fields the service accepts,
cascade semantics between members, error-code aggregation. The CLI only surfaces these errors; it
does not branch on them. Each such rule is an untested claim about the backend that the goldens then
pin. Model create, read, update, delete and server-assigned fields; cover service behaviour with the
cloud test.

**Failure triggered by a magic string in a fixture.** A feature name containing `fails` that makes
the fake return an error is invisible to the next reader. Use `fault.py` and `kill_after.py`.

**Operations that are always already complete.** If the API is asynchronous, the fake should be
able to return a pending operation at least once, or the polling loop is never exercised locally.

**Plain-text error bodies.** The SDK maps errors to typed sentinels only for JSON bodies with
`error_code`. Return JSON.

## Tests

**An invariant config that is not in the matrix.** Adding the template is half the step. Register it
under `EnvMatrix.INPUT_CONFIG`, exclude the legs it cannot run, and prove it ran.

**Many directories, one behaviour each.** Five or more directories under `resources/<plural>/`
usually means backend refusal cases got their own tests. The canonical set is `basic`, `lifecycle`,
`remote-delete`, plus one directory per plan behaviour the resource adds (an immutable field, a
dependency). Fold the rest.

**Plan-only coverage of a recreate.** `bundle plan` showing `recreate` proves the classifier; only
`bundle deploy --auto-approve` proves the delete and create requests. Execute it.

**Dumping an API response into the golden.** The backend adds fields; the golden breaks. Project to
the fields you assert.

**Titles as paragraphs.** `title` takes one line. Explanation belongs in a code comment, not in
`output.txt`.

**Goldens with unresolved `${resources...}` references.** The script printed config, not state.
Print what the deploy produced.

**A `[[Server]]` stub for a route the fake models.** Stubs are for routes the fake does not have.
Teach the fake instead.

## Wiring

**Approval prompt in deploy but not destroy.** Both groups must list the type, with a destroy
message.

**A type missing from the deployment-history allowlist, silently.** Either add it (and coordinate
the backend allowlist) or exclude the `DMS=true` leg with a comment. Never leave the leg failing or
unexplained.

**The bind test in the resource directory.** It lives under `acceptance/bundle/deployment/bind/`.

**Placeholder descriptions left in `annotations.yml`.** Fill the resource-level and field
descriptions, then regenerate the schema; do not edit `jsonschema.json` by hand.

## Scope and prose

**Changes to another resource riding along.** Approval prompts for a related type, a mutator
tweak, a message rewording. Each is its own PR.

**Comments that explain the engine.** A comment on the adapter should say what the API does that
the code is reacting to. How the planner matches patterns, or what `ignore_remote_changes` means,
belongs in the engine's README, once.

**A shared helper extracted from the exemplar's pattern.** Two resources carrying the same block in
the same shape is the intended result of copying the exemplar. A new `script.prepare`, `libs/`
function or `acceptance/bin/` script that exists only to deduplicate your copy widens the PR and
gives review a second design to read. Copy the block, keep its shape, and use a helper only when the
repo already has one for that job.

**PR descriptions that carry what the code should.** Decisions, API requests and test runs belong
in the code: a comment on the rule or mask, the test list entry, `Cloud = true` in `test.toml`. The
description says what the resource is and the shape of its identity and update, in a few
sentences, and stops.
