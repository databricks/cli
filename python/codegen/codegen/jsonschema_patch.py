from dataclasses import replace

from codegen.jsonschema import Schema

REMOVED_FIELDS = {
    # fields that were deprecated a long time ago
    "resources.Pipeline": {
        # 'trigger' is deprecated, use 'continuous' or schedule pipeline refresh using job instead
        "trigger",
    },
    "pipelines.PipelineLibrary": [
        # 'whl' is deprecated, install libraries through notebooks and %pip command
        "whl",
    ],
}

EXTRA_REQUIRED_FIELDS: dict[str, list[str]] = {
    "jobs.SparkJarTask": ["main_class_name"],
}

# Burn-down list of upstream API descriptions that aren't valid reStructuredText
# and break the Sphinx docs build. Each entry is a temporary override until the
# proto comment is fixed upstream; remove it once the fix lands (the no-op guard
# in override_descriptions flags entries that upstream has already fixed).
#
# sql.SpotInstancePolicy: the upstream comment is a hard-wrapped ASCII grid table
# that docutils rejects as malformed. Rewritten as a list-table.
# See sqlgateway/scheduler/api/proto/endpoint_common.proto.
DESCRIPTIONS: dict[str, str] = {
    "sql.SpotInstancePolicy": (
        "EndpointSpotInstancePolicy configures whether the endpoint should use spot instances.\n"
        "\n"
        "The breakdown of how the EndpointSpotInstancePolicy converts to per cloud configurations is:\n"
        "\n"
        ".. list-table::\n"
        "   :header-rows: 1\n"
        "\n"
        "   * - Cloud\n"
        "     - COST_OPTIMIZED\n"
        "     - RELIABILITY_OPTIMIZED\n"
        "   * - AWS\n"
        "     - On Demand Driver with Spot Executors\n"
        "     - On Demand Driver and Executors\n"
        "   * - AZURE\n"
        "     - On Demand Driver and Executors\n"
        "     - On Demand Driver and Executors\n"
    ),
}

# Field-level counterpart to DESCRIPTIONS: individual upstream property
# descriptions that aren't valid reStructuredText and break the Sphinx docs
# build. Keyed by schema name, then field name. Same burn-down rules apply --
# each entry is a temporary override until the proto comment is fixed upstream,
# and override_field_descriptions flags entries that upstream has already fixed.
#
# jobs.DeploymentSpec.command_path: the upstream comment embeds a Markdown
# ```bash code fence, which docutils parses as an unterminated inline literal.
# Rewritten as an RST literal block.
#
# compute.InstancePoolGcpAttributes.gcp_availability: the upstream comment's
# final bullet wraps onto an unindented continuation line, which docutils
# rejects as an unexpected unindent. Rewritten with the continuation aligned to
# the bullet text.
FIELD_DESCRIPTIONS: dict[str, dict[str, str]] = {
    "jobs.DeploymentSpec": {
        "command_path": (
            "[Public Preview] Workspace path of the script to run on each node in this deployment.\n"
            "Upload the script to this path and supply the path here. When the task\n"
            "runs, the file at this path is run on each node; if it fails, the task\n"
            "fails with its exit code.\n"
            "\n"
            "Example script contents::\n"
            "\n"
            "    # Plain Python:\n"
            "    python train.py --epochs 10\n"
            "\n"
            "    # Multi-GPU via accelerate:\n"
            "    accelerate launch train.py --config config.yaml\n"
            "\n"
            "    # Distributed via torchrun:\n"
            "    torchrun --nproc_per_node=8 train.py"
        ),
    },
    "compute.InstancePoolGcpAttributes": {
        "gcp_availability": (
            "Availability type for the instances in the pool. One of:\n"
            "\n"
            "- `ON_DEMAND_GCP`: the pool uses on-demand instances only.\n"
            "- `PREEMPTIBLE_GCP`: the pool uses preemptible instances only.\n"
            "- `PREEMPTIBLE_WITH_FALLBACK_GCP`: the pool acquires preemptible instances first, and falls\n"
            "  back to on-demand instances when preemptible capacity is unavailable."
        ),
    },
}


def add_extra_required_fields(schemas: dict[str, Schema]):
    output = {}

    for name, schema in schemas.items():
        if extra_required := EXTRA_REQUIRED_FIELDS.get(name):
            new_required = [*schema.required, *extra_required]
            new_required = list(set(new_required))

            if set(new_required) == set(schema.required):
                raise ValueError(
                    f"Extra required fields for {name} are already present in the schema"
                )

            new_schema = replace(schema, required=new_required)

            output[name] = new_schema
        else:
            output[name] = schema

    return output


def override_descriptions(schemas: dict[str, Schema]):
    if missing := DESCRIPTIONS.keys() - schemas.keys():
        raise ValueError(f"Cannot override description for unknown schemas: {missing}")

    output = {}
    for name, schema in schemas.items():
        if override := DESCRIPTIONS.get(name):
            if schema.description == override:
                raise ValueError(
                    f"Description override for {name} is a no-op; the upstream "
                    "description was fixed, so remove the override"
                )
            output[name] = replace(schema, description=override)
        else:
            output[name] = schema

    return output


def override_field_descriptions(schemas: dict[str, Schema]):
    if missing := FIELD_DESCRIPTIONS.keys() - schemas.keys():
        raise ValueError(
            f"Cannot override field descriptions for unknown schemas: {missing}"
        )

    output = {}
    for name, schema in schemas.items():
        if field_overrides := FIELD_DESCRIPTIONS.get(name):
            if unknown := field_overrides.keys() - schema.properties.keys():
                raise ValueError(
                    f"Cannot override unknown fields {unknown} in schema {name}"
                )

            new_properties = dict(schema.properties)
            for field_name, override in field_overrides.items():
                prop = new_properties[field_name]
                if prop.description == override:
                    raise ValueError(
                        f"Field description override for {name}.{field_name} is a "
                        "no-op; the upstream description was fixed, so remove the "
                        "override"
                    )
                new_properties[field_name] = replace(prop, description=override)

            output[name] = replace(schema, properties=new_properties)
        else:
            output[name] = schema

    return output


def remove_unsupported_fields(schemas: dict[str, Schema]):
    output = {}

    for name, schema in schemas.items():
        if removed_fields := REMOVED_FIELDS.get(name):
            new_properties = {
                field: prop
                for field, prop in schema.properties.items()
                if field not in removed_fields
            }

            if new_properties.keys() == schema.properties.keys():
                raise ValueError(f"No fields to remove in schema {name}")

            new_schema = replace(schema, properties=new_properties)

            output[name] = new_schema
        else:
            output[name] = schema

    return output
