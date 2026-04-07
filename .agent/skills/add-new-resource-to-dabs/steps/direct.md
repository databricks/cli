# Direct Deployment

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
