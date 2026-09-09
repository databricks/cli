"""Make the two resolution methods disagree about dst's name.

Runs as a post-deploy script, so the deploy applies against a state it wrote and
only the migration sees the drift. Editing the stored value is the point: a real
deploy stores the same string on both sides of a name-to-name reference, so the
disagreement the conversion warns about cannot be produced by config alone.
"""

import json
import pathlib

p = pathlib.Path(".databricks/bundle/default/terraform/terraform.tfstate")
state = json.loads(p.read_text())

for resource in state["resources"]:
    if resource.get("type") == "databricks_job" and resource.get("name") == "dst":
        resource["instances"][0]["attributes"]["name"] = "source-drifted"

p.write_text(json.dumps(state, indent=2))
