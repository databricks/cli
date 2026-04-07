# Direct Deployment

Checklist:
- Register the resource in `bundle/direct/dresources/all.go`.
- Implement the adapter in one resource-focused file under `bundle/direct/dresources/`.
- Wire field behavior in `bundle/direct/dresources/resources.yml` when needed.
- Leave `DoUpdate` unimplemented when the API has no update method.

Canonical behavior for adapter contracts, field semantics, and testing expectations lives in:
- `bundle/direct/dresources/README.md`
