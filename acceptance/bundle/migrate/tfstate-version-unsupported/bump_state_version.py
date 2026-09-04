"""Bump the recorded terraform state format version.

Runs as a post-deploy script. The deploy runs it after applying and before the
migration, so the deploy succeeds against the state it wrote and only the
migration sees a format it does not understand.
"""

import pathlib

p = pathlib.Path(".databricks/bundle/default/terraform/terraform.tfstate")
p.write_text(p.read_text().replace('"version": 4', '"version": 5'))
