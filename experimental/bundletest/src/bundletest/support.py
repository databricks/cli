"""Static report of what the local backend can exercise."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

from bundletest.backends.duckdb import _QUALIFIED, _RESERVED_CATALOGS, _VAR_REF
from bundletest.project import load_bundle_config


@dataclass(frozen=True)
class SupportEntry:
    resource: str
    task: str
    status: str
    reason: str


@dataclass(frozen=True)
class SupportReport:
    bundle_name: str
    entries: tuple[SupportEntry, ...]
    config_only_resources: int

    @property
    def local_count(self) -> int:
        return sum(entry.status == "local" for entry in self.entries)

    @property
    def cloud_count(self) -> int:
        return sum(entry.status == "cloud" for entry in self.entries)

    @property
    def config_count(self) -> int:
        return self.config_only_resources + sum(entry.status == "config" for entry in self.entries)

    @property
    def error_count(self) -> int:
        return sum(entry.status == "error" for entry in self.entries)


def inspect_local_support(root: Path) -> SupportReport:
    config = load_bundle_config(root)
    resources = config.get("resources", {}) or {}
    entries: list[SupportEntry] = []
    for job_name, job in (resources.get("jobs", {}) or {}).items():
        tasks = job.get("tasks", []) or []
        if not tasks:
            entries.append(SupportEntry(f"jobs.{job_name}", "-", "config", "no executable tasks declared"))
            continue
        for task in tasks:
            entries.append(_inspect_task(root, job_name, task))

    config_only = sum(
        len(declared or {}) for kind, declared in resources.items() if kind != "jobs" and isinstance(declared, dict)
    )
    return SupportReport(
        bundle_name=(config.get("bundle", {}) or {}).get("name", root.name),
        entries=tuple(entries),
        config_only_resources=config_only,
    )


def _inspect_task(root: Path, job_name: str, task: dict[str, Any]) -> SupportEntry:
    resource = f"jobs.{job_name}"
    task_key = task.get("task_key", "-")
    sql_task = task.get("sql_task")
    if not isinstance(sql_task, dict):
        kind = next((key for key in task if key.endswith("_task")), "unknown task")
        return SupportEntry(resource, task_key, "cloud", f"{kind} requires a Databricks workspace")

    file = sql_task.get("file")
    if not isinstance(file, dict) or not file.get("path"):
        return SupportEntry(resource, task_key, "cloud", "sql_task does not reference a local file")
    relative = str(file["path"])
    if _VAR_REF.search(relative):
        return SupportEntry(resource, task_key, "cloud", f"SQL path contains online reference {relative!r}")
    source = root / relative
    if not source.is_file():
        return SupportEntry(resource, task_key, "error", f"SQL file does not exist: {relative}")

    sql = source.read_text()
    reserved = next(
        (
            match.group(1)
            for match in _QUALIFIED.finditer(sql)
            if match.group(3) and match.group(1) in _RESERVED_CATALOGS
        ),
        None,
    )
    if reserved:
        return SupportEntry(resource, task_key, "cloud", f"catalog {reserved!r} is reserved by DuckDB")
    return SupportEntry(resource, task_key, "local", f"runs {relative}")


def format_support_report(root: Path, report: SupportReport) -> str:
    lines = [f"bundletest local support: {report.bundle_name}", f"bundle: {root}", ""]
    labels = {"local": "LOCAL", "cloud": "CLOUD", "config": "CONFIG", "error": "ERROR"}
    for entry in report.entries:
        target = entry.resource if entry.task == "-" else f"{entry.resource}/{entry.task}"
        lines.append(f"[{labels[entry.status]:6}] {target}: {entry.reason}")
    if report.config_only_resources:
        lines.append(f"[CONFIG] {report.config_only_resources} non-job resources: configuration assertions only")
    summary = f"summary: {report.local_count} local, {report.cloud_count} cloud-only, {report.config_count} config-only"
    if report.error_count:
        suffix = "error" if report.error_count == 1 else "errors"
        summary += f", {report.error_count} {suffix}"
    lines.extend(("", summary))
    return "\n".join(lines)
