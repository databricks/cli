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

# Burn-down lists of upstream descriptions that aren't valid
# reStructuredText and break the Sphinx docs build. We can't render them and don't
# want to maintain a hand-copied RST version, so drop the description entirely
# until the proto comment is fixed upstream. Field descriptions are keyed by schema
# name, then the field names whose descriptions to drop. drop_descriptions flags
# entries that upstream has already fixed (the description is now empty).
#
# jobs.DeploymentSpec.command_path: embeds a Markdown ```bash code fence, which
# docutils parses as an unterminated inline literal.
# compute.InstancePoolGcpAttributes.gcp_availability: the final bullet wraps onto
# an unindented continuation line, which docutils rejects as an unexpected unindent.
DROP_FIELD_DESCRIPTIONS: dict[str, list[str]] = {
    "jobs.DeploymentSpec": ["command_path"],
    "compute.InstancePoolGcpAttributes": ["gcp_availability"],
}

DROP_SCHEMA_DESCRIPTIONS: set[str] = {"jobs.SqlConditionTriggerMode"}


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


def drop_descriptions(schemas: dict[str, Schema]):
    if (
        missing := (DROP_FIELD_DESCRIPTIONS.keys() | DROP_SCHEMA_DESCRIPTIONS)
        - schemas.keys()
    ):
        raise ValueError(f"Cannot drop descriptions for unknown schemas: {missing}")

    output = {}
    for name, schema in schemas.items():
        if name in DROP_SCHEMA_DESCRIPTIONS:
            if not schema.description:
                raise ValueError(
                    f"Schema description drop for {name} is a no-op; "
                    "the upstream description is already empty, so remove the entry"
                )
            schema = replace(schema, description=None)

        if drop_fields := DROP_FIELD_DESCRIPTIONS.get(name):
            if unknown := set(drop_fields) - schema.properties.keys():
                raise ValueError(
                    f"Cannot drop description for unknown fields {unknown} in schema {name}"
                )

            new_properties = dict(schema.properties)
            for field_name in drop_fields:
                prop = new_properties[field_name]
                if not prop.description:
                    raise ValueError(
                        f"Field description drop for {name}.{field_name} is a no-op; "
                        "the upstream description is already empty, so remove the entry"
                    )
                new_properties[field_name] = replace(prop, description=None)

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
