from typing import Any, Dict, List, TypedDict, Union

from databricks.bundles.core._variable import VariableOr, VariableOrOptional


class AppEnvVarDict(TypedDict, total=False):
    """Environment variable configuration for an app"""

    name: VariableOr[str]
    """The name of the environment variable"""

    value: VariableOrOptional[str]
    """
    The value of the environment variable.
    Either value or valueFrom must be specified, but not both.
    """

    valueFrom: VariableOrOptional[str]
    """
    Reference to another environment variable to get the value from.
    Either value or valueFrom must be specified, but not both.
    """


class AppConfigDict(TypedDict, total=False):
    """
    Configuration for a Databricks app.

    This is a flexible dictionary structure that can contain various app-specific
    configuration settings. Common configuration options include:

    - command: List of strings for the command to run the app
    - env: List of environment variable configurations
    - Any other app-specific settings
    """

    command: Union[List[str], VariableOr[str]]
    """
    The command to run the app. This is typically a list of strings
    representing the executable and its arguments.
    Example: ["python", "app.py"] or ["streamlit", "run", "main.py"]
    """

    env: List[AppEnvVarDict]
    """
    Environment variables to set for the app.
    Each variable can have a direct value or reference another environment variable.
    Example: [{"name": "PORT", "value": "8080"}, {"name": "DB_URL", "valueFrom": "DATABASE_URL"}]
    """


# AppConfigParam is a flexible type that accepts various config formats
AppConfigParam = Union[AppConfigDict, Dict[str, Any]]
