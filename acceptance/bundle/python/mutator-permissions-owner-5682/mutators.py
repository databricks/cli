from dataclasses import replace
from databricks.bundles.pipelines import Pipeline, PipelinePermission
from databricks.bundles.core import pipeline_mutator, Bundle


# Regression test for https://github.com/databricks/cli/issues/5682.
# Adds a permission to a pipeline that is already defined in YAML (mirrors the
# reporter's mutator, which reads a bundle variable). Resources updated by a
# PythonMutator go through NormalizeResources; that path now runs FixPermissions,
# so the deploying user is added as IS_OWNER and the permissions PUT succeeds.
# Before the fix the ACL had no owner and the backend rejected the PUT with
# "The pipeline must have exactly one owner".
@pipeline_mutator
def add_pipeline_permission(bundle: Bundle, pipeline: Pipeline) -> Pipeline:
    group = bundle.resolve_variable(bundle.variables["grantee_group"])
    permissions = [
        *pipeline.permissions,
        PipelinePermission.from_dict({"group_name": group, "level": "CAN_RUN"}),
    ]
    return replace(pipeline, permissions=permissions)
