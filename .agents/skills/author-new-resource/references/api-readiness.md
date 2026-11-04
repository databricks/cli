# API readiness gate

Run this before writing any CLI code. The bundle engine derives most of its plan behaviour from
the API's proto annotations, so a gap in the API becomes hand-written resource code, and a gap in
the API's shape becomes engine workarounds. Both are expensive to carry and neither is accepted in
a new-resource PR. Fix the API first; the API is usually still in preview when this question comes
up and the proto is the cheapest place to change.

Each check ends in one of three verdicts:

- **STOP**: do not start the CLI work until the API changes. Send the note in section 4.
- **ASK**: file the API change, continue in parallel, and carry a small, commented hand-written rule
  until the annotation lands. Say so in the PR.
- **NOTE**: record the decision in the PR description. No API change needed.

## 1. How the CLI consumes the API

- Proto `google.api.field_behavior` values reach `.codegen/cli.json` as `behaviors` on each field:
  `IMMUTABLE`, `INPUT_ONLY`, `OPTIONAL`, `OUTPUT_ONLY`, `UNORDERED_LIST`. `REQUIRED` and
  `IDENTIFIER` are not carried, so identity has to be read from the proto itself.
- `./task generate-direct` turns those into `bundle/direct/dresources/configs/<plural>.generated.yml`:
  `OUTPUT_ONLY` and `INPUT_ONLY` become `ignore_remote_changes` (reasons `spec:output_only`,
  `spec:input_only`); `IMMUTABLE` becomes `recreate_on_changes` (reason `spec:immutable`). A
  behaviour on a container field applies to every child.
- Only fields that are part of the resource's state (listed in
  `acceptance/bundle/refschema/output/<plural>.txt` as `STATE` or `ALL`) get rules.
- The generator picks the API type for a resource by field overlap and writes it to
  `apitypes.generated.yml`. When the heuristic picks a request type instead of the resource type,
  pin the right one in `apitypes.yml` with a one-line reason.
- Everything else is hand-written in `configs/<plural>.yml`: `provided_id_fields`,
  `updatable_id_fields`, `stable_output_fields`, `backend_defaults`, `ignore_local_changes`. Each
  hand-written rule is a fact about the API the proto could not express. Keep them few, and give
  each a comment that names the API behaviour it stands in for.

## 2. Checks

### 2.1 Identity

**Check.** The resource has one stable identifier the server returns on GET and accepts on GET,
Update and Delete. Best: a `name` field marked `IDENTIFIER` (AIP-122, AIP-203). If the client picks
the ID, the Create request carries `<resource>_id` and the entity's own ID field is `OUTPUT_ONLY`
(AIP-133).

**Verify.** In the proto: `rg -n "IDENTIFIER|identifier: true|identifier_field_names" <proto>`;
identity may be declared on the field or in the file-level resource option
(`databricks.api.resource` with `identifier_field_names`). Read the Get request's path parameters.
The identifier is not visible in `cli.json`.

**Verdicts.**
- One field, returned on GET: **NOTE**. Declare it in `provided_id_fields` (recreate when the user
  changes it) or `updatable_id_fields` (rename in place, needs `DoUpdateWithID`).
- Parent plus leaf id on the Create request, with the resource carrying only the server-derived
  `name` (the AIP-133 shape): **NOTE**. The config type wraps `parent` and `<resource>_id` around
  the embedded resource, both go in `provided_id_fields`, and the engine id is the returned `name`.
- An id the client may set or leave for the server to assign: **NOTE**, treat it as user-provided
  (`provided_id_fields`); if the user later sets a different value the resource recreates.
- Two or more fields together identify the resource (for example an object type plus an object id
  in the URL): **ASK** for a single `name`, and in the meantime compose the engine ID from the
  components with a small pair of helpers (`<Resource>ID(...)`, `Parse<Resource>ID(...)`) on the
  config type and list every component in `provided_id_fields`. This pattern exists in the repo and
  needs no engine change.
