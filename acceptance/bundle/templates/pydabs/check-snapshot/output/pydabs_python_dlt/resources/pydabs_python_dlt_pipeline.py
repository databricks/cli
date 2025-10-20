from databricks.bundles.pipelines import Pipeline

"""
The main pipeline for pydabs_python_dlt
"""

pydabs_python_dlt_pipeline = Pipeline.from_dict(
    {
        "name": "pydabs_python_dlt_pipeline",
        ## Specify the 'catalog' field to configure this pipeline to make use of Unity Catalog:
        # "catalog": "catalog_name",
        "schema": "pydabs_python_dlt_${bundle.target}",
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
