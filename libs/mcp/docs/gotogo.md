Alalyze this code base. I want to create a tool that has similar functionality to `edda_mcp` but is written in go.

Requirements:

- use idiomatic go - I don't need a 1:1 mapping of the rust code
- use the Databricks SDK for Go to make calls to Databricks (replaces databricks.rs) - see https://docs.databricks.com/aws/en/dev-tools/sdk-go
- use an abstraction for the execution sandbox for reading and writing files and executing code. The fist implementation can use local file system APIs biut I should be able to add a dagger backend later
- I do not need google sheets integration
- keep the code clean. Don't add inline comments. Comments to document the public API are OK
- add unit tests

Create a high level plan and then for each high level step create detailed instructions.

Write out the high level plan to a markdown file and create separate markdown files for each step.

afterwards add each step to the beads issue tracker. Run `bd quickstart` to learn more