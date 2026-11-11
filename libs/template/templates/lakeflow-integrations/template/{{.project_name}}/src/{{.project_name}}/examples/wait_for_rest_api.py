import json
import re
import time
from dataclasses import dataclass
from datetime import timedelta
from typing import Any, Optional
from urllib.parse import unquote, urlsplit

from databricks.lakeflow.integrations import (
    Context,
    HttpUcConnectionParameter,
    integration,
    ParameterMetadata,
    PollResult,
    Sensor,
)
from databricks.sdk import WorkspaceClient
from databricks.sdk.service.serving import ExternalFunctionRequestHttpMethod as Method

CONDITION = re.compile(r"(?P<path>[\w.\[\]-]+)\s*(?P<operator>==|!=|\bin\b)\s*(?P<expected>.+)")
PREVIEW_LENGTH = 2000


@dataclass(frozen=True)
class Condition:
    text: str
    response_path: str
    operator: str
    expected: Any


@dataclass(frozen=True)
class WaitSettings:
    success_condition: Condition
    failure_condition: Optional[Condition]
    poll_interval_seconds: int
    poll_timeout_seconds: int
    output_mappings: dict[str, str]


@dataclass(frozen=True)
class Defer:
    status: str


@dataclass(frozen=True)
class Complete:
    outputs: dict[str, Any]


@integration(
    display_name="Wait for REST API",
    description="Polls a REST API until its response meets a condition, releasing compute between polls",
    icon="clock",
    promoted_parameter="path",
    parameters=dict(
        connection=ParameterMetadata(display_name="Connection", description="Unity Catalog connection to the API."),
        path=ParameterMetadata(
            display_name="Path", description="Status path after the connection's base path. For example, jobs/42."
        ),
        success_condition=ParameterMetadata(
            display_name="Success condition",
            description='Condition that ends the wait, using ==, !=, or in. For example, status == "done".',
        ),
        failure_condition=ParameterMetadata(
            display_name="Failure condition",
            description='Condition that fails the task. For example, status == "failed".',
        ),
        method=ParameterMetadata(display_name="Method", description="HTTP method of each poll."),
        query_params=ParameterMetadata(
            display_name="Query parameters",
            description="Added to the status URL. For example, run_id with 42 adds ?run_id=42.",
        ),
        headers=ParameterMetadata(display_name="Headers", description="Extra headers to send with each poll."),
        poll_interval_seconds=ParameterMetadata(
            display_name="Poll interval in seconds", description="Wait between polls."
        ),
        poll_timeout_seconds=ParameterMetadata(
            display_name="Timeout in seconds", description="How long to keep polling before the task fails."
        ),
        output_mappings=ParameterMetadata(
            display_name="Output mappings",
            description="Each key is a task value name and each value is its path in the final response. For "
            "example, rows with result.rows.",
        ),
    ),
)
class WaitForRestApi(Sensor):
    def __init__(
        self,
        connection: HttpUcConnectionParameter,
        path: str,
        success_condition: str,
        failure_condition: Optional[str] = None,
        method: Method = Method.GET,
        query_params: Optional[dict[str, str]] = None,
        headers: Optional[dict[str, str]] = None,
        poll_interval_seconds: int = 30,
        poll_timeout_seconds: int = 3600,
        output_mappings: Optional[dict[str, str]] = None,
    ) -> None:
        if poll_interval_seconds < 0 or poll_timeout_seconds < 0:
            raise ValueError("poll_interval_seconds and poll_timeout_seconds cannot be negative")
        if ".." in unquote(urlsplit(path).path).split("/"):
            raise ValueError(f"path can't contain '..' segments, got: {path}")
        self.settings = WaitSettings(
            success_condition=parse_condition("success_condition", success_condition, 'status == "done"'),
            failure_condition=(
                parse_condition("failure_condition", failure_condition, 'status == "failed"')
                if failure_condition
                else None
            ),
            poll_interval_seconds=poll_interval_seconds,
            poll_timeout_seconds=poll_timeout_seconds,
            output_mappings=output_mappings or {},
        )
        self.request = dict(conn=connection, method=method, path=path, params=query_params, headers=headers)

    def poll(self, ctx: Context) -> PollResult:
        waited_seconds = time.time() - get_or_save_start_time(ctx)

        workspace = WorkspaceClient()
        response = workspace.serving_endpoints.http_request(**self.request)
        decision = decide_poll(
            self.settings,
            response=parse_response(response),
            response_text=response.text,
            waited_seconds=waited_seconds,
        )

        match decision:
            case Defer(status):
                print(status)
                print(f"Polling again in {self.settings.poll_interval_seconds}s.")
                return PollResult.deferred(timedelta(seconds=self.settings.poll_interval_seconds))
            case Complete(outputs):
                print(f"{self.settings.success_condition.text} is met")
                print(f"Response: {preview(response.text)}")
                for name, value in outputs.items():
                    workspace.dbutils.jobs.taskValues.set(key=name, value=value)
                    print(f"Saved task value {name}: {preview(json.dumps(value))}")
                return PollResult.completed()


