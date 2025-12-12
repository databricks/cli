from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrDict, VariableOrOptional
from databricks.bundles.serving._models.external_model import (
    ExternalModel,
    ExternalModelParam,
)
from databricks.bundles.serving._models.serving_model_workload_type import (
    ServingModelWorkloadType,
    ServingModelWorkloadTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ServedEntityInput:
    """"""

    entity_name: VariableOrOptional[str] = None
    """
    The name of the entity to be served. The entity may be a model in the Databricks Model Registry, a model in the Unity Catalog (UC), or a function of type FEATURE_SPEC in the UC. If it is a UC object, the full name of the object should be given in the form of **catalog_name.schema_name.model_name**.
    """

    entity_version: VariableOrOptional[str] = None

    environment_vars: VariableOrDict[str] = field(default_factory=dict)
    """
    An object containing a set of optional, user-specified environment variable key-value pairs used for serving this entity. Note: this is an experimental feature and subject to change. Example entity environment variables that refer to Databricks secrets: `{"OPENAI_API_KEY": "{{secrets/my_scope/my_key}}", "DATABRICKS_TOKEN": "{{secrets/my_scope2/my_key2}}"}`
    """

    external_model: VariableOrOptional[ExternalModel] = None
    """
    The external model to be served. NOTE: Only one of external_model and (entity_name, entity_version, workload_size, workload_type, and scale_to_zero_enabled) can be specified with the latter set being used for custom model serving for a Databricks registered model. For an existing endpoint with external_model, it cannot be updated to an endpoint without external_model. If the endpoint is created without external_model, users cannot update it to add external_model later. The task type of all external models within an endpoint must be the same.
    """

    instance_profile_arn: VariableOrOptional[str] = None
    """
    ARN of the instance profile that the served entity uses to access AWS resources.
    """

    max_provisioned_concurrency: VariableOrOptional[int] = None
    """
    The maximum provisioned concurrency that the endpoint can scale up to. Do not use if workload_size is specified.
    """

    max_provisioned_throughput: VariableOrOptional[int] = None
    """
    The maximum tokens per second that the endpoint can scale up to.
    """

    min_provisioned_concurrency: VariableOrOptional[int] = None
    """
    The minimum provisioned concurrency that the endpoint can scale down to. Do not use if workload_size is specified.
    """

    min_provisioned_throughput: VariableOrOptional[int] = None
    """
    The minimum tokens per second that the endpoint can scale down to.
    """

    name: VariableOrOptional[str] = None
    """
    The name of a served entity. It must be unique across an endpoint. A served entity name can consist of alphanumeric characters, dashes, and underscores. If not specified for an external model, this field defaults to external_model.name, with '.' and ':' replaced with '-', and if not specified for other entities, it defaults to entity_name-entity_version.
    """

    provisioned_model_units: VariableOrOptional[int] = None
    """
    The number of model units provisioned.
    """

    scale_to_zero_enabled: VariableOrOptional[bool] = None
    """
    Whether the compute resources for the served entity should scale down to zero.
    """

    workload_size: VariableOrOptional[str] = None
    """
    The workload size of the served entity. The workload size corresponds to a range of provisioned concurrency that the compute autoscales between. A single unit of provisioned concurrency can process one request at a time. Valid workload sizes are "Small" (4 - 4 provisioned concurrency), "Medium" (8 - 16 provisioned concurrency), and "Large" (16 - 64 provisioned concurrency). Additional custom workload sizes can also be used when available in the workspace. If scale-to-zero is enabled, the lower bound of the provisioned concurrency for each workload size is 0. Do not use if min_provisioned_concurrency and max_provisioned_concurrency are specified.
    """

    workload_type: VariableOrOptional[ServingModelWorkloadType] = None
    """
    The workload type of the served entity. The workload type selects which type of compute to use in the endpoint. The default value for this parameter is "CPU". For deep learning workloads, GPU acceleration is available by selecting workload types like GPU_SMALL and others. See the available [GPU types](https://docs.databricks.com/en/machine-learning/model-serving/create-manage-serving-endpoints.html#gpu-workload-types).
    """

    @classmethod
    def from_dict(cls, value: "ServedEntityInputDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ServedEntityInputDict":
        return _transform_to_json_value(self)  # type:ignore


class ServedEntityInputDict(TypedDict, total=False):
    """"""

    entity_name: VariableOrOptional[str]
    """
    The name of the entity to be served. The entity may be a model in the Databricks Model Registry, a model in the Unity Catalog (UC), or a function of type FEATURE_SPEC in the UC. If it is a UC object, the full name of the object should be given in the form of **catalog_name.schema_name.model_name**.
    """

    entity_version: VariableOrOptional[str]

    environment_vars: VariableOrDict[str]
    """
    An object containing a set of optional, user-specified environment variable key-value pairs used for serving this entity. Note: this is an experimental feature and subject to change. Example entity environment variables that refer to Databricks secrets: `{"OPENAI_API_KEY": "{{secrets/my_scope/my_key}}", "DATABRICKS_TOKEN": "{{secrets/my_scope2/my_key2}}"}`
    """

    external_model: VariableOrOptional[ExternalModelParam]
    """
    The external model to be served. NOTE: Only one of external_model and (entity_name, entity_version, workload_size, workload_type, and scale_to_zero_enabled) can be specified with the latter set being used for custom model serving for a Databricks registered model. For an existing endpoint with external_model, it cannot be updated to an endpoint without external_model. If the endpoint is created without external_model, users cannot update it to add external_model later. The task type of all external models within an endpoint must be the same.
    """

    instance_profile_arn: VariableOrOptional[str]
    """
    ARN of the instance profile that the served entity uses to access AWS resources.
    """

    max_provisioned_concurrency: VariableOrOptional[int]
    """
    The maximum provisioned concurrency that the endpoint can scale up to. Do not use if workload_size is specified.
    """

    max_provisioned_throughput: VariableOrOptional[int]
    """
    The maximum tokens per second that the endpoint can scale up to.
    """

    min_provisioned_concurrency: VariableOrOptional[int]
    """
    The minimum provisioned concurrency that the endpoint can scale down to. Do not use if workload_size is specified.
    """

    min_provisioned_throughput: VariableOrOptional[int]
    """
    The minimum tokens per second that the endpoint can scale down to.
    """

    name: VariableOrOptional[str]
    """
    The name of a served entity. It must be unique across an endpoint. A served entity name can consist of alphanumeric characters, dashes, and underscores. If not specified for an external model, this field defaults to external_model.name, with '.' and ':' replaced with '-', and if not specified for other entities, it defaults to entity_name-entity_version.
    """

    provisioned_model_units: VariableOrOptional[int]
    """
    The number of model units provisioned.
    """

    scale_to_zero_enabled: VariableOrOptional[bool]
    """
    Whether the compute resources for the served entity should scale down to zero.
    """

    workload_size: VariableOrOptional[str]
    """
    The workload size of the served entity. The workload size corresponds to a range of provisioned concurrency that the compute autoscales between. A single unit of provisioned concurrency can process one request at a time. Valid workload sizes are "Small" (4 - 4 provisioned concurrency), "Medium" (8 - 16 provisioned concurrency), and "Large" (16 - 64 provisioned concurrency). Additional custom workload sizes can also be used when available in the workspace. If scale-to-zero is enabled, the lower bound of the provisioned concurrency for each workload size is 0. Do not use if min_provisioned_concurrency and max_provisioned_concurrency are specified.
    """

    workload_type: VariableOrOptional[ServingModelWorkloadTypeParam]
    """
    The workload type of the served entity. The workload type selects which type of compute to use in the endpoint. The default value for this parameter is "CPU". For deep learning workloads, GPU acceleration is available by selecting workload types like GPU_SMALL and others. See the available [GPU types](https://docs.databricks.com/en/machine-learning/model-serving/create-manage-serving-endpoints.html#gpu-workload-types).
    """


ServedEntityInputParam = ServedEntityInputDict | ServedEntityInput
