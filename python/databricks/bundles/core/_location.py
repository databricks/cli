import inspect
import os
from dataclasses import dataclass
from typing import Callable, Optional

__all__ = [
    "Location",
]


@dataclass(kw_only=True, frozen=True)
class Location:
    file: str

    line: Optional[int] = None
    """
    Line number in the file. Line numbers are 1-based and should be greater than 0.
    """

    column: Optional[int] = None
    """
    Column number in the line. Column numbers are 1-based and should be greater than 0.
    """

    def __post_init__(self):
        if self.line is not None and self.line < 1:
            raise ValueError(f"Line number must be greater than 0, got {self.line}")

        if self.column is not None and self.column < 1:
            raise ValueError(f"Column number must be greater than 0, got {self.column}")

    @staticmethod
    def from_callable(fn: Callable) -> Optional["Location"]:
        """
        Capture location of callable. This is useful for creating
        diagnostics of decorated functions.
        """

        code = hasattr(fn, "__code__") and getattr(fn, "__code__")

        if not code:
            return None

        file = code.co_filename
        if file and file.startswith(os.getcwd()):
            # simplify path if we can
            file = os.path.relpath(file, os.getcwd())

        return Location(
            file=file,
            line=code.co_firstlineno,
            column=1,  # there is no way to get column
        )

    @staticmethod
    def from_stack_frame(depth: int = 0) -> "Location":
        """
        Capture location of the caller
        """

        stack = inspect.stack()
        frame = stack[1 + depth].frame

        return Location(
            file=frame.f_code.co_filename,
            line=frame.f_lineno,
            column=1,  # there is no way to get column
        )

    def as_dict(self) -> dict:
        def omit_none(values: dict):
            return {key: value for key, value in values.items() if value is not None}

        return omit_none(
            {
                "file": self.file,
                "line": self.line,
                "column": self.column,
            }
        )
