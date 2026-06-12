from enum import Enum
from typing import Literal


class Kind(Enum):
    """
    The kind of compute described by this compute specification.

    Depending on `kind`, different validations and default values will be applied.

    Clusters with `kind = CLASSIC_PREVIEW` support the following fields, whereas clusters with no specified `kind` do not.
    * [is_single_node](/api/workspace/clusters/create#is_single_node)
    * [use_ml_runtime](/api/workspace/clusters/create#use_ml_runtime)

    By using the [simple form](https://docs.databricks.com/compute/simple-form.html), your clusters are automatically using `kind = CLASSIC_PREVIEW`.
    """

    CLASSIC_PREVIEW = "CLASSIC_PREVIEW"


KindParam = Literal["CLASSIC_PREVIEW"] | Kind
