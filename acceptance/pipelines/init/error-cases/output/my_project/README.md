# my_project

The 'my_project' project was generated by using the CLI Pipelines template.

## Setup

1. Install the Databricks CLI from https://docs.databricks.com/dev-tools/cli/databricks-cli.html

2. Install the Pipelines CLI:
   ```
   $ databricks install-pipelines-cli
   ```

3. Authenticate to your Databricks workspace, if you have not done so already:
    ```
    $ databricks auth login
    ```

4. Optionally, install developer tools such as the Databricks extension for Visual Studio Code from
   https://docs.databricks.com/dev-tools/vscode-ext.html. Or the PyCharm plugin from
   https://www.databricks.com/blog/announcing-pycharm-integration-databricks.


## Deploying pipelines

1. To deploy a development copy of this project, type:
    ```
    $ pipelines deploy --target dev
    ```
    (Note that "dev" is the default target, so the `--target` parameter
    is optional here.)

2. Similarly, to deploy a production copy, type:
   ```
   $ pipelines deploy --target prod
   ```

3. To run a pipeline, use the "run" command:
   ```
   $ pipelines run
   ```
