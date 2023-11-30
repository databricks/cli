import cowsay

"""
This is the entry point for the wheel file.

The `pyproject.toml` file in the root of this bundle refers to this function
as an entry point in the [tool.poetry.scripts] section.

The key in this section is the name of the command that will be created if the wheel is installed.
The value is the path to the function that is called when the command is run.
The key is referred to in the `python_wheel_task` section in `databricks.yml`.
"""
def hello():
    cowsay.cow("Hello, world!")
