package metricviews

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateModelChecksRequiredValues(t *testing.T) {
	want := map[string]string{"source": "source", "expr": "expr", "parameter name": "name", "join source": "source", "materialization schedule": "schedule"}
	for name, view := range map[string]*MetricView{
		"source": {SingleSource: &SingleSourceMetricView{Version: "1.1", Dimensions: []ColumnV11{{Expr: "id"}}}},
		"expr":   {SingleSource: &SingleSourceMetricView{Version: "1.1", Source: "main.sales.orders", Dimensions: []ColumnV11{{}}}},
		"parameter name": {SingleSource: &SingleSourceMetricView{
			Version: "1.1", Source: "main.sales.orders", Parameters: []ParameterV11{{DataType: "STRING"}},
		}},
		"join source": {SingleSource: &SingleSourceMetricView{
			Version: "1.1", Source: "main.sales.orders", Joins: []Join{{Name: "items"}},
		}},
		"materialization schedule": {SingleSource: &SingleSourceMetricView{
			Version: "1.1", Source: "main.sales.orders", Materialization: &Materialization{Mode: "fresh", MaterializedViews: []MaterializedView{{Name: "mv", MVType: "aggregated"}}},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Validate(view), want[name])
		})
	}
}

func TestValidateModelChecksFormatVariants(t *testing.T) {
	usd := "USD"
	for name, format := range map[string]*ColumnFormat{
		"uppercase type":          {Type: "NUMBER"},
		"currency_code":           {Type: "currency"},
		"date_format":             {Type: "date"},
		"time_format":             {Type: "date_time", DateFormat: &usd},
		"decimal_places.type":     {Type: "number", DecimalPlaces: &DecimalPlaces{Type: "MAX"}},
		"currency_code on number": {Type: "number", CurrencyCode: &usd},
	} {
		t.Run(name, func(t *testing.T) {
			view := &MetricView{SingleSource: &SingleSourceMetricView{Version: "1.1", Source: "main.sales.orders", Dimensions: []ColumnV11{{Expr: "amount", Format: format}}}}
			require.Error(t, Validate(view))
		})
	}
}

func TestValidateModelAcceptsBundleShape(t *testing.T) {
	view := &MetricView{SingleSource: &SingleSourceMetricView{Version: "1.1", Source: "main.sales.orders", Dimensions: []ColumnV11{{Expr: "source.*"}}}}
	require.NoError(t, Validate(view))
}

func TestValidateModelRejectsInvalidDiscriminator(t *testing.T) {
	for name, view := range map[string]*MetricView{
		"missing multi-source type":  {MultiSource: &MultiSourceMetricView{Version: "1.1", Sources: []SourceNode{{Name: "orders", From: "main.sales.orders"}}}},
		"unknown single-source type": {SingleSource: &SingleSourceMetricView{Version: "1.1", ViewType: "OTHER", Source: "main.sales.orders"}},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Validate(view), "view_type")
		})
	}
}
