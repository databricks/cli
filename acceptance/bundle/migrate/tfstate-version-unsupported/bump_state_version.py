"""Bump the recorded terraform state format version, once armed.

Runs as a post-deploy script. The deploy runs it after applying and before the
migration, so the deploy succeeds against the state it wrote and only the
migration sees a format it does not understand.

Gated on a sentinel so the first deploy leaves a state terraform can still read;
without that, the second deploy would fail on the corrupted state instead.
"""

import pathlib

if not pathlib.Path(".databricks/bump-state-version").exists():
    raise SystemExit(0)

p = pathlib.Path(".databricks/bundle/default/terraform/terraform.tfstate")
p.write_text(p.read_text().replace('"version": 4', '"version": 5'))
