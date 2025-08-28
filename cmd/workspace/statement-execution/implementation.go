package statement_execution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

func executeStatement(ctx context.Context, w *databricks.WorkspaceClient, req *ExecuteStatementRequest) error {
	// Validate required fields
	if req.WarehouseId == "" {
		// Try to get warehouse_id from environment variable first
		if envWarehouseId := os.Getenv("DATABRICKS_WAREHOUSE_ID"); envWarehouseId != "" {
			req.WarehouseId = envWarehouseId
		} else {
			// Use default warehouse_id
			req.WarehouseId = "a5e694153a0d5e8c"
		}
	}
	if req.Statement == "" {
		return fmt.Errorf("statement is required")
	}

	// Set default values if not provided
	if req.Disposition == "" {
		req.Disposition = "INLINE"
	}
	if req.Format == "" {
		req.Format = "JSON_ARRAY"
	}
	if req.WaitTimeout == "" {
		req.WaitTimeout = "10s"
	}
	if req.OnWaitTimeout == "" {
		req.OnWaitTimeout = "CONTINUE"
	}
	if req.ByteLimit == 0 {
		req.ByteLimit = 16777216 // 16MB default
	}

	// Create API client
	apiClient, err := client.New(w.Config)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Execute request
	var statementResp StatementResponse
	err = apiClient.Do(ctx, "POST", "/api/2.0/sql/statements/", nil, nil, req, &statementResp)
	if err != nil {
		return fmt.Errorf("failed to execute statement: %w", err)
	}

	// Display results
	return displayStatementResponse(ctx, &statementResp)
}

func getStatement(ctx context.Context, w *databricks.WorkspaceClient, statementId string) error {
	if statementId == "" {
		return fmt.Errorf("statement_id is required")
	}

	// Create API client
	apiClient, err := client.New(w.Config)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Execute request
	var statementResp StatementResponse
	url := fmt.Sprintf("/api/2.0/sql/statements/%s", statementId)
	err = apiClient.Do(ctx, "GET", url, nil, nil, nil, &statementResp)
	if err != nil {
		return fmt.Errorf("failed to get statement: %w", err)
	}

	// Display results
	return displayStatementResponse(ctx, &statementResp)
}

func displayStatementResponse(ctx context.Context, resp *StatementResponse) error {
	// Display basic information
	cmdio.LogString(ctx, fmt.Sprintf("Statement ID: %s", resp.StatementId))
	cmdio.LogString(ctx, fmt.Sprintf("Status: %s", resp.Status.State))

	// Display error if present
	if resp.Status.Error != nil {
		cmdio.LogString(ctx, fmt.Sprintf("Error: %s (%s)",
			resp.Status.Error.Message, resp.Status.Error.ErrorCode))
		return nil
	}

	// Display manifest if present
	if resp.Manifest != nil {
		cmdio.LogString(ctx, fmt.Sprintf("Total Rows: %d", resp.Manifest.TotalRows))
		cmdio.LogString(ctx, fmt.Sprintf("Truncated: %t", resp.Manifest.Truncated))

		if len(resp.Manifest.Schema.Columns) > 0 {
			cmdio.LogString(ctx, "Schema:")
			for _, col := range resp.Manifest.Schema.Columns {
				cmdio.LogString(ctx, fmt.Sprintf("  %s: %s", col.Name, col.Type))
			}
		}
	}

	// Display results based on disposition
	if resp.Result != nil {
		// INLINE disposition
		cmdio.LogString(ctx, "Results (INLINE):")
		for i, row := range resp.Result.DataArray {
			if i >= 10 { // Limit output to first 10 rows
				cmdio.LogString(ctx, fmt.Sprintf("... and %d more rows", len(resp.Result.DataArray)-10))
				break
			}
			cmdio.LogString(ctx, fmt.Sprintf("  %v", row))
		}

		// Also provide structured JSON output
		if len(resp.Manifest.Schema.Columns) > 0 && len(resp.Result.DataArray) > 0 {
			cmdio.LogString(ctx, "\nStructured JSON format:")
			for i, row := range resp.Result.DataArray {
				if i >= 5 { // Limit to first 5 rows for structured output
					if len(resp.Result.DataArray) > 5 {
						cmdio.LogString(ctx, fmt.Sprintf("... and %d more rows", len(resp.Result.DataArray)-5))
					}
					break
				}

				// Create structured object
				rowObj := make(map[string]interface{})
				for j, col := range resp.Manifest.Schema.Columns {
					if j < len(row) {
						rowObj[col.Name] = row[j]
					}
				}

				// Convert to JSON
				if jsonBytes, err := json.MarshalIndent(rowObj, "", "  "); err == nil {
					cmdio.LogString(ctx, string(jsonBytes))
				}
			}
		}
	} else if len(resp.ExternalLinks) > 0 {
		// EXTERNAL_LINKS disposition
		cmdio.LogString(ctx, "Results (EXTERNAL_LINKS):")
		for i, link := range resp.ExternalLinks {
			cmdio.LogString(ctx, fmt.Sprintf("  Link %d: %s (expires: %s, size: %d bytes, rows: %d)",
				i+1, link.URL, link.Expiration.Format("2006-01-02T15:04:05Z07:00"), link.ByteSize, link.RowCount))
		}
	}

	return nil
}
