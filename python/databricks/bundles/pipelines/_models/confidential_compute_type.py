from enum import Enum
from typing import Literal


class ConfidentialComputeType(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Confidential computing technology for GCP instances.
    Aligns with gcloud's --confidential-compute-type flag and the REST API's
    confidentialInstanceConfig.confidentialInstanceType field.
    See: https://cloud.google.com/confidential-computing/confidential-vm/docs/create-a-confidential-vm-instance
    """

    CONFIDENTIAL_COMPUTE_TYPE_NONE = "CONFIDENTIAL_COMPUTE_TYPE_NONE"
    SEV_SNP = "SEV_SNP"


ConfidentialComputeTypeParam = (
    Literal["CONFIDENTIAL_COMPUTE_TYPE_NONE", "SEV_SNP"] | ConfidentialComputeType
)
