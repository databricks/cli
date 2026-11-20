from databricks.bundles.core import Resources


# The pipeline is defined in Python and refers to a schema defined in YAML, so only the final
# config, after the Python mutators have run, shows that the schema has grants.
def load_resources() -> Resources:
    resources = Resources()
    resources.add_pipeline(
        resource_name="load",
        pipeline={
            "name": "python-pipeline-grants-load",
            "catalog": "${resources.schemas.src.catalog_name}",
            "schema": "${resources.schemas.src.name}",
            "serverless": True,
            "libraries": [{"file": {"path": "./transformation.py"}}],
            "run_as": {"service_principal_name": "${var.run_as_sp}"},
        },
    )
    return resources