- No field the client can predict and the server returns: **STOP**.

### 2.2 Repeated fields of resource-like elements

**Check.** Every `repeated` message field whose elements have their own lifecycle (they are created,
updated or deleted individually, carry server-assigned ids, or appear in other RPCs) is either keyed
by one client-settable field per element, or expressed as a list of resource names (AIP-144,
AIP-124). Lists of plain settings (tags, column names, notification addresses) need no key.

**Verify.** For each `repeated` field in the resource message, open the element type. Does it have a
field the user sets that is unique within the list and echoed on GET? Is the list's order
preserved by GET? The API docs often say "order is not significant"; that is a key requirement, not
an excuse.

**Why.** The engine pairs list elements by a key (`KeyedSlices`) and then looks elements up by
`field=value`. The key must be a real field present in config, state and the GET response. A key
composed from several fields, or a value the client cannot set, cannot be looked up, and the
resource then needs plan overrides that reimplement engine behaviour.

**Verdicts.**
- Keyed by one user-set field, or a list of names: **NOTE**. Declare it in `KeyedSlices`.
- Elements with a lifecycle but no single key, order not preserved: **STOP**. Ask for a
  client-settable key field on the element (the pattern of `task_key` on job tasks), or for the
  list to become resource names. Do not build a composite key. Do not propose an engine change.
- Elements that only exist as part of this resource and have no identity at all (pure config):
  **NOTE**, positional diff is fine.

### 2.3 Inputs round-trip

**Check.** Every field the client sends on Create or Update is returned by GET with the same value,
or is annotated `INPUT_ONLY` with a reason (write-only secrets are the legitimate case).

**Verify.** Compare the Create body with the GET response in the proto or API docs. In `cli.json`
look for `INPUT_ONLY`. Quote the annotation line for every field you classify; a doc comment that
says "not returned" or "input only" on a field annotated `OPTIONAL` is an ASK, not a NOTE, because
the generator reads annotations, not prose. After generation, `configs/<plural>.generated.yml` must
list each non-echoed field under `ignore_remote_changes` with `spec:input_only`.

**Verdicts.**
- All inputs echoed, or non-echoed ones annotated: **NOTE**.
- A field is not echoed and not annotated: **ASK** for the annotation (or for the field to be
  echoed). Until then a hand-written `ignore_remote_changes` rule with a comment is acceptable for
  a setting, never for the resource's main content: ignoring a content field disables drift
  detection for exactly what the user cares about.
- The main content is echoed but re-serialized by the server (reformatted document, re-keyed ids):
  **ASK** for a stable representation. Decide with the API team, before the PR, how drift is
  detected: a canonical form the adapter produces in `DoRead`, a server-provided hash, or an
  etag. A file reference for such a payload is part of correctness here, not convenience syntax.
- An input field the server rejects on Create or Update whenever it is set: **ASK** to remove it
  from the public surface or to implement it; check the create and update validators, not only
  the proto.
- Whole configuration blocks are not echoed (the server keeps the intent private): **ASK** loudly.
  The engine cannot detect drift on them at all; say so in the PR.
- A field is returned only when a query flag asks for it: **NOTE**, and `DoRead` must pass the
  flag. If the server also rewrites the value (reformats a document, re-keys ids), the round-trip
  can never settle: **ASK** for a stable representation, and decide with the API team whether the
  bundle or the server owns that content.

### 2.4 Immutable fields

**Check.** Every field the API rejects a change to is `IMMUTABLE` (AIP-203). Fields that are
immutable only in some states must not carry the annotation; document the condition instead.

**Verify.** Compare `IMMUTABLE` in the proto with the Update method's documentation and, if you
have access, with the server's validation code. Internal proto comment markers such as
`(--[Create:REQ Update:IGN]--)` describe create-only behaviour the annotation does not carry; they
are evidence for an ASK, not a substitute for the annotation. After generation, each one appears in
`configs/<plural>.generated.yml` under `recreate_on_changes` with `spec:immutable`.

