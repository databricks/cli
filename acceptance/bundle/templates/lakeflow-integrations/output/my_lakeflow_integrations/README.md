# my_lakeflow_integrations

A Declarative Automation Bundles project that extends Lakeflow Jobs with custom integrations.

This bundle ships with examples under `src/my_lakeflow_integrations/examples/` that call external
services through Unity Catalog connections:

| Example | Type | What it does |
| --- | --- | --- |
| `send_slack_message.send_slack_message` | Function | Posts a message to a Slack channel or direct message, with optional rich formatting. |
| `wait_for_slack_reaction.WaitForSlackReaction` | Sensor | Waits until someone reacts to a Slack message. |
| `create_jira_issue.create_jira_issue` | Function | Creates a Jira issue. |
| `call_rest_api.call_rest_api` | Function | Sends a request to a REST API and saves the response as task values. |
| `wait_for_rest_api.WaitForRestApi` | Sensor | Polls a REST API until its response meets a condition. |

Define jobs that run them under `resources/`.

## Getting started

1. Open this bundle in a Databricks workspace.
2. Click the **deployments rocket** 🚀 in the left sidebar to open the
   **Deployments** panel, then click **Deploy**.
3. To make the deployed integrations appear in the task palette, add them to `/Workspace/Users/<user>/.lakeflow_integrations.yml`. For example:
  ```yaml
  integrations:
    - '/Workspace/Users/<user>/.bundle/<bundle>/dev/artifacts/.internal/*.yml'
  ```

## Documentation

* [Declarative Automation Bundles](https://docs.databricks.com/aws/en/dev-tools/bundles/workspace-bundles)
* [Declarative Automation Bundles configuration reference](https://docs.databricks.com/aws/en/dev-tools/bundles/reference)
