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
		sampleData, err := ExecuteQuery(ctx, cfg, query)
		if err == nil && len(sampleData) > 0 {
			details.SampleData = sampleData
		}
		// Note: We intentionally don't return error for sample data failures
		// as the table metadata is still valuable
	}

	// Get row count
	countQuery := "SELECT COUNT(*) as count FROM " + args.TableFullName
	countData, err := ExecuteQuery(ctx, cfg, countQuery)
	if err == nil && len(countData) > 0 {
		if count, ok := countData[0]["count"].(int64); ok {
			details.RowCount = &count
		}
	}
	// Note: We intentionally don't return error for row count failures

	return details, nil
}
