---
name: Add a new resource to DABs
description: Use this skill if tasked with adding a new resource
---

## Rules for adding a new resource

ALL FILE PATHS AND COMMANDS ARE REFERENCED FROM/TO BE EXECUTED FROM THE `databricks/cli` REPO ROOT.

DO NOT EDIT AUTO-GENERATED FILES! RUN THE APPROPRIATE `make` COMMAND INSTEAD!

README files in repo directories are the source of truth for contributor-facing behavior and long-lived implementation policy.
This skill keeps only the execution order and concise checklists.

## Steps

1. [test-server.md](./steps/test-server.md)
2. [auto-generation.md](./steps/auto-generation.md)
3. [terraform.md](./steps/terraform.md)
4. [direct.md](./steps/direct.md)
5. [acceptance-tests.md](./steps/acceptance-tests.md)
6. [guardrails.md](./steps/guardrails.md)

## Index of other resources

- [acceptance/README.md](../../../acceptance/README.md)
- [acceptance/bundle/deployment/README.md](../../../acceptance/bundle/deployment/README.md)
- [acceptance/bundle/invariant/README.md](../../../acceptance/bundle/invariant/README.md)
- [bundle/tests/README.md](../../../bundle/tests/README.md)
- [bundle/direct/dresources/README.md](../../../bundle/direct/dresources/README.md)

# GENERAL RULES

- Use the `ForceSendFields: utils.FilterFields` pattern
