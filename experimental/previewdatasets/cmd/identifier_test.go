package previewdatasetscmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteTableName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"three part", "main.sales.orders", "`main`.`sales`.`orders`"},
		{"two part", "sales.orders", "`sales`.`orders`"},
		{"hive metastore", "hive_metastore.sales.orders", "`hive_metastore`.`sales`.`orders`"},
		{"leading digit segment", "cat.sch.2020_sales", "`cat`.`sch`.`2020_sales`"},
		{"hyphen segment", "cat.sch.order-items", "`cat`.`sch`.`order-items`"},
		{"space segment", "cat.sch.my table", "`cat`.`sch`.`my table`"},
		{"reserved word segment", "cat.sch.order", "`cat`.`sch`.`order`"},
		{"internal backtick doubled", "cat.sch.wei`rd", "`cat`.`sch`.`wei``rd`"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := quoteTableName(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestQuoteTableNameErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"one part", "orders"},
		{"four part", "a.b.c.d"},
		{"empty", ""},
		{"empty middle segment", "cat..table"},
		{"trailing dot", "cat.sch."},
		{"leading dot", ".sch.table"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := quoteTableName(tc.in)
			assert.Error(t, err)
		})
	}
}
