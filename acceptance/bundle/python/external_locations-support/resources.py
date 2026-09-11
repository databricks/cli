from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_external_location(
        "my_location_2",
        {
            "name": "my_location_2",
            "url": "s3://test-bucket/path2",
            "credential_name": "test_cred",
        },
    )

    return resources
