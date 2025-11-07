from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.jobs._models.git_provider import GitProvider, GitProviderParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GitSource:
    """
    An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.

    If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.

    Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
    """

    git_provider: VariableOr[GitProvider]
    """
    Unique identifier of the service used to host the Git repository. The value is case insensitive.
    """

    git_url: VariableOr[str]
    """
    URL of the repository to be cloned by this job.
    """

    git_branch: VariableOrOptional[str] = None
    """
    Name of the branch to be checked out and used by this job. This field cannot be specified in conjunction with git_tag or git_commit.
    """

    git_commit: VariableOrOptional[str] = None
    """
    Commit to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_tag.
    """

    git_tag: VariableOrOptional[str] = None
    """
    Name of the tag to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_commit.
    """

    @classmethod
    def from_dict(cls, value: "GitSourceDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GitSourceDict":
        return _transform_to_json_value(self)  # type:ignore


class GitSourceDict(TypedDict, total=False):
    """"""

    git_provider: VariableOr[GitProviderParam]
    """
    Unique identifier of the service used to host the Git repository. The value is case insensitive.
    """

    git_url: VariableOr[str]
    """
    URL of the repository to be cloned by this job.
    """

    git_branch: VariableOrOptional[str]
    """
    Name of the branch to be checked out and used by this job. This field cannot be specified in conjunction with git_tag or git_commit.
    """

    git_commit: VariableOrOptional[str]
    """
    Commit to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_tag.
    """

    git_tag: VariableOrOptional[str]
    """
    Name of the tag to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_commit.
    """


GitSourceParam = GitSourceDict | GitSource
