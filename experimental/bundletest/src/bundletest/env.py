"""The test-facing environment: BundleEnv, resource handles, and the fixture helper."""

from __future__ import annotations

import os
from contextlib import contextmanager
from typing import TYPE_CHECKING, Any, Iterator

from bundletest.backend import Backend, RunResult
from bundletest.table import TableHandle

if TYPE_CHECKING:
    pass

DEFAULT_BACKEND = "local"


class JobHandle:
    """A single job resource."""

    def __init__(self, backend: Backend, name: str):
        self._backend = backend
        self.name = name
        self._last: RunResult | None = None

    def run(self, params: dict[str, Any] | None = None) -> RunResult:
        self._last = self._backend.run_job(self.name, params)
        return self._last

    def last_run(self) -> RunResult:
        if self._last is None:
            raise RuntimeError(f"job {self.name!r} has not been run yet")
        return self._last


class _Jobs:
    """`env.jobs["name"]` accessor that reuses handles so `last_run()` persists."""

    def __init__(self, backend: Backend):
        self._backend = backend
        self._cache: dict[str, JobHandle] = {}

    def __getitem__(self, name: str) -> JobHandle:
        return self._cache.setdefault(name, JobHandle(self._backend, name))


class VolumeHandle:
    """A single volume resource."""

    def __init__(self, backend: Backend, name: str):
        self._backend = backend
        self.name = name

    def upload(self, src: str, dst: str | None = None) -> None:
        self._backend.put_file(dst or f"/Volumes/{self.name}/{os.path.basename(src)}", src)


class BundleEnv:
    """A deployed bundle under test, backed by a single ``Backend``."""

    def __init__(self, bundle_path: str, backend: Backend):
        self.bundle_path = bundle_path
        self.backend = backend
        self.jobs = _Jobs(backend)

    # --- lifecycle ---
    def deploy(self) -> None:
        self.backend.deploy(self.bundle_path)

    def teardown(self) -> None:
        self.backend.teardown()

    # --- scaffolding: stand in for an upstream resource ---
    def seed(self, table: str, rows: list[dict[str, Any]]) -> None:
        self.backend.seed_table(table, rows)

    # --- resource handles ---
    def table(self, fqn: str) -> TableHandle:
        return TableHandle(self.backend, fqn)

    def volume(self, name: str) -> VolumeHandle:
        return VolumeHandle(self.backend, name)

    def run_job(self, name: str, params: dict[str, Any] | None = None) -> RunResult:
        return self.jobs[name].run(params)


def make_backend(kind: str, **kwargs: Any) -> Backend:
    """Construct a backend by name. Backends are imported lazily so selecting one
    never pulls in the others' dependencies."""
    if kind == "local":
        from bundletest.backends.duckdb import DuckDBBackend

        return DuckDBBackend(**kwargs)
    if kind == "cloud":
        raise NotImplementedError(
            "the cloud backend arrives in a follow-up PR on top of this base branch"
        )
    raise ValueError(f"unknown backend {kind!r} (expected 'local' or 'cloud')")


def current_backend_kind() -> str:
    """Backend selected for this run. Lets the same test run against either tier
    without editing test code."""
    return os.environ.get("BUNDLETEST_BACKEND", DEFAULT_BACKEND)


@contextmanager
def bundle_env(
    bundle_path: str,
    backend: str | None = None,
    deploy: bool = True,
    **kwargs: Any,
) -> Iterator[BundleEnv]:
    """Build a backend, deploy the bundle, yield the env, and tear it down."""
    be = make_backend(backend or current_backend_kind(), **kwargs)
    env = BundleEnv(bundle_path, be)
    if deploy:
        env.deploy()
    try:
        yield env
    finally:
        env.teardown()
