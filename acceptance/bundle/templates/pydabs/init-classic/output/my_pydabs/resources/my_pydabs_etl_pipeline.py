from databricks.bundles.pipelines import Pipeline

"""
The main pipeline for my_pydabs
"""

my_pydabs_etl = Pipeline.from_dict(
    {
        "name": "my_pydabs_etl",
        ## Specify the 'catalog' field to configure this pipeline to make use of Unity Catalog:
        # "catalog": "${var.catalog}",
        "schema": "${var.schema}",
        "root_path": "src/my_pydabs_etl",
        "libraries": [
            {
                "glob": {
                    "include": "src/my_pydabs_etl/transformations/**",
                },
            },
        ],
        "environment": {
            "dependencies": [
                # We include every dependency defined by pyproject.toml by defining an editable environment
                # that points to the folder where pyproject.toml is deployed.
                "--editable ${workspace.file_path}",
            ],
        },
    }
)
