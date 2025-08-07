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
    "sql_warehouses",
    "secret_scopes",
]

CLI = os.environ["CLI"]
TEMPLATE = os.path.join(os.path.dirname(__file__), "databricks.yml.tmpl")
assert os.path.exists(TEMPLATE), TEMPLATE

for section in SECTIONS:
    os.environ["SECTION"] = section
    os.system(f"envsubst < {TEMPLATE} > databricks.yml")
    print(f"\n=== resources.{section}.rname ===", flush=True)

    ret = os.system(CLI + " bundle validate -o json | jq .resources")
    if ret != 0:
        print(f"Exit code: {ret}", flush=True)

os.unlink("databricks.yml")
