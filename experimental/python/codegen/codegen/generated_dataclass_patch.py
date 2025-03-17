from dataclasses import replace

from codegen.generated_dataclass import (
    GeneratedDataclass,
    GeneratedField,
    GeneratedOneOf,
    GeneratedType,
)


def reorder_required_fields(models: dict[str, GeneratedDataclass]):
    """
    Reorder fields in dataclasses so that required fields come first.
    It's necessary for kwargs in the constructor to work correctly.
    """
    for name, model in models.items():
        if not model.fields:
            continue

        required_fields = [field for field in model.fields if _is_required(field)]
        optional_fields = [field for field in model.fields if not _is_required(field)]

        models[name] = replace(model, fields=required_fields + optional_fields)


def quote_recursive_references(models: dict[str, GeneratedDataclass]):
    """
    If there is a cycle between two dataclasses, we need to quote one of them.

    Example:

        class Foo:
            bar: Optional[Bar]

        class Bar:
            foo: "Foo"
    """

    # see also _append_resolve_recursive_imports

    models["jobs.ForEachTask"] = _quote_recursive_references_for_model(
        models["jobs.ForEachTask"],
        references={"Task", "TaskParam"},
    )


def _quote_recursive_references_for_model(
    model: GeneratedDataclass,
    references: set[str],
) -> GeneratedDataclass:
    def update_type_name(type_name: GeneratedType):
        if type_name.name in references:
            return replace(
                type_name,
                name=f'"{type_name.name}"',
            )
        elif type_name.parameters:
            return replace(
                type_name,
                parameters=[update_type_name(param) for param in type_name.parameters],
            )
        else:
            return type_name

    def update_field(field: GeneratedField):
        return replace(
            field,
            type_name=update_type_name(field.type_name),
            param_type_name=update_type_name(field.param_type_name),
        )

    return replace(
        model,
        fields=[update_field(field) for field in model.fields],
    )


def _is_required(field: GeneratedField) -> bool:
    return field.default is None and field.default_factory is None


def add_default_values(models: dict[str, GeneratedDataclass]):
    models["jobs.CronSchedule"] = _add_default_value(
        models["jobs.CronSchedule"],
        field_name="timezone_id",
        default_value='"UTC"',
    )


def add_oneofs(models: dict[str, GeneratedDataclass]):
    models["jobs.JobRunAs"] = _add_oneof(
        models["jobs.JobRunAs"],
        ["user_name", "service_principal_name"],
        required=True,
    )

    models["resources.Permission"] = _add_oneof(
        models["resources.Permission"],
        ["user_name", "service_principal_name", "group_name"],
        required=True,
    )

    models["jobs.Task"] = _add_oneof(
        models["jobs.Task"],
        ["new_cluster", "job_cluster_key", "environment_key", "existing_cluster_id"],
        required=False,
    )

    models["jobs.TriggerSettings"] = _add_oneof(
        models["jobs.TriggerSettings"],
        ["file_arrival", "periodic", "table_update"],
        required=True,
    )


def _add_default_value(
    model: GeneratedDataclass,
    field_name: str,
    default_value: str,
):
    """
    Add a default value for a field in a dataclass.
    """

    def update_field(field: GeneratedField):
        if field.field_name == field_name:
            return replace(
                field,
                default=default_value,
                create_func_default=default_value,
            )
        else:
            return field

    return replace(
        model,
        fields=[update_field(field) for field in model.fields],
    )


def _add_oneof(
    model: GeneratedDataclass,
    values: list[str],
    required: bool,
):
    """
    Make a field in a dataclass a one-of, one of the fields can be present.
    """

    return replace(
        model,
        one_ofs=[GeneratedOneOf(values, required=required)],
    )
