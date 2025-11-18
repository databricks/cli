package databricks

import (
	"context"
	"fmt"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// DescribeTableArgs represents arguments for describing a table
type DescribeTableArgs struct {
	TableFullName string `json:"table_full_name"`
	SampleSize    int    `json:"sample_size,omitempty"`
}

// ColumnMetadata represents metadata about a table column
type ColumnMetadata struct {
	Name     string  `json:"name"`
	DataType string  `json:"data_type"`
	Comment  *string `json:"comment,omitempty"`
}

// TableDetails represents detailed information about a table
type TableDetails struct {
	FullName         string           `json:"full_name"`
	TableType        string           `json:"table_type"`
	Owner            *string          `json:"owner,omitempty"`
	Comment          *string          `json:"comment,omitempty"`
	StorageLocation  *string          `json:"storage_location,omitempty"`
	DataSourceFormat *string          `json:"data_source_format,omitempty"`
	Columns          []ColumnMetadata `json:"columns"`
	SampleData       []map[string]any `json:"sample_data,omitempty"`
	RowCount         *int64           `json:"row_count,omitempty"`
}

// DescribeTable retrieves detailed information about a table including metadata and sample data
func DescribeTable(ctx context.Context, cfg *mcp.Config, args *DescribeTableArgs) (*TableDetails, error) {
	if args.SampleSize == 0 {
		args.SampleSize = 5
	}

	w := cmdctx.WorkspaceClient(ctx)

	// Get table metadata
	tableInfo, err := w.Tables.Get(ctx, catalog.GetTableRequest{
		FullName: args.TableFullName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %w", err)
	}

	// Build column metadata
	columns := make([]ColumnMetadata, len(tableInfo.Columns))
	for i, col := range tableInfo.Columns {
		var comment *string
		if col.Comment != "" {
			comment = &col.Comment
		}
		columns[i] = ColumnMetadata{
			Name:     col.Name,
			DataType: string(col.TypeName),
			Comment:  comment,
		}
	}

	var owner, tableComment, storageLocation, dataSourceFormat *string
	if tableInfo.Owner != "" {
		owner = &tableInfo.Owner
	}
	if tableInfo.Comment != "" {
		tableComment = &tableInfo.Comment
	}
	if tableInfo.StorageLocation != "" {
		storageLocation = &tableInfo.StorageLocation
	}
	if tableInfo.DataSourceFormat != "" {
		dsf := string(tableInfo.DataSourceFormat)
		dataSourceFormat = &dsf
	}

	details := &TableDetails{
		FullName:         args.TableFullName,
		TableType:        string(tableInfo.TableType),
		Owner:            owner,
		Comment:          tableComment,
		StorageLocation:  storageLocation,
		DataSourceFormat: dataSourceFormat,
		Columns:          columns,
	}

	// Get sample data if requested
	if args.SampleSize > 0 {
		query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", args.TableFullName, args.SampleSize)
		sampleResult, err := ExecuteQuery(ctx, cfg, &ExecuteQueryArgs{Query: query})
		if err == nil && sampleResult.RowCount > 0 {
			// Convert [][]any to []map[string]any
			sampleData := make([]map[string]any, sampleResult.RowCount)
			for i, row := range sampleResult.Rows {
				rowMap := make(map[string]any)
				for j, val := range row {
					if j < len(sampleResult.Columns) {
						rowMap[sampleResult.Columns[j]] = val
					}
				}
				sampleData[i] = rowMap
			}
			details.SampleData = sampleData
		}
		// Note: We intentionally don't return error for sample data failures
		// as the table metadata is still valuable
	}

	// Get row count
	countQuery := "SELECT COUNT(*) as count FROM " + args.TableFullName
	countResult, err := ExecuteQuery(ctx, cfg, &ExecuteQueryArgs{Query: countQuery})
	if err == nil && countResult.RowCount > 0 {
		// The count is in the first row, first column
		if len(countResult.Rows) > 0 && len(countResult.Rows[0]) > 0 {
			if count, ok := countResult.Rows[0][0].(int64); ok {
				details.RowCount = &count
			} else if countFloat, ok := countResult.Rows[0][0].(float64); ok {
				count := int64(countFloat)
				details.RowCount = &count
			}
		}
	}
	// Note: We intentionally don't return error for row count failures

	return details, nil
}