def decide_poll(settings: WaitSettings, response: Any, response_text: str, waited_seconds: float) -> Defer | Complete:
    failure = settings.failure_condition
    if failure and satisfies_condition(response, failure):
        raise RuntimeError(f"failure_condition met: {failure.text}. Response: {preview(response_text)}")

    success = settings.success_condition
    if satisfies_condition(response, success):
        return Complete(extract_outputs(response, settings.output_mappings))

    current_value = json.dumps(extract_nested_value(response, success.response_path))
    status = f"{success.text} is not met, {success.response_path} is {current_value}"
    if waited_seconds + settings.poll_interval_seconds > settings.poll_timeout_seconds:
        raise RuntimeError(
            f"Timed out after {settings.poll_timeout_seconds}s. {status}. Response: {preview(response_text)}"
        )

    return Defer(status)


def get_or_save_start_time(ctx: Context) -> float:
    # Deferring releases the compute, so the time the wait started is kept in a task value.
    start_time = ctx.get_task_value("start_time", default=0)
    if not start_time:
        start_time = time.time()
        ctx.set_task_value("start_time", start_time)

    return start_time


def parse_condition(name: str, text: str, example: str) -> Condition:
    error = ValueError(f"{name} must look like {example}, got: {text}")
    match = CONDITION.fullmatch(text.strip())
    if not match:
        raise error

    try:
        expected = json.loads(match["expected"])
    except json.JSONDecodeError:
        raise error from None

    if match["operator"] == "in" and not isinstance(expected, list):
        raise error

    return Condition(text, match["path"], match["operator"], expected)


def satisfies_condition(response: Any, condition: Condition) -> bool:
    actual = extract_nested_value(response, condition.response_path)
    # A missing field only meets a condition on null, so the wait goes on until the field appears.
    if actual is None and condition.expected is not None:
        return False

    candidates = condition.expected if condition.operator == "in" else [condition.expected]
    found = any(json_equal(actual, candidate) for candidate in candidates)
    return not found if condition.operator == "!=" else found


def json_equal(left: Any, right: Any) -> bool:
    # Python treats True as 1, but JSON true and 1 are different values.
    return left == right and isinstance(left, bool) == isinstance(right, bool)


def parse_response(response: Any) -> Any:
    try:
        return response.json()
    except ValueError:
        return response.text


def preview(text: str) -> str:
    if not text:
        return "(empty)"
    if len(text) > PREVIEW_LENGTH:
        return f"{text[:PREVIEW_LENGTH]}... (truncated)"
    return text


def extract_outputs(response: Any, mappings: dict[str, str]) -> dict[str, Any]:
    outputs = {}
    for name, response_path in mappings.items():
        value = extract_nested_value(response, response_path)
        if value is None:
            raise RuntimeError(f"The response has no value at {response_path}")

        outputs[name] = value
    return outputs


def extract_nested_value(response: Any, response_path: str) -> Any:
    # result.items[0].id becomes the keys result, items, 0, and id. A leading [0] indexes a list response.
    normalized_path = response_path.replace("[", ".").replace("]", "").lstrip(".")
    value = response
    for key in normalized_path.split("."):
        if isinstance(value, dict):
            value = value.get(key)
        elif isinstance(value, list) and key.isdigit() and int(key) < len(value):
            value = value[int(key)]
        else:
            return None
    return value
