from dataclasses import replace

from codegen.jsonschema import Schema

REMOVED_FIELDS = {
    "jobs.RunJobTask": {
        # all params except job_parameters should be deprecated and should not be supported
        "jar_params",
        "notebook_params",
        "python_params",
        "spark_submit_params",
        "python_named_params",
        "sql_params",
        "dbt_commands",
        # except pipeline_params, that is not deprecated
    },
    "jobs.TriggerSettings": {
        # Old table trigger settings name. Deprecated in favor of `table_update`
        "table",
    },
    "compute.ClusterSpec": {
        # doesn't work, openapi schema needs to be updated to be enum
        "kind",
    },
    "jobs.TaskEmailNotifications": {
        # Deprecated
        "no_alert_for_skipped_runs",
    },
    "jobs.SparkJarTask": {
        # Deprecated. A value of `false` is no longer supported.
        "run_as_repl",
        # Deprecated
        "jar_uri",
    },
    "resources.Pipeline": {
        # Deprecated
        "trigger",
    },
    "pipelines.PipelineLibrary": {
        # Deprecated
        "whl",
    },
}

EXTRA_REQUIRED_FIELDS: dict[str, list[str]] = {
    "jobs.SparkJarTask": ["main_class_name"],
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
