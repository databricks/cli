"""Mock dbutils module for local Databricks notebook execution."""

import os
import sys


class Widgets:
    """Mock for dbutils.widgets."""

    def __init__(self, params=None):
        self._params = params or {}

    def get(self, key, default=""):
        """Get a widget parameter value."""
        return self._params.get(key, default)


class Library:
    """Mock for dbutils.library."""

    def restartPython(self):
        """Mock restart - does nothing in local execution."""


class Notebook:
    """Mock for dbutils.notebook."""

    def exit(self, value):
        """Stop the notebook with value as its result, which jobs/runs/get-output returns as notebook_output.result."""
        with open(os.environ["TESTSERVER_NOTEBOOK_EXIT_FILE"], "w", encoding="utf-8", newline="") as f:
            f.write(str(value))
        sys.exit(0)


class DBUtils:
    """Mock Databricks utilities (dbutils) for local notebook execution."""

    def __init__(self, params=None):
        self.widgets = Widgets(params)
        self.library = Library()
        self.notebook = Notebook()


# Global dbutils instance - will be initialized with parameters
dbutils = None
