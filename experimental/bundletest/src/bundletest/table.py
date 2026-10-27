"""Table and column assertion handles.

These are written once against the backend seam (``execute_sql`` + ``table_schema``)
and work unchanged on every backend.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from bundletest.backend import Backend


class FileHandle:
    """Assertions on a file uploaded to a volume."""

    def __init__(self, backend: Backend, volume: str, filename: str):
        self._backend = backend
        self.volume = volume
        self.filename = filename

    def exists(self) -> bool:
        try:
            self._backend.read_volume_file(self.volume, self.filename)
            return True
        except FileNotFoundError:
            return False

    def rows(self) -> list[dict]:
        return self._backend.read_volume_file(self.volume, self.filename)

    def row_count(self) -> int:
        return len(self.rows())

    @property
    def columns(self) -> list[str]:
        rows = self.rows()
        return list(rows[0].keys()) if rows else []


class ColumnHandle:
    """Column-level assertions."""

    def __init__(self, backend: Backend, fqn: str, name: str):
        self._backend = backend
        self.fqn = fqn
        self.name = name

    def min(self) -> Any:
        return self._backend.execute_sql(f"SELECT MIN({self.name}) FROM {self.fqn}")[0][0]

    def max(self) -> Any:
        return self._backend.execute_sql(f"SELECT MAX({self.name}) FROM {self.fqn}")[0][0]

    def is_unique(self) -> bool:
        rows = self._backend.execute_sql(f"SELECT {self.name} FROM {self.fqn} GROUP BY {self.name} HAVING COUNT(*) > 1")
        return len(rows) == 0


class TableHandle:
    """Table-level assertions."""

    def __init__(self, backend: Backend, fqn: str):
        self._backend = backend
        self.fqn = fqn

    def row_count(self) -> int:
        return self._backend.execute_sql(f"SELECT COUNT(*) FROM {self.fqn}")[0][0]

    def exists(self) -> bool:
        try:
            self._backend.execute_sql(f"SELECT 1 FROM {self.fqn} LIMIT 1")
            return True
        except Exception:
            return False

    @property
    def schema(self) -> dict[str, str]:
        return self._backend.table_schema(self.fqn)

    def has_no_nulls(self, column: str) -> bool:
        count = self._backend.execute_sql(f"SELECT COUNT(*) FROM {self.fqn} WHERE {column} IS NULL")[0][0]
        return count == 0

    def column(self, name: str) -> ColumnHandle:
        return ColumnHandle(self._backend, self.fqn, name)
