import datetime
from typing import Annotated, Any, Optional

from databricks.lakeflow.integrations import (
    Context,
    HttpConnectionFilter,
    HttpUcConnectionParameter,
    integration,
    ParameterMetadata,
    PollResult,
    Sensor,
)
from databricks.sdk import WorkspaceClient
from databricks.sdk.service.serving import ExternalFunctionRequestHttpMethod

SlackConnection = Annotated[HttpUcConnectionParameter, HttpConnectionFilter(host="https://slack.com", base_path="/api")]

MAX_RETRIES = 5


@integration(
    display_name="Wait for Slack reaction",
    description="Pause until someone reacts to a Slack message",
    icon="clock",
    promoted_parameter="message_ts",
    parameters=dict(
        connection=ParameterMetadata(
            display_name="Connection",
            description="Unity Catalog connection to the Slack API. Its bot token needs the reactions:read scope.",
        ),
        channel_id=ParameterMetadata(
            display_name="Channel ID",
            description="For example, {{tasks.send_slack_message.values.channel_id}} from a Send Slack message task.",
        ),
        message_ts=ParameterMetadata(
            display_name="Message timestamp",
            description="Slack's ID for the message. For example, {{tasks.send_slack_message.values.ts}} from a Send Slack message task.",
        ),
        emojis=ParameterMetadata(
            display_name="Emojis",
            description="Slack emoji names. For example, white_check_mark. Leave empty for any reaction.",
        ),
        poll_interval_seconds=ParameterMetadata(
            display_name="Poll interval (seconds)", description="How often to check for reactions."
        ),
    ),
)
class WaitForSlackReaction(Sensor):
    def __init__(
        self,
        connection: SlackConnection,
        channel_id: str,
        message_ts: str,
        emojis: Optional[list[str]] = None,
        poll_interval_seconds: int = 60,
    ):
        if poll_interval_seconds < 0:
            raise ValueError(f"Poll interval must be 0 seconds or more, got {poll_interval_seconds}.")

        self.connection = connection
        self.channel_id = channel_id
        self.message_ts = message_ts
        self.emojis = {emoji_base_name(emoji.strip(":")) for emoji in emojis or [] if emoji.strip(":")}
        self.poll_interval_seconds = poll_interval_seconds

    def poll(self, ctx: Context) -> PollResult:
        workspace = WorkspaceClient()
        try:
            reactions = self.get_reactions(workspace)
        except Exception as error:
            return self.retry_or_fail(ctx, error)

        ctx.set_task_value("failed_polls", 0)

        reaction = find_matching_reaction(reactions, self.emojis)
        if reaction is None:
            print(f"No matching reaction yet, checking again in {self.poll_interval_seconds} seconds")
            return PollResult.deferred(datetime.timedelta(seconds=self.poll_interval_seconds))

        emoji, user_id = emoji_base_name(reaction["name"]), reaction["users"][0]
        print(f"Found reaction :{emoji}: by user {user_id}")
        workspace.dbutils.jobs.taskValues.set(key="reaction", value=emoji)
        workspace.dbutils.jobs.taskValues.set(key="user_id", value=user_id)
        return PollResult.completed()

    def get_reactions(self, workspace: WorkspaceClient) -> list[dict[str, Any]]:
        response = workspace.serving_endpoints.http_request(
            conn=self.connection,
            method=ExternalFunctionRequestHttpMethod.GET,
            path="reactions.get",
            params={"channel": self.channel_id, "timestamp": self.message_ts, "full": "true"},
        )

        try:
            result = response.json()
        except ValueError:
            raise RuntimeError(f"Slack returned a non-JSON response: {response.text}") from None

        # Slack reports API errors with HTTP 200 and ok=false.
        if not result.get("ok"):
            raise RuntimeError(
                f"Slack API error for message {self.message_ts} in channel {self.channel_id}: {result.get('error')}"
            )
        return result["message"].get("reactions", [])

    def retry_or_fail(self, ctx: Context, error: Exception) -> PollResult:
        failed_polls = ctx.get_task_value("failed_polls", 0) + 1
        if failed_polls > MAX_RETRIES:
            raise RuntimeError(f"Checking for reactions failed {failed_polls} times in a row: {error}") from error

        ctx.set_task_value("failed_polls", failed_polls)
        print(
            f"Checking for reactions failed, retry {failed_polls} of {MAX_RETRIES} in {self.poll_interval_seconds} seconds: {error}"
        )
        return PollResult.deferred(datetime.timedelta(seconds=self.poll_interval_seconds))


def emoji_base_name(emoji: str) -> str:
    return emoji.split("::")[0]


def find_matching_reaction(reactions: list[dict[str, Any]], emojis: set[str]) -> Optional[dict[str, Any]]:
    # Slack lists reactions in the order they were first added, so the earliest match wins.
    for reaction in reactions:
        if not emojis or emoji_base_name(reaction["name"]) in emojis:
            return reaction
    return None
