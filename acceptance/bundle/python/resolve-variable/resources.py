from databricks.bundles.core import variables, Bundle, Resources, Variable
from databricks.bundles.jobs import Job, Task, NotebookTask


@variables
class Variables:
    string_variable: Variable[str]
    int_variable: Variable[int]
    nested_variable: Variable[str]
    bool_variable_true: Variable[bool]
    bool_variable_false: Variable[bool]
    list_variable: Variable[list[int]]
    dict_variable: Variable[dict[str, int]]
    complex_variable: Variable[Task]
    complex_list_variable: Variable[list[Task]]


def load_resources(bundle: Bundle) -> Resources:
    string_variable = bundle.resolve_variable(Variables.string_variable)
    assert string_variable == "abc"

    int_variable = bundle.resolve_variable(Variables.int_variable)
    assert int_variable == 42

    nested_variable = bundle.resolve_variable(Variables.nested_variable)
    assert nested_variable == "abc abc"

    bool_variable_true = bundle.resolve_variable(Variables.bool_variable_true)
    assert bool_variable_true

    bool_variable_false = bundle.resolve_variable(Variables.bool_variable_false)
    assert not bool_variable_false

    list_variable = bundle.resolve_variable(Variables.list_variable)
    assert list_variable == [1, 2, 3]

    dict_variable = bundle.resolve_variable(Variables.dict_variable)
    assert dict_variable == {"a": 1, "b": 2}

    complex_variable = bundle.resolve_variable(Variables.complex_variable)
    assert complex_variable == Task(
        task_key="abc",
        notebook_task=NotebookTask(notebook_path="/Workspace/cde"),
    )

    complex_list_variable = bundle.resolve_variable(Variables.complex_list_variable)
    print(complex_list_variable)
    assert complex_list_variable == [
        Task(task_key="abc", notebook_task=NotebookTask(notebook_path="/Workspace/cde")),
        Task(task_key="def", notebook_task=NotebookTask(notebook_path="/Workspace/ghi")),
    ]

    resources = Resources()
    resources.add_job(
        "my_job",
        Job(
            name=Variables.string_variable,
            tasks=Variables.complex_list_variable,
        ),
    )

    return resources