**Verdicts.**
- Annotated: **NOTE**.
- Not annotated but documented as create-only: **ASK** for the annotation; carry a hand-written
  `recreate_on_changes` rule with a comment naming the API behaviour, and remove it once the
  generated file covers it.
- Immutable only in some states (settable once, or fixed only after a sub-object exists): no
  annotation and no recreate rule, which would also fire on the allowed transition. **NOTE** when
  the API documents the condition; **ASK** for documentation when the only evidence is a server
  validator. Either way, say in the PR that a later change fails at deploy with the server's
  error, and test the allowed transition in the `lifecycle` test.
- The API has a "cannot be changed" rule nobody has written down: **ASK**, and test it against a
  live workspace before relying on it.

### 2.5 Server-computed fields

**Check.** Timestamps, status, server-assigned ids and derived values are `OUTPUT_ONLY`. A
user-settable field with a server default stays `OPTIONAL`; the default is documented.

**Verify.** `cli.json` `behaviors`; generated `ignore_remote_changes` with `spec:output_only`.

**Verdicts.** Annotated: **NOTE**. A computed field left unannotated: **ASK**; expect a phantom diff
on every plan until it is fixed, and cover it with a hand-written rule in the meantime. A field the
user may set that is wrongly marked `OUTPUT_ONLY`: **ASK**; the engine will ignore user edits to it.

A server-filled field annotated `IMMUTABLE` instead of `OUTPUT_ONLY` (a derived `name`, a
server-assigned id) is an **ASK** for the right annotation, but as long as the config never sets it
no hand rule is needed: the planner treats "unset in config and state, set remotely" as a backend
default and skips it, so the generated recreate rule does not fire on its own. Confirm with a plan
against the fake after the first deploy.

### 2.6 Update semantics

**Check.** Update replaces the whole resource with `update_mask: "*"` (AIP-134), or documents a
closed list of updatable fields. Everything the user can set is either updatable or `IMMUTABLE`.

**Verify.** The Update request has a `FieldMask`; the docs state whether `*` is accepted. Try it
against a live workspace once; some services accept the mask and ignore fields anyway.

**Verdicts.**
- `*` works and every field is updatable or immutable: **NOTE**. The adapter sends `*`.
- A documented static field list: **NOTE**. The adapter sends that list, never a list derived from
  the plan.
- `*` accepted only by an internal code path, or the public doc says it is unsupported: **ASK**
  for public documentation of `*` or of the closed list, and read the server's mask validator, not
  only the proto comment.
- A mask path the server rejects depending on the resource's state (a list field on a resource
  that has no principal, for example): **ASK**; a static mask then fails for some resources, and
  the PR must say which.
- A full-body PATCH or PUT with no mask: **NOTE** if the server treats absent fields as cleared (that
  is full replacement). If absent fields keep their old value, removing a field from config can
  never clear it: **ASK** for a mask or for documented clearing semantics, and test clearing in the
  `lifecycle` test.
- No Update at all: **NOTE**. Do not write `DoUpdate`; declare every settable field immutable so the
  planner recreates. A test in the repo checks this.
- Update exists but can change only a subset of what Create accepts, with no annotation telling
  the two apart: **ASK** for `IMMUTABLE` on the rest (2.4).
- Membership changes only through custom add/remove methods, or an update that can remove members
  but not add them: **ASK** for full replacement through the standard Update (AIP-144). Until it
  lands, do not implement `DoUpdate`: every change recreates the resource, and the PR says so. If a
  recreate is unacceptable for this resource (it drops data that takes hours to rebuild), that
  turns the ASK into a **STOP**. Never classify changes in Go to work around it.

### 2.7 Long-running operations

**Check.** A Create or Update that returns an operation declares `operation_info` (AIP-151) and the
resource id is available from the Create response or the operation metadata before the operation
completes.

**Verify.** `rg -n "operation_info" <proto>`; read the operation's metadata type.

