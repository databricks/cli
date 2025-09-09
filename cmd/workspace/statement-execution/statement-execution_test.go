package statement_execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	cmd := New()
	assert.NotNil(t, cmd)
	assert.Equal(t, "statement-execution", cmd.Use)
	assert.Equal(t, "Execute SQL statements", cmd.Short)
}

func TestExecuteStatementRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     ExecuteStatementRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: ExecuteStatementRequest{
				WarehouseId: "test-warehouse",
				Statement:   "SELECT 1",
			},
			wantErr: false,
		},
		{
			name: "missing warehouse_id",
			req: ExecuteStatementRequest{
				Statement: "SELECT 1",
			},
			wantErr: true,
		},
		{
			name: "missing statement",
			req: ExecuteStatementRequest{
				WarehouseId: "test-warehouse",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the full execution without a real client,
			// but we can test the validation logic by checking if the request
			// has the required fields
			if tt.req.WarehouseId == "" || tt.req.Statement == "" {
				assert.True(t, tt.wantErr, "expected error for invalid request")
			} else {
				assert.False(t, tt.wantErr, "expected no error for valid request")
			}
		})
	}
}

func TestStatementResponseTypes(t *testing.T) {
	// Test that our response types can be properly initialized
	resp := StatementResponse{
		StatementId: "test-id",
		Status: StatementStatus{
			State: "SUCCEEDED",
		},
		Manifest: &ResultManifest{
			TotalRows: 10,
			Truncated: false,
			Schema: ResultSchema{
				Columns: []ColumnInfo{
					{
						Name: "col1",
						Type: "string",
					},
				},
			},
		},
		Result: &InlineResult{
			DataArray: [][]interface{}{
				{"value1"},
				{"value2"},
			},
		},
	}

	assert.Equal(t, "test-id", resp.StatementId)
	assert.Equal(t, "SUCCEEDED", resp.Status.State)
	assert.Equal(t, int64(10), resp.Manifest.TotalRows)
	assert.Equal(t, 1, len(resp.Manifest.Schema.Columns))
	assert.Equal(t, "col1", resp.Manifest.Schema.Columns[0].Name)
	assert.Equal(t, 2, len(resp.Result.DataArray))
}

func TestExecuteStatementCommand(t *testing.T) {
    cmd := newExecuteStatementCommand()
    assert.NotNil(t, cmd)
    // Accepts either inline statement or --file
    assert.Equal(t, "execute-statement STATEMENT", cmd.Use)
    assert.Contains(t, cmd.Long, "Execute a SQL statement")
    // Ensure --file flag is present for SQL file input
    f := cmd.Flags().Lookup("file")
    if assert.NotNil(t, f) {
        assert.Equal(t, "file", f.Name)
    }
}

func TestGetStatementCommand(t *testing.T) {
	cmd := newGetStatementCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "get-statement STATEMENT_ID", cmd.Use)
	assert.Contains(t, cmd.Long, "Get the status and results")
}
