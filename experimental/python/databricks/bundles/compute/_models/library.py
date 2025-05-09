from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.maven_library import (
    MavenLibrary,
    MavenLibraryParam,
)
from databricks.bundles.compute._models.python_py_pi_library import (
    PythonPyPiLibrary,
    PythonPyPiLibraryParam,
)
from databricks.bundles.compute._models.r_cran_library import (
    RCranLibrary,
    RCranLibraryParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Library:
    """"""

    cran: VariableOrOptional[RCranLibrary] = None
    """
    Specification of a CRAN library to be installed as part of the library
    """

    jar: VariableOrOptional[str] = None
    """
    URI of the JAR library to install. Supported URIs include Workspace paths, Unity Catalog Volumes paths, and S3 URIs.
    For example: `{ "jar": "/Workspace/path/to/library.jar" }`, `{ "jar" : "/Volumes/path/to/library.jar" }` or
    `{ "jar": "s3://my-bucket/library.jar" }`.
    If S3 is used, please make sure the cluster has read access on the library. You may need to
    launch the cluster with an IAM role to access the S3 URI.
    """

    maven: VariableOrOptional[MavenLibrary] = None
    """
    Specification of a maven library to be installed. For example:
    `{ "coordinates": "org.jsoup:jsoup:1.7.2" }`
    """

    pypi: VariableOrOptional[PythonPyPiLibrary] = None
    """
    Specification of a PyPi library to be installed. For example:
    `{ "package": "simplejson" }`
    """

    requirements: VariableOrOptional[str] = None
    """
    URI of the requirements.txt file to install. Only Workspace paths and Unity Catalog Volumes paths are supported.
    For example: `{ "requirements": "/Workspace/path/to/requirements.txt" }` or `{ "requirements" : "/Volumes/path/to/requirements.txt" }`
    """

    whl: VariableOrOptional[str] = None
    """
    URI of the wheel library to install. Supported URIs include Workspace paths, Unity Catalog Volumes paths, and S3 URIs.
    For example: `{ "whl": "/Workspace/path/to/library.whl" }`, `{ "whl" : "/Volumes/path/to/library.whl" }` or
    `{ "whl": "s3://my-bucket/library.whl" }`.
    If S3 is used, please make sure the cluster has read access on the library. You may need to
    launch the cluster with an IAM role to access the S3 URI.
    """

    @classmethod
    def from_dict(cls, value: "LibraryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "LibraryDict":
        return _transform_to_json_value(self)  # type:ignore


class LibraryDict(TypedDict, total=False):
    """"""

    cran: VariableOrOptional[RCranLibraryParam]
    """
    Specification of a CRAN library to be installed as part of the library
    """

    jar: VariableOrOptional[str]
    """
    URI of the JAR library to install. Supported URIs include Workspace paths, Unity Catalog Volumes paths, and S3 URIs.
    For example: `{ "jar": "/Workspace/path/to/library.jar" }`, `{ "jar" : "/Volumes/path/to/library.jar" }` or
    `{ "jar": "s3://my-bucket/library.jar" }`.
    If S3 is used, please make sure the cluster has read access on the library. You may need to
    launch the cluster with an IAM role to access the S3 URI.
    """

    maven: VariableOrOptional[MavenLibraryParam]
    """
    Specification of a maven library to be installed. For example:
    `{ "coordinates": "org.jsoup:jsoup:1.7.2" }`
    """

    pypi: VariableOrOptional[PythonPyPiLibraryParam]
    """
    Specification of a PyPi library to be installed. For example:
    `{ "package": "simplejson" }`
    """

    requirements: VariableOrOptional[str]
    """
    URI of the requirements.txt file to install. Only Workspace paths and Unity Catalog Volumes paths are supported.
    For example: `{ "requirements": "/Workspace/path/to/requirements.txt" }` or `{ "requirements" : "/Volumes/path/to/requirements.txt" }`
    """

    whl: VariableOrOptional[str]
    """
    URI of the wheel library to install. Supported URIs include Workspace paths, Unity Catalog Volumes paths, and S3 URIs.
    For example: `{ "whl": "/Workspace/path/to/library.whl" }`, `{ "whl" : "/Volumes/path/to/library.whl" }` or
    `{ "whl": "s3://my-bucket/library.whl" }`.
    If S3 is used, please make sure the cluster has read access on the library. You may need to
    launch the cluster with an IAM role to access the S3 URI.
    """


LibraryParam = LibraryDict | Library
