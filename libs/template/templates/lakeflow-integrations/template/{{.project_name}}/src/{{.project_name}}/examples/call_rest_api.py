import json
from typing import Any, Optional
from urllib.parse import unquote, urlsplit

from databricks.lakeflow.integrations import (
    HttpUcConnectionParameter,
    integration,
    ParameterMetadata,
    TextAreaParameter,
)
from databricks.sdk import WorkspaceClient
from databricks.sdk.service.serving import ExternalFunctionRequestHttpMethod as Method

PREVIEW_LENGTH = 2000


@integration(
    display_name="Call REST API",
    description="Sends a request through a UC connection, saving the response as task values",
    icon="globe",
    promoted_parameter="path",
    parameters=dict(
        connection=ParameterMetadata(display_name="Connection", description="Unity Catalog connection to the API."),
        path=ParameterMetadata(
            display_name="Path", description="Path after the connection's base path. For example, jobs/42/refresh."
        ),
        method=ParameterMetadata(display_name="Method", description="HTTP method of the request."),
        query_params=ParameterMetadata(
            display_name="Query parameters",
            description="Added to the request URL. For example, limit with 10 adds ?limit=10.",
        ),
        headers=ParameterMetadata(display_name="Headers", description="Extra headers to send with the request."),
        body=ParameterMetadata(display_name="Body", description="Request body, in JSON."),
        output_mappings=ParameterMetadata(
            display_name="Output mappings",
            description="Each key is a task value name and each value is its path in the response. For example, "
            "job_id with id, or first_id with items[0].id.",
        ),
    ),
)
def call_rest_api(
    connection: HttpUcConnectionParameter,
    path: str,
    method: Method = Method.GET,
    query_params: Optional[dict[str, str]] = None,
    headers: Optional[dict[str, str]] = None,
    body: Optional[TextAreaParameter] = None,
    output_mappings: Optional[dict[str, str]] = None,
) -> None:
    if ".." in unquote(urlsplit(path).path).split("/"):
        raise ValueError(f"path can't contain '..' segments, got: {path}")
    json_body = parse_json_body(body)

    workspace = WorkspaceClient()
    print(f"Sending {method.value} {path}")
    response = workspace.serving_endpoints.http_request(
        conn=connection, method=method, path=path, params=query_params, headers=headers, json=json_body
    )
    print(f"Response: {preview(response.text)}")

    outputs = extract_outputs(parse_response(response), output_mappings or {})
    for name, value in outputs.items():
        workspace.dbutils.jobs.taskValues.set(key=name, value=value)
        print(f"Saved task value {name}: {preview(json.dumps(value))}")


def parse_json_body(body: Optional[str]) -> Any:
    try:
        return json.loads(body) if body else None
    except json.JSONDecodeError as error:
        raise ValueError(f"body must be valid JSON: {error}") from None


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
