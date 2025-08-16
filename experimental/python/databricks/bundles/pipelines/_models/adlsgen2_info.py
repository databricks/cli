from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Adlsgen2Info:
    """
    A storage location in Adls Gen2
    """

    destination: VariableOr[str]
    """
    abfss destination, e.g. `abfss://<container-name>@<storage-account-name>.dfs.core.windows.net/<directory-name>`.
    """

    @classmethod
    def from_dict(cls, value: "Adlsgen2InfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "Adlsgen2InfoDict":
        return _transform_to_json_value(self)  # type:ignore


class Adlsgen2InfoDict(TypedDict, total=False):
    """"""

    destination: VariableOr[str]
    """
    abfss destination, e.g. `abfss://<container-name>@<storage-account-name>.dfs.core.windows.net/<directory-name>`.
    """


Adlsgen2InfoParam = Adlsgen2InfoDict | Adlsgen2Info
