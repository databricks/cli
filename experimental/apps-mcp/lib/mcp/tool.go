package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// AddTool registers a typed tool handler with automatic schema generation and validation.
// This is the high-level API that most users should use.
//
// The In type provides the input schema, and Out provides the output schema.
// Input validation is automatic. Output is automatically converted to Content.
func AddTool[In, Out any](
	server *Server,
	tool *Tool,
	handler ToolHandlerFor[In, Out],
) {
	// Generate input schema if not provided
	if tool.InputSchema == nil {
		schema, err := jsonschema.For[In](nil)
		if err != nil {
			panic(fmt.Errorf("failed to generate input schema for tool %s: %w", tool.Name, err))
		}
		tool.InputSchema = schema
	}

	// Create a low-level handler that wraps the typed handler
	lowLevelHandler := func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
		// Unmarshal input
		var input In
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
				return nil, fmt.Errorf("invalid input: %w", err)
			}
		}

		// Call the typed handler
		result, output, err := handler(ctx, req, input)
		// If there's an error, wrap it in a tool result
		if err != nil {
			content := &TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error: %v", err),
			}
			return &CallToolResult{
				Content: []Content{content},
				IsError: true,
			}, nil
		}

		// If result is nil, create a default result from output
		if result == nil {
			result = &CallToolResult{}
		}

		// If content is empty, generate content from output
		if len(result.Content) == 0 {
			// Check if output is the zero value
			outputValue := reflect.ValueOf(output)
			if !isZeroValue(outputValue) {
				// Marshal output to JSON
				outputJSON, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal output: %w", err)
				}

				content := &TextContent{
					Type: "text",
					Text: string(outputJSON),
				}
				result.Content = []Content{content}
			}
		}

		return result, nil
	}

	// Register the tool with the server
	server.AddTool(tool, lowLevelHandler)
}

// isZeroValue checks if a reflect.Value is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		zero := reflect.Zero(v.Type())
		return v.Interface() == zero.Interface()
	}
}
