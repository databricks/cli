import traceback
from dataclasses import dataclass, field, replace
from enum import Enum
from io import StringIO
from typing import TYPE_CHECKING, Optional, TypeVar

from databricks.bundles.core._location import Location

_T = TypeVar("_T")

if TYPE_CHECKING:
    from typing_extensions import Self

__all__ = [
    "Diagnostic",
    "Diagnostics",
    "Severity",
]


class Severity(Enum):
    WARNING = "warning"
    ERROR = "error"


@dataclass(kw_only=True, frozen=True)
class Diagnostic:
    severity: Severity
    """
    Severity of the diagnostics item.
    """

    summary: str
    """
    Short summary of the error or warning.
    """

    detail: Optional[str] = None
    """
    Explanation of the error or warning.
    """

    path: Optional[tuple[str, ...]] = None
    """
    Path in databricks.yml where the error or warning occurred.
    """

    location: Optional[Location] = None
    """
    Source code location where the error or warning occurred.
    """

    def as_dict(self) -> dict:
        def omit_none(values: dict):
            return {key: value for key, value in values.items() if value is not None}

        if self.location:
            location = self.location.as_dict()
        else:
            location = None

        return omit_none(
            {
                "severity": self.severity.value,
                "summary": self.summary,
                "detail": self.detail,
                "path": self.path,
                "location": location,
            }
        )


@dataclass(frozen=True)
class Diagnostics:
    """
    Diagnostics is a collection of errors and warnings we print to users.

    Each item can have source location or path associated, that is reported in output to
    indicate where the error or warning occurred.
    """

    items: tuple[Diagnostic, ...] = field(default_factory=tuple, kw_only=False)

    def extend(self, diagnostics: "Self") -> "Self":
        """
        Extend items with another diagnostics. This pattern allows
        to accumulate errors and warnings.

        Example:

        .. code-block:: python

            def foo() -> Diagnostics: ...
            def bar() -> Diagnostics: ...

            diagnostics = Diagnostics()
            diagnostics = diagnostics.extend(foo())
            diagnostics = diagnostics.extend(bar())
        """

        return replace(
            self,
            items=(*self.items, *diagnostics.items),
        )

    def extend_tuple(self, pair: tuple[_T, "Self"]) -> tuple[_T, "Self"]:
        """
        Extend items with another diagnostics. This variant is useful when
        methods return a pair of value and diagnostics. This pattern allows
        to accumulate errors and warnings.

        Example:

        .. code-block:: python

            def foo() -> (int, Diagnostics): ...

            diagnostics = Diagnostics()
            value, diagnostics = diagnostics.extend_tuple(foo())
        """

        value, other_diagnostics = pair

        return value, self.extend(other_diagnostics)

    def has_error(self) -> bool:
        """
        Returns True if there is at least one error in diagnostics.
        """

        for item in self.items:
            if item.severity == Severity.ERROR:
                return True

        return False

    @classmethod
    def create_error(
        cls,
        msg: str,
        *,
        detail: Optional[str] = None,
        location: Optional[Location] = None,
        path: Optional[tuple[str, ...]] = None,
    ) -> "Self":
        """
        Create an error diagnostics.
        """

        return cls(
            items=(
                Diagnostic(
                    severity=Severity.ERROR,
                    summary=msg,
                    detail=detail,
                    location=location,
                    path=path,
                ),
            ),
        )

    @classmethod
    def create_warning(
        cls,
        msg: str,
        *,
        detail: Optional[str] = None,
        location: Optional[Location] = None,
        path: Optional[tuple[str, ...]] = None,
    ) -> "Self":
        """
        Create a warning diagnostics.
        """

        return cls(
            items=(
                Diagnostic(
                    severity=Severity.WARNING,
                    summary=msg,
                    detail=detail,
                    location=location,
                    path=path,
                ),
            )
        )

    @classmethod
    def from_exception(
        cls,
        exc: Exception,
        *,
        summary: str,
        location: Optional[Location] = None,
        path: Optional[tuple[str, ...]] = None,
        explanation: Optional[str] = None,
    ) -> "Self":
        """
        Create diagnostics from an exception.

        :param exc: exception to create diagnostics from
        :param summary: short summary of the error
        :param location: optional location in the source code where the error occurred
        :param path: optional path to relevant property in databricks.yml
        :param explanation: optional explanation to add to the details
        """

        detail_io = StringIO()
        traceback.print_exception(exc, file=detail_io)

        detail = detail_io.getvalue()
        if explanation:
            detail = f"{detail}\n\n\033[0;36mExplanation:\033[0m {explanation}"

        diagnostic = Diagnostic(
            severity=Severity.ERROR,
            summary=summary,
            location=location,
            path=path,
            detail=detail,
        )

        return cls(items=(diagnostic,))
