package resources

import (
	"context"
	"net/url"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type Feature struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	ml.Feature

	// =========================================
	// === overrides over [ml.Feature] =========
	// =========================================

	Function   FeatureFunction    `json:"function"`
	Source     FeatureDataSource  `json:"source"`
	TimeWindow *FeatureTimeWindow `json:"time_window,omitempty"`
}

// The types below re-type the API's duration fields as strings so a bundle can write a duration
// with day and hour units; featureState converts them before deploy. Each embeds the API type and
// shadows only the fields it re-types, so every other field resolves through the embedding.

type FeatureFunction struct {
	ml.Function
	AggregationFunction *FeatureAggregationFunction `json:"aggregation_function,omitempty"`
}

type FeatureAggregationFunction struct {
	ml.AggregationFunction
	TimeWindow *FeatureTimeWindow `json:"time_window,omitempty"`
}

type FeatureTimeWindow struct {
	ml.TimeWindow
	Rolling  *FeatureRollingWindow  `json:"rolling,omitempty"`
	Sawtooth *FeatureSawtoothWindow `json:"sawtooth,omitempty"`
	Sliding  *FeatureSlidingWindow  `json:"sliding,omitempty"`
	Tumbling *FeatureTumblingWindow `json:"tumbling,omitempty"`
}

type FeatureRollingWindow struct {
	ml.RollingWindow
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type FeatureSawtoothWindow struct {
	ml.SawtoothWindow
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type FeatureSlidingWindow struct {
	ml.SlidingWindow
	Delay  string `json:"delay,omitempty"`
	Offset string `json:"offset,omitempty"`
}

type FeatureTumblingWindow struct {
	ml.TumblingWindow
	Delay  string `json:"delay,omitempty"`
	Offset string `json:"offset,omitempty"`
}

type FeatureDataSource struct {
	ml.DataSource
	Lateness *FeatureSourceLateness `json:"lateness,omitempty"`
}

type FeatureSourceLateness struct {
	ml.SourceLateness
	SettlingDelay string `json:"settling_delay,omitempty"`
}

// Each wrapper declares its own JSON methods, overriding the ones its embedded API type promotes.

func (c *FeatureFunction) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureFunction) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureAggregationFunction) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureAggregationFunction) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureTimeWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureTimeWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureRollingWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureRollingWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureSawtoothWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureSawtoothWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureSlidingWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureSlidingWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureTumblingWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureTumblingWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureDataSource) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureDataSource) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *FeatureSourceLateness) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c FeatureSourceLateness) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (f *Feature) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, f)
}

func (f Feature) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(f)
}

func (f *Feature) Exists(ctx context.Context, w *databricks.WorkspaceClient, fullName string) (bool, error) {
	_, err := w.FeatureEngineering.GetFeature(ctx, ml.GetFeatureRequest{
		FullName: fullName,
	})
	if err != nil {
		log.Debugf(ctx, "feature with full name %s does not exist: %v", fullName, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (*Feature) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "feature",
		PluralName:    "features",
		SingularTitle: "Feature",
		PluralTitle:   "Features",
	}
}

func (f *Feature) InitializeURL(baseURL url.URL) {
	if f.ID == "" {
		return
	}
	f.URL = workspaceurls.ResourceURL(baseURL, "features", f.ID)
}

// GetName returns the fully qualified name. Callers read it off the config, which only ever carries
// what the user wrote plus id and modified_status (see statemgmt.StateToBundle), so the OUTPUT_ONLY
// name field is always empty here.
func (f *Feature) GetName() string {
	return f.FullName
}
