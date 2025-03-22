import pytest

from databricks.bundles.core import Variable
from databricks.bundles.core._variable import variables


@variables
class MyVariables:
    foo: Variable[str]
    unresolved_foo: "Variable[str]"


def test_my_variables():
    assert MyVariables.foo == Variable(
        path="var.foo",
        type=str,
    )


def test_my_variables_unresolved():
    assert MyVariables.unresolved_foo == Variable(
        path="var.unresolved_foo",
        type=str,
    )


def test_my_variables_untyped():
    with pytest.raises(ValueError) as exc_info:

        @variables
        class UntypedVariables:  # noqa
            foo: Variable

    [msg] = exc_info.value.args

    assert msg == "Variable type must be specified for 'foo', e.g. Variable[str]"


def test_bad_type():
    with pytest.raises(ValueError) as exc_info:

        @variables
        class BadType:  # noqa
            foo: str

    [msg] = exc_info.value.args

    assert (
        msg
        == "Only 'Variable' type is allowed in classes annotated with @variables, got <class 'str'>"
    )