**Why.** The engine persists the id right after `DoCreate` returns and polls afterwards in
`WaitAfterCreate`. If the id exists only in the final response, the adapter has to wait inline and
an interrupted deploy leaves an untracked resource.

**Verdicts.** Id available early: **NOTE**, use the wait hooks. Id only at completion: **ASK** for
the id in metadata; until then wait inline and say in the adapter comment why the hook does not fit.

The wait hooks receive the id, not the operation, so they poll the resource, or poll the operation
endpoint with a short loop when the SDK waiter cannot be rebuilt from the id. Either is fine inside
the hook; neither belongs in `DoCreate`, `DoUpdate` or `DoDelete`. If the resource has no
`state` field (AIP-216) that reports a failed or pending operation, a failure is indistinguishable
from "still running" or, for a failed delete, from success: **ASK** for a `state` on the resource
that the hooks can read. Also read the operation message: an operation that reports `done` without
setting `error` when the work failed makes the SDK waiter return success; that is an **ASK** for a
correct `error` on the operation, and until then the adapter must inspect the resource state after
the wait.

### 2.8 Delete semantics

**Check.** Delete returns `NOT_FOUND` for a missing resource unless `allow_missing` is set, and
either completes synchronously or is an operation with a documented completion signal (AIP-135).
The declared error codes are on the RPC's `error_codes` list in the proto; a status enum with a
pending-delete value is the usual hint that delete is asynchronous or soft. If Delete is refused while child objects exist, the children must be bundle resources the engine
can order through `lifecycle.depends_on` or references; otherwise `bundle destroy` cannot succeed
without manual steps. Say which in the PR.

**Verdicts.** Standard: **NOTE**. Async delete with no completion signal, or a soft delete that
keeps the resource readable for a while (GET still returns it, perhaps in a trashed or deleting
state): **ASK** for a documented completion contract; implement `IsGone` on the terminal state and
`WaitAfterDelete` as the repo's existing resources do, and cover delete-then-recreate in the
`lifecycle` test.

### 2.9 Read after create

**Check.** GET returns the resource as soon as Create (or its operation) completes.

**Why.** The planner treats a 404 on read as "gone" and plans another create.

**Verdicts.** Immediate: **NOTE**. Documented lag: **ASK**, and cover it in `WaitAfterCreate`.
Not documented either way: **ASK** the service to state it, and let the `lifecycle` test's
create-then-plan prove it against a real workspace. A 404 observed after create is a **STOP**.

### 2.10 Launch stage

**Check.** The resource, its methods and new fields carry a `stage`. PyDABs generates Python classes
only for `GA` and `PUBLIC_PREVIEW` resources; `PRIVATE_PREVIEW` and `PUBLIC_BETA` are excluded, and
fields at those stages are marked experimental.

**Verify.** Eligibility is decided by the stage on the resource message, not on its fields. A
resource whose fields are all private preview but whose message carries no stage is generated as
GA. After `./task generate-schema`, check the `x-databricks-launch-stage` on the resource's entry
in `bundle/schema/jsonschema.json`, and `bundle/internal/annotation/preview.go` for CLI-side
overrides.

**Verdicts.** Set on the resource and matching the fields: **NOTE** the stage and the resulting
PyDABs eligibility in the PR. Missing on the resource, or lower than its fields: **ASK**, and add
the CLI-side override so a preview resource does not ship as a public Python API in the meantime.

### 2.11 ETag

**Check.** If the resource is edited concurrently by people and automation, it has `string etag`
(AIP-154) and Update honours it.

**Verdicts.** Present, and Update or Delete honour it as a precondition (a mismatch fails with
`ABORTED` or 409): **ASK** the team how the bundle should use it, and write in the PR which channel
detects drift if the adapter does not send it. Present but recomputed by the server on its own
schedule (content migrations, re-keying): **ASK** for a content-stable etag or a documented rule, and
note that the adapter omits it on update. Absent: **NOTE**; drift detection falls back to full read
comparison.

