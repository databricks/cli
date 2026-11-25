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

SlackConnection = Annotated[HttpUcConnectionParameter, HttpConnectionFilter(host="https://slack.com", base_path="/api")]


@integration(
    display_name="Send Slack message",
    description="Posts a message to a Slack channel or direct message with optional rich formatting",
    icon="send",
    promoted_parameter="channel",
    parameters=dict(
        connection=ParameterMetadata(
            display_name="Connection",
            description="Unity Catalog connection to the Slack API. Its bot token needs the chat:write scope.",
        ),
        channel=ParameterMetadata(display_name="Channel", description="Channel name or ID. For example, #alerts."),
        message=ParameterMetadata(
            display_name="Message",
            description="Text to post, in plain text or Slack markdown. Notifications show it when you also set blocks.",
        ),
        blocks=ParameterMetadata(
            display_name="Blocks", description="Slack Block Kit layout, in JSON, for richer formatting."
        ),
        thread_ts=ParameterMetadata(
            display_name="Thread timestamp", description="Timestamp of the message to reply to in its thread."
        ),
        username=ParameterMetadata(
            display_name="Username",
            description="Bot name shown on the message. Needs the chat:write.customize scope.",
        ),
        icon_url=ParameterMetadata(
            display_name="Icon URL",
            description="Image shown as the bot's avatar. Needs the chat:write.customize scope.",
        ),
        icon_emoji=ParameterMetadata(
            display_name="Icon emoji",
            description="Emoji shown as the bot's avatar. For example, :robot_face:. Needs the chat:write.customize scope.",
        ),
    ),
)
def send_slack_message(
    connection: SlackConnection,
    channel: str,
    message: TextAreaParameter,
    blocks: Optional[TextAreaParameter] = None,
    thread_ts: Optional[str] = None,
    username: Optional[str] = None,
    icon_url: Optional[str] = None,
    icon_emoji: Optional[str] = None,
) -> None:
    optional_fields = {
        "blocks": json.loads(blocks) if blocks else None,
        "thread_ts": thread_ts,
        "username": username,
        "icon_emoji": icon_emoji,
        "icon_url": None if icon_emoji else icon_url,
    }
    payload: dict[str, Any] = {"channel": channel, "text": message}
    payload |= {key: value for key, value in optional_fields.items() if value}

    workspace = WorkspaceClient()
    response = workspace.serving_endpoints.http_request(
        conn=connection,
        method=ExternalFunctionRequestHttpMethod.POST,
        path="chat.postMessage",
        json=payload,
    )

    try:
        result = response.json()
    except ValueError:
        raise RuntimeError(f"Slack returned a non-JSON response: {response.text}") from None

    # Slack reports API errors with HTTP 200 and ok=false.
    if not result.get("ok"):
        raise RuntimeError(f"Slack API error: {result.get('error')}")

    print(f"Posted message {result['ts']} to channel {result['channel']}")
    workspace.dbutils.jobs.taskValues.set(key="ts", value=result["ts"])
    workspace.dbutils.jobs.taskValues.set(key="channel_id", value=result["channel"])
