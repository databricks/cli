from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from typing_extensions import Self


class CodeBuilder:
    def __init__(self):
        self._code = ""

    def append(self, *args: str) -> "Self":
        for arg in args:
            self._code += arg

        return self

    def indent(self):
        return self.append("    ")

    def newline(self) -> "Self":
        return self.append("\n")

    def append_list(self, args: list[str], sep: str = ",") -> "Self":
        return self.append(sep.join(args))

    def append_dict(self, args: dict[str, str], sep: str = ",") -> "Self":
        return self.append_list([f"{k}={v}" for k, v in args.items()], sep)

    def append_triple_quote(self) -> "Self":
        return self.append('"""')

    def append_repr(self, value) -> "Self":
        return self.append(repr(value))

    def build(self):
        return self._code
