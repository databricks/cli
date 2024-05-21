package main

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

func main() {
	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	if err != nil {
		panic(err)
	}

	messages, err := PromptMessage()
	if err != nil {
		panic(err)
	}

	// Please author Markdown documentation that first
	// creates the necessary resources, demonstrates usage of the commands, and then
	// finally cleans up the resources. Make sure to demonstrate usage of necessary flags. Remember to not delete
	// the resource before demonstrating the other commands. If the final step is to delete the resource, then
	// you can delete the resource at the end of the script.

	g := Find("jobs")
	contents := g.Prompt()

	// 	contents += `

	// The output should have the following structure:

	// # <command group>

	// <prose description of the command group>

	// ## Quick start

	// BEGIN

	// <
	// Markdown documentation that first creates the necessary resources, demonstrates usage of the commands, and then
	// finally cleans up the resources. Make sure to demonstrate usage of necessary flags. Remember to not delete
	// the resource before demonstrating the other commands. If the final step is to delete the resource, then
	// you can delete the resource at the end of the script.
	// Each command is a separate code block with prose separating them.
	// Do not include command output; you don't know.
	// Never use placeholders in the commands, but use for example "my_catalog" for a "CATALOG_NAME" placeholder.
	// All code blocks concatenated should be runnable as a bash script.
	// If commands depend on pre-existing resources, do not include the commands to create or destroy them,
	// but call out this requirement in a comment.
	// >

	// END
	// `

	// `
	// ## Commands

	// <Markdown headers for each command, with the command name as the header text>
	// `

	contents += `
Output an executable Bash script that demonstrates how to use these commands.
Insert comments where you expect the user to have pre-existing resources.
`

	messages = append(messages, serving.ChatMessage{
		Role:    serving.ChatMessageRoleUser,
		Content: contents,
	})

	res, err := w.ServingEndpoints.Query(ctx, serving.QueryEndpointInput{
		Name:     "databricks-dbrx-instruct",
		Messages: messages,
	})
	if err != nil {
		panic(err)
	}

	// enc := json.NewEncoder(os.Stdout)
	// enc.SetIndent("", "  ")
	// enc.Encode(res)

	fmt.Printf("Output: %s\n", res.Choices[0].Message.Content)
}
