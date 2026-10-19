from dataclasses import replace

from databricks.bundles.core import pipeline_mutator
from databricks.bundles.pipelines import Pipeline


@pipeline_mutator
def disable_development(pipeline: Pipeline) -> Pipeline:
    return replace(pipeline, development=False)
