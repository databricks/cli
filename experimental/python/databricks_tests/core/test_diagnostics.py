from databricks.bundles.core import (
    Diagnostic,
    Diagnostics,
    Location,
    Severity,
)


def test_catch_exceptions():
    diagnostics = Diagnostics()

    try:
        diagnostics = diagnostics.extend(
            Diagnostics.create_warning("foo is deprecated")
        )

        raise ValueError("foo is not available")

    except ValueError as exc:
        diagnostics = diagnostics.extend(
            Diagnostics.from_exception(
                exc=exc,
                summary="failed to get foo",
            )
        )

    assert len(diagnostics.items) == 2
    assert diagnostics.items[0].summary == "foo is deprecated"
    assert diagnostics.items[0].severity == Severity.WARNING

    assert diagnostics.items[1].summary == "failed to get foo"
    assert diagnostics.items[1].severity == Severity.ERROR


def test_extend():
    diagnostics = Diagnostics.create_warning("foo is deprecated")
    diagnostics = diagnostics.extend(Diagnostics.create_warning("bar is deprecated"))

    assert diagnostics == Diagnostics(
        items=(
            Diagnostic(
                severity=Severity.WARNING,
                summary="foo is deprecated",
            ),
            Diagnostic(
                severity=Severity.WARNING,
                summary="bar is deprecated",
            ),
        ),
    )


def test_extend_tuple():
    def foo() -> tuple[int, Diagnostics]:
        return 42, Diagnostics.create_warning("bar is deprecated")

    diagnostics = Diagnostics.create_warning("foo is deprecated")
    value, diagnostics = diagnostics.extend_tuple(foo())

    assert value == 42
    assert diagnostics == Diagnostics(
        items=(
            Diagnostic(
                severity=Severity.WARNING,
                summary="foo is deprecated",
            ),
            Diagnostic(
                severity=Severity.WARNING,
                summary="bar is deprecated",
            ),
        ),
    )


def test_location_from_callable():
    location = Location.from_callable(test_location_from_callable)

    assert location
    assert location.file == "databricks_tests/core/test_diagnostics.py"
    assert location.line and location.line > 0
    assert location.column and location.column > 0


def test_location_from_weird_callable():
    location = Location.from_callable(print)

    assert location is None
