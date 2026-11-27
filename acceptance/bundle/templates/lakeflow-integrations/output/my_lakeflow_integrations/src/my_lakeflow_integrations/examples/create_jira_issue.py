import json
from typing import Annotated, Any, Optional

from databricks.lakeflow.integrations import (
    HttpConnectionFilter,
    HttpUcConnectionParameter,
    integration,
    ParameterMetadata,
    TextAreaParameter,
)
from databricks.sdk import WorkspaceClient
from databricks.sdk.service.serving import ExternalFunctionRequestHttpMethod

JiraConnection = Annotated[HttpUcConnectionParameter, HttpConnectionFilter(host="https://api.atlassian.com")]


@integration(
    display_name="Create Jira issue",
    description="Creates a Jira issue",
    icon="plus-circle",
    promoted_parameter="project_key",
    parameters=dict(
        connection=ParameterMetadata(
            display_name="Connection", description="Unity Catalog connection to the Jira Cloud REST API."
        ),
        project_key=ParameterMetadata(display_name="Project key", description="For example, DATA."),
        summary=ParameterMetadata(display_name="Summary", description="Title of the issue."),
        issue_type=ParameterMetadata(display_name="Issue type", description="For example, Bug."),
        description=ParameterMetadata(display_name="Description", description="Body of the issue, in plain text."),
        assignee_email=ParameterMetadata(display_name="Assignee email", description="For example, ana@example.com."),
        labels=ParameterMetadata(display_name="Labels", description="For example, pipeline."),
        priority=ParameterMetadata(display_name="Priority", description="For example, High."),
        fields=ParameterMetadata(
            display_name="Other fields",
            description="More Jira fields to set, as a JSON object of field IDs and values. "
            'For example, {"customfield_10000": "data-eng"}. The fields above take precedence.',
        ),
    ),
)
def create_jira_issue(
    connection: JiraConnection,
    project_key: str,
    summary: str,
    issue_type: str = "Task",
    description: Optional[TextAreaParameter] = None,
    assignee_email: Optional[str] = None,
    labels: Optional[list[str]] = None,
    priority: Optional[str] = None,
    fields: Optional[TextAreaParameter] = None,
) -> None:
    workspace = WorkspaceClient()
    assignee_id = find_account_id_by_email(workspace, connection, assignee_email) if assignee_email else None
    optional_fields = {
        "description": plain_text_to_jira_document(description) if description else None,
        "assignee": {"accountId": assignee_id} if assignee_id else None,
        "labels": labels,
        "priority": {"name": priority} if priority else None,
    }
    issue_fields: dict[str, Any] = {
        **parse_other_fields(fields),
        "project": {"key": project_key},
        "summary": summary,
        "issuetype": {"name": issue_type},
        **{key: value for key, value in optional_fields.items() if value},
    }

    result = send_jira_request(
        workspace, connection, ExternalFunctionRequestHttpMethod.POST, "issue", {"fields": issue_fields}
    )
    print(f"Created issue {result['key']}")
    workspace.dbutils.jobs.taskValues.set(key="issue_key", value=result["key"])


def find_account_id_by_email(workspace: WorkspaceClient, connection: str, email: str) -> str:
    users = send_jira_request(
        workspace, connection, ExternalFunctionRequestHttpMethod.GET, "user/search", None, params={"query": email}
    )
    # Jira searches by prefix and returns a null email for users with a private profile.
    matches = [user for user in users if (user.get("emailAddress") or "").lower() == email.lower()]
    if len(matches) != 1:
        raise ValueError(f"Found {len(matches)} Jira users with email {email}. Enter the email of exactly one user.")
    return matches[0]["accountId"]


def parse_other_fields(fields: Optional[str]) -> dict[str, Any]:
    try:
        parsed = json.loads(fields) if fields else {}
    except json.JSONDecodeError as error:
        raise ValueError(f"Other fields must be a JSON object: {error}") from None
    if not isinstance(parsed, dict):
        raise ValueError("Other fields must be a JSON object, not a list or a single value")
    return parsed


def send_jira_request(
    workspace: WorkspaceClient,
    connection: str,
    method: ExternalFunctionRequestHttpMethod,
    path: str,
    body: Optional[dict[str, Any]],
    params: Optional[dict[str, str]] = None,
) -> Any:
    response = workspace.serving_endpoints.http_request(
        conn=connection, method=method, path=path, json=body, params=params
    )
    try:
        return response.json()
    except ValueError:
        raise RuntimeError(f"Jira returned a non-JSON response: {response.text}") from None


def plain_text_to_jira_document(text: str) -> dict[str, Any]:
    # Jira's v3 API only takes rich text as an Atlassian Document Format document.
    paragraphs = [
        {"type": "paragraph", "content": [{"type": "text", "text": line}]} for line in text.splitlines() if line.strip()
    ]
    return {"type": "doc", "version": 1, "content": paragraphs}
