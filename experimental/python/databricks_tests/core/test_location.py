import pytest

from databricks.bundles.core._location import Location


def test_line_number_positive():
    """
    Line numbers are 1-based and should be greater than 0.
    """
    with pytest.raises(ValueError, match="Line number must be greater than 0"):
        Location(file="test.py", line=0)


def test_column_number_positive():
    """
    Column numbers are 1-based and should be greater than 0.
    """
    with pytest.raises(ValueError, match="Column number must be greater than 0"):
        Location(file="test.py", line=1, column=0)
