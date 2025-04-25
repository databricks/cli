import os

SECTIONS = [
    "jobs",
    "pipelines",
    "models",
    "experiments",
    "registered_models",
    "quality_monitors",
    "schemas",
    "volumes",
    "clusters",
    "dashboards",
    "apps",
]

CLI = os.environ["CLI"]

for section in SECTIONS:
    for ind, content in enumerate(["{}", "", "null"]):
        config = f"""
bundle:
  name: BUNDLE_{section}_{ind}

resources:
  {section}:
    rname: {content}
"""
        print(f"\n=== resources.{section}.rname: {content} ===", flush=True)

        with open("databricks.yml", "w") as fobj:
            fobj.write(config)

        ret = os.system(CLI + " bundle validate -o json | jq .resources")
        if ret != 0:
            print(f"Exit code: {ret}", flush=True)

os.unlink("databricks.yml")
