import os
import sys

# On windows, shutil.which or os.system cannot find "envsubst.py" even though
# it is available in the path. In order to work around this we directly
# import the envsubst module from the bin directory.
sys.path.append(os.path.join(os.path.dirname(__file__), "../../../bin"))
import envsubst

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

    # Read the template file and substitute the variables.
    with open(TEMPLATE, "r") as f:
        template = f.read()
    with open("databricks.yml", "w") as f:
        f.write(envsubst.substitute_variables(template))

    print(f"\n=== resources.{section}.rname ===", flush=True)

    ret = os.system(CLI + " bundle validate -o json | jq .resources")
    if ret != 0:
        print(f"Exit code: {ret}", flush=True)

os.unlink("databricks.yml")
