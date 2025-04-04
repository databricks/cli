from dataclasses import replace

from databricks.bundles.core import pipeline_mutator
from databricks.bundles.pipelines import Pipeline


@pipeline_mutator
def update_pipeline(pipeline: Pipeline) -> Pipeline:
    assert isinstance(pipeline.name, str)

    return replace(pipeline, name=f"{pipeline.name} (updated)")
