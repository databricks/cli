package dresources

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTestConfig("features", &resources.Feature{
		FullName:    "main.default.my_feature",
		Description: "Test feature",
		Source: ml.DataSource{
			DeltaTableSource: &ml.DeltaTableSource{
				FullName: "main.default.my_source",
			},
		},
		Function: ml.Function{
			ColumnSelection: &ml.ColumnSelection{Column: "amount"},
		},
		Entities:         []ml.EntityColumn{{Name: "id"}},
		TimeseriesColumn: &ml.TimeseriesColumn{Name: "ts"},
	})
}
