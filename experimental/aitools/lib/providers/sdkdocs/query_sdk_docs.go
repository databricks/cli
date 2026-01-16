package sdkdocs

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/databricks/cli/experimental/aitools/lib/mcp"
	"github.com/databricks/cli/libs/log"
)

// QuerySDKDocsInput represents the input for the databricks_query_sdk_docs tool.
type QuerySDKDocsInput struct {
	Query    string `json:"query" jsonschema:"required" jsonschema_description:"Search query for SDK documentation (e.g., 'how to create a job', 'cluster configuration', 'JobSettings fields')"`
	Category string `json:"category,omitempty" jsonschema_description:"Optional category filter: 'services', 'methods', 'types', or 'enums'"`
	Service  string `json:"service,omitempty" jsonschema_description:"Optional service filter (e.g., 'jobs', 'clusters', 'pipelines', 'catalog')"`
	Limit    int    `json:"limit,omitempty" jsonschema_description:"Maximum number of results to return (default: 10, max: 50)"`
}

// querySDKDocs handles the databricks_query_sdk_docs tool invocation.
func (p *Provider) querySDKDocs(ctx context.Context, args QuerySDKDocsInput) (*mcpsdk.CallToolResult, any, error) {
	log.Debugf(ctx, "databricks_query_sdk_docs called: query=%q, category=%q, service=%q, limit=%d",
		args.Query, args.Category, args.Service, args.Limit)

	results := p.index.Search(SearchOptions{
		Query:    args.Query,
		Category: args.Category,
		Service:  args.Service,
		Limit:    args.Limit,
	})

	if len(results) == 0 {
		return mcpsdk.CreateNewTextContentResult(
			fmt.Sprintf("No SDK documentation found for query: %q\n\nTry:\n- Using different keywords\n- Removing filters\n- Checking available services: %s",
				args.Query, strings.Join(p.index.ListServices(), ", ")),
		), nil, nil
	}

	response := p.formatResponse(results)
	return mcpsdk.CreateNewTextContentResult(response), nil, nil
}

// formatResponse formats search results for LLM consumption.
func (p *Provider) formatResponse(results []SearchResult) string {
	var sb strings.Builder

	sb.WriteString("## SDK Documentation Results\n\n")

	for _, result := range results {
		switch result.Type {
		case "method":
			p.formatMethodResult(&sb, result)
		case "type":
			p.formatTypeResult(&sb, result)
		case "service":
			p.formatServiceResult(&sb, result)
		case "enum":
			p.formatEnumResult(&sb, result)
		}
		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

// formatMethodResult formats a method search result.
func (p *Provider) formatMethodResult(sb *strings.Builder, result SearchResult) {
	method := p.index.GetMethod(result.Service, result.Name)
	if method == nil {
		sb.WriteString(fmt.Sprintf("### Method: %s\n\n%s\n", result.Name, result.Description))
		return
	}

	sb.WriteString(fmt.Sprintf("### Method: %s.%s\n\n", result.Service, method.Name))

	if method.Signature != "" {
		sb.WriteString("**Signature:**\n```go\n")
		sb.WriteString(method.Signature)
		sb.WriteString("\n```\n\n")
	}

	if method.Description != "" {
		sb.WriteString("**Description:**\n")
		sb.WriteString(method.Description)
		sb.WriteString("\n\n")
	}

	if len(method.Parameters) > 0 {
		sb.WriteString("**Parameters:**\n")
		for _, param := range method.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}
			sb.WriteString(fmt.Sprintf("- `%s` (%s)%s: %s\n", param.Name, param.Type, required, param.Description))
		}
		sb.WriteString("\n")
	}

	if method.Returns != nil {
		sb.WriteString("**Returns:**\n")
		sb.WriteString(fmt.Sprintf("- `%s`: %s\n\n", method.Returns.Type, method.Returns.Description))
	}

	if method.Example != "" {
		sb.WriteString("**Example:**\n```go\n")
		sb.WriteString(method.Example)
		sb.WriteString("\n```\n")
	}
}

// formatTypeResult formats a type search result.
func (p *Provider) formatTypeResult(sb *strings.Builder, result SearchResult) {
	typeDoc := p.index.GetType(result.Path)
	if typeDoc == nil {
		sb.WriteString(fmt.Sprintf("### Type: %s\n\n%s\n", result.Name, result.Description))
		return
	}

	sb.WriteString(fmt.Sprintf("### Type: %s\n\n", typeDoc.Name))

	if typeDoc.Package != "" {
		sb.WriteString(fmt.Sprintf("**Package:** `%s`\n\n", typeDoc.Package))
	}

	if typeDoc.Description != "" {
		sb.WriteString("**Description:**\n")
		sb.WriteString(typeDoc.Description)
		sb.WriteString("\n\n")
	}

	if len(typeDoc.Fields) > 0 {
		sb.WriteString("**Fields:**\n\n")
		sb.WriteString("| Field | Type | Required | Description |\n")
		sb.WriteString("|-------|------|----------|-------------|\n")

		for _, field := range typeDoc.Fields {
			required := "No"
			if field.Required {
				required = "Yes"
			}
			desc := field.Description
			if field.OutputOnly {
				desc = "(output-only) " + desc
			}
			if field.Deprecated {
				desc = "(deprecated) " + desc
			}
			// Escape pipe characters in descriptions
			desc = strings.ReplaceAll(desc, "|", "\\|")
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", field.Name, field.Type, required, truncate(desc, 100)))
		}
		sb.WriteString("\n")
	}
}

// formatServiceResult formats a service search result.
func (p *Provider) formatServiceResult(sb *strings.Builder, result SearchResult) {
	service := p.index.GetService(result.Path)
	if service == nil {
		sb.WriteString(fmt.Sprintf("### Service: %s\n\n%s\n", result.Name, result.Description))
		return
	}

	sb.WriteString(fmt.Sprintf("### Service: %s\n\n", service.Name))

	if service.Description != "" {
		sb.WriteString("**Description:**\n")
		sb.WriteString(service.Description)
		sb.WriteString("\n\n")
	}

	if len(service.Methods) > 0 {
		sb.WriteString("**Available Methods:**\n")
		for methodName, method := range service.Methods {
			desc := truncate(method.Description, 80)
			sb.WriteString(fmt.Sprintf("- `%s`: %s\n", methodName, desc))
		}
		sb.WriteString("\n")
	}
}

// formatEnumResult formats an enum search result.
func (p *Provider) formatEnumResult(sb *strings.Builder, result SearchResult) {
	enumDoc := p.index.GetEnum(result.Path)
	if enumDoc == nil {
		sb.WriteString(fmt.Sprintf("### Enum: %s\n\n%s\n", result.Name, result.Description))
		return
	}

	sb.WriteString(fmt.Sprintf("### Enum: %s\n\n", enumDoc.Name))

	if enumDoc.Package != "" {
		sb.WriteString(fmt.Sprintf("**Package:** `%s`\n\n", enumDoc.Package))
	}

	if enumDoc.Description != "" {
		sb.WriteString("**Description:**\n")
		sb.WriteString(enumDoc.Description)
		sb.WriteString("\n\n")
	}

	if len(enumDoc.Values) > 0 {
		sb.WriteString("**Values:**\n")
		for _, value := range enumDoc.Values {
			sb.WriteString(fmt.Sprintf("- `%s`\n", value))
		}
		sb.WriteString("\n")
	}
}
