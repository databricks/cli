from databricks.bundles.pipelines import Pipeline

"""
The main pipeline for pydabs_python_dlt_serverless
"""

pydabs_python_dlt_serverless_pipeline = Pipeline.from_dict(
    {
        "name": "pydabs_python_dlt_serverless_pipeline",
        ## Catalog is required for serverless compute
        "catalog": "main",
        "schema": "pydabs_python_dlt_serverless_${bundle.target}",
        "serverless": True,
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
