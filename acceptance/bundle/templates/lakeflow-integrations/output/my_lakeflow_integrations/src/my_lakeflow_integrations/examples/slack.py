import requests
from databricks.sdk import WorkspaceClient
from databricks.lakeflow.integrations import integration


@integration(display_name="Send Slack message")
def send_slack_message(
    text: str,
    conn_id: str,
    channel: str,  # Example: "C0A874NCL8N"
):
    w = WorkspaceClient()

    response = requests.post(
        f"{w.config.host}/api/2.0/unity-catalog/connections/{conn_id}/proxy/chat.postMessage",
        headers={
            **w.config.authenticate(),
            "Accept": "application/json",
            "Content-Type": "application/json",
            "Accept-Encoding": "identity",  # ask for no compression
        },
        json={
            "channel": channel,
            "text": text,
        },
    )
    print(response.json())
