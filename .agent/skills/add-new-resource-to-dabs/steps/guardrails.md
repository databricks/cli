# Guardrails

Checklist:
- If deleting a resource is non-ephemeral/high-impact, require warning + confirmation in deploy flows.
- If recreate-on-change can cause downtime or costly rebuilds, require warning + confirmation.
- Implement guardrail behavior in `bundle/phases/deploy.go` and related deployment checks.

Canonical guardrail intent for deployment-facing behavior lives in:
- `acceptance/bundle/deployment/README.md`
