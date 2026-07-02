from dataclasses import replace
from databricks.bundles.pipelines import Pipeline, PipelinePermission
from databricks.bundles.core import pipeline_mutator, Bundle


# Reproduces https://github.com/databricks/cli/issues/5682.
# Adds a permission to a pipeline that is already defined in YAML. Resources updated
# by a PythonMutator go through NormalizeResources, which does NOT run FixPermissions,
# so the deploying user is never added as IS_OWNER. The resulting ownerless permissions
# PUT is rejected by the backend with "The pipeline must have exactly one owner".
@pipeline_mutator
def add_pipeline_permission(bundle: Bundle, pipeline: Pipeline) -> Pipeline:
    permissions = [
        *pipeline.permissions,
        PipelinePermission.from_dict({"group_name": "some-group", "level": "CAN_RUN"}),
    ]
    return replace(pipeline, permissions=permissions)
