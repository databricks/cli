from databricks.bundles.core import Resources


def load_resources() -> Resources:
    resources = Resources()

    resources.add_genie_agent(
        "my_genie_agent_2",
        {
            "display_name": "my_genie_agent_2",
            "warehouse_id": "abc123",
            "definition": {
                "config": {
                    "sample_questions": [
                        {
                            "id": "01f0a1b2c3d4e5f60718293a4b5c6d7e",
                            "question": "What was the average trip distance last month?",
                        }
                    ]
                },
                "instructions": {
                    "text_instruction": {
                        "id": "01f0a1b2c3d4e5f60718293a4b5c6d70",
                        "content": "Report average trip distance.",
                    },
                },
                "resources": {
                    "tables": [{"identifier": "samples.nyctaxi.trips"}],
                },
            },
        },
    )

    return resources
