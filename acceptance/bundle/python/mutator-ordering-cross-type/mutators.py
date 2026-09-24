from dataclasses import replace

from databricks.bundles.core import job_mutator, pipeline_mutator
from databricks.bundles.jobs import Job
from databricks.bundles.pipelines import Pipeline

ORDER = []


@pipeline_mutator
def record_pipeline(pipeline: Pipeline) -> Pipeline:
    ORDER.append("pipeline")
    return pipeline


@job_mutator
def record_job(job: Job) -> Job:
    ORDER.append("job")
    return job


@job_mutator
def write_order(job: Job) -> Job:
    return replace(job, description=",".join(ORDER))
