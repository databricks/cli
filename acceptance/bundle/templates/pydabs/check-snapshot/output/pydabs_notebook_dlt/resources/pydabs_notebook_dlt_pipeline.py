from databricks.bundles.pipelines import Pipeline

"""
The main pipeline for pydabs_notebook_dlt
"""

pydabs_notebook_dlt_pipeline = Pipeline.from_dict(
    {
        "name": "pydabs_notebook_dlt_pipeline",
        ## Specify the 'catalog' field to configure this pipeline to make use of Unity Catalog:
        # "catalog": "catalog_name",
        "schema": "pydabs_notebook_dlt_${bundle.target}",
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
