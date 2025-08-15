from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class MavenLibrary:
    """"""

    coordinates: VariableOr[str]
    """
    Gradle-style maven coordinates. For example: "org.jsoup:jsoup:1.7.2".
    """

    exclusions: VariableOrList[str] = field(default_factory=list)
    """
    List of dependences to exclude. For example: `["slf4j:slf4j", "*:hadoop-client"]`.
    
    Maven dependency exclusions:
    https://maven.apache.org/guides/introduction/introduction-to-optional-and-excludes-dependencies.html.
    """

    repo: VariableOrOptional[str] = None
    """
    Maven repo to install the Maven package from. If omitted, both Maven Central Repository
    and Spark Packages are searched.
    """

    @classmethod
    def from_dict(cls, value: "MavenLibraryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "MavenLibraryDict":
        return _transform_to_json_value(self)  # type:ignore


class MavenLibraryDict(TypedDict, total=False):
    """"""

    coordinates: VariableOr[str]
    """
    Gradle-style maven coordinates. For example: "org.jsoup:jsoup:1.7.2".
    """

    exclusions: VariableOrList[str]
    """
    List of dependences to exclude. For example: `["slf4j:slf4j", "*:hadoop-client"]`.
    
    Maven dependency exclusions:
    https://maven.apache.org/guides/introduction/introduction-to-optional-and-excludes-dependencies.html.
    """

    repo: VariableOrOptional[str]
    """
    Maven repo to install the Maven package from. If omitted, both Maven Central Repository
    and Spark Packages are searched.
    """


MavenLibraryParam = MavenLibraryDict | MavenLibrary