### 2.12 References from existing resources

**Check.** If an existing bundle resource already carries a field that names this new type (an app
naming its space, a job naming a pool), the two must deploy and destroy in the right order. The
engine orders by `${resources...}` references and `lifecycle.depends_on`; a plain string field
gives it nothing.

**Verdicts.** The referrer's field can hold a `${resources.<plural>.<key>.<field>}` reference and
the API deletes the referenced object only after its referrers are gone: **NOTE**, and add an
acceptance test that deploys and destroys both. The referenced object's delete is refused or races
while referrers exist and nothing signals completion: **ASK** for a completion signal, and record
the destroy race in the PR.

### 2.13 Singletons

**Check.** A singleton (one per parent, no id) defines Get and Update only (AIP-156).

**Verdicts.** Clean singleton: **NOTE**. Create means update and Delete means reset; say so in the
adapter and in the PR. Check `apitypes.generated.yml` picked the resource type and pin it in
`apitypes.yml` if it picked a request type.

## 3. What the API linter already covers

"Linter clean" in universe does not mean "ready for DABs". Blocking rule groups include resource Get
and List, output-only resource ids, Create and Update and Delete request shapes, declarative
friendly fields (`name`, `uid`, `display_name`, `create_time`, `update_time`, `delete_time`), and
`OUTPUT_ONLY` on state fields. Not blocking, so check them yourself: field behaviour on every field
(2.3 to 2.5), LRO annotations (2.7), etag (2.11), singleton method rules (2.12). Every
`method_exception`, `message_exception` or `field_exception` in the proto is a place to look
twice; exceptions mark exactly the deviations that cost the CLI hand-written code.

## 4. Note to the API team

Send this when any check is STOP or ASK. Keep each item to the field, the annotation, and the AIP.

```
Subject: <service> API readiness for bundle support (<resource>)

We checked <resource> against what the bundle deployment engine derives from proto annotations.
The items below need an API change; the CLI does not work around them.

1. Identity (AIP-122, AIP-203): <resource> has no single identifier returned on GET. Add `name`
   marked IDENTIFIER, or add `<resource>_id` to Create (AIP-133).
2. Repeated field <field> (AIP-144): elements have a lifecycle but no client-settable key. Add a
   key field on the element, or make the field a list of resource names.
3. Inputs not echoed (AIP-203): <fields> are accepted on Create but not returned by GET. Return
   them, or annotate INPUT_ONLY with the reason.
4. Immutable fields (AIP-203): <fields> cannot change after Create. Annotate IMMUTABLE.
5. Server-computed fields (AIP-203): <fields>. Annotate OUTPUT_ONLY.
6. Update (AIP-134): support update_mask "*", or document the updatable field list.
7. Long-running Create (AIP-151): return the resource id in the operation metadata.
8. Delete (AIP-135): NOT_FOUND unless allow_missing; document async completion.
9. Read after Create: state when GET returns the new resource.
10. Launch stage: set `stage` on the resource, methods and new fields.

Items marked STOP block the bundle work; items marked ASK can land after the first PR.
```

## 5. Shapes that look unusual but are fine

- A server-assigned id that is stable and returned on GET. Identity is that id.
- A user-chosen id that is `OUTPUT_ONLY` on the resource and supplied as `<resource>_id` on Create.
- Hierarchical names (`parents/{p}/things/{t}`) as long as `name` alone is what GET accepts.
- Read-only resources (Get and List only). They need no Create or Update rules.
- `IMMUTABLE` deliberately absent on a conditionally immutable field. Check the condition is
  documented instead.
- `OPTIONAL` where you expected `INPUT_ONLY`, as long as the field is echoed.
- A repeated field of plain values or settings with no identity. Positional diffing is correct.
- A whole request-only block such as an initial sub-resource spec that is consumed on Create and
  never echoed, annotated `INPUT_ONLY`.
