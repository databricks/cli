from enum import Enum
from typing import Literal


class GitProvider(Enum):
    GIT_HUB = "gitHub"
    BITBUCKET_CLOUD = "bitbucketCloud"
    AZURE_DEV_OPS_SERVICES = "azureDevOpsServices"
    GIT_HUB_ENTERPRISE = "gitHubEnterprise"
    BITBUCKET_SERVER = "bitbucketServer"
    GIT_LAB = "gitLab"
    GIT_LAB_ENTERPRISE_EDITION = "gitLabEnterpriseEdition"
    AWS_CODE_COMMIT = "awsCodeCommit"


GitProviderParam = (
    Literal[
        "gitHub",
        "bitbucketCloud",
        "azureDevOpsServices",
        "gitHubEnterprise",
        "bitbucketServer",
        "gitLab",
        "gitLabEnterpriseEdition",
        "awsCodeCommit",
    ]
    | GitProvider
)
