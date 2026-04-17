from databricks.bundles.pipelines import Pipeline

my_jobs_as_code_pipeline = Pipeline.from_dict(
    {
        "name": "my_jobs_as_code_pipeline",
        "target": "my_jobs_as_code_${bundle.target}",
        ## Specify the 'catalog' field to configure this pipeline to make use of Unity Catalog:
        "catalog": "catalog_name",
        "libraries": [
            {
                "notebook": {
                    "path": "src/dlt_pipeline.ipynb",
                },
            },
        ],
        "configuration": {
            "bundle.sourcePath": "${workspace.file_path}/src",
        },
    }
)
