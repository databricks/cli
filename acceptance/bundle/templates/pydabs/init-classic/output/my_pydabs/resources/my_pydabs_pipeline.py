from databricks.bundles.pipelines import Pipeline

"""
The main pipeline for my_pydabs
"""

my_pydabs_pipeline = Pipeline.from_dict(
    {
        "name": "my_pydabs_pipeline",
        ## Specify the 'catalog' field to configure this pipeline to make use of Unity Catalog:
        # "catalog": "catalog_name",
        "schema": "my_pydabs_${bundle.target}",
        "libraries": [
            {
                "notebook": {
                    "path": "src/pipeline.ipynb",
                },
            },
        ],
        "configuration": {
            "bundle.sourcePath": "${workspace.file_path}/src",
        },
    }
)
