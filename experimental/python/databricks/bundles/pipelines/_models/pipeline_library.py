from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.maven_library import (
    MavenLibrary,
    MavenLibraryParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.file_library import (
    FileLibrary,
    FileLibraryParam,
)
from databricks.bundles.pipelines._models.notebook_library import (
    NotebookLibrary,
    NotebookLibraryParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineLibrary:
    """"""

    file: VariableOrOptional[FileLibrary] = None
    """
    The path to a file that defines a pipeline and is stored in the Databricks Repos.
    """

    jar: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    URI of the jar to be installed. Currently only DBFS is supported.
    """

    maven: VariableOrOptional[MavenLibrary] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Specification of a maven library to be installed.
    """

    notebook: VariableOrOptional[NotebookLibrary] = None
    """
    The path to a notebook that defines a pipeline and is stored in the Databricks workspace.
    """

    @classmethod
    def from_dict(cls, value: "PipelineLibraryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineLibraryDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineLibraryDict(TypedDict, total=False):
    """"""

    file: VariableOrOptional[FileLibraryParam]
    """
    The path to a file that defines a pipeline and is stored in the Databricks Repos.
    """

    jar: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    URI of the jar to be installed. Currently only DBFS is supported.
    """

    maven: VariableOrOptional[MavenLibraryParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Specification of a maven library to be installed.
    """

    notebook: VariableOrOptional[NotebookLibraryParam]
    """
    The path to a notebook that defines a pipeline and is stored in the Databricks workspace.
    """


PipelineLibraryParam = PipelineLibraryDict | PipelineLibrary
