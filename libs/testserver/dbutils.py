"""Mock dbutils module for local Databricks notebook execution."""


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
        pass


class Notebook:
    """Mock for dbutils.notebook."""

    def exit(self, value):
        """Exit notebook and return value - converted to print for local execution."""
        print(value)


class DBUtils:
    """Mock Databricks utilities (dbutils) for local notebook execution."""

    def __init__(self, params=None):
        self.widgets = Widgets(params)
        self.library = Library()
        self.notebook = Notebook()


# Global dbutils instance - will be initialized with parameters
dbutils = None
