from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_domain(
        "my_domain_2",
        {
            "tag_key": "Payments/Fraud",
            "subtitle": "Fraud detection",
            "parent_domain_id": "${resources.domains.my_domain_1.id}",
            "icon": {"name": "SHIELD_CHECKERED", "color": "#C83243"},
            "draft": True,
        },
    )

    return resources
