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

	Function   Function    `json:"function"`
	Source     DataSource  `json:"source"`
	TimeWindow *TimeWindow `json:"time_window,omitempty"`
}

// The types below re-type the API's duration fields as strings so a bundle can write a duration
// with day and hour units; featureState converts them before deploy. Each embeds the API type and
// shadows only the fields it re-types, so every other field resolves through the embedding.

type Function struct {
	ml.Function
	AggregationFunction *AggregationFunction `json:"aggregation_function,omitempty"`
}

type AggregationFunction struct {
	ml.AggregationFunction
	TimeWindow *TimeWindow `json:"time_window,omitempty"`
}

type TimeWindow struct {
	ml.TimeWindow
	Rolling  *RollingWindow  `json:"rolling,omitempty"`
	Sawtooth *SawtoothWindow `json:"sawtooth,omitempty"`
	Sliding  *SlidingWindow  `json:"sliding,omitempty"`
	Tumbling *TumblingWindow `json:"tumbling,omitempty"`
}

type RollingWindow struct {
	ml.RollingWindow
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type SawtoothWindow struct {
	ml.SawtoothWindow
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type SlidingWindow struct {
	ml.SlidingWindow
	Delay  string `json:"delay,omitempty"`
	Offset string `json:"offset,omitempty"`
}

type TumblingWindow struct {
	ml.TumblingWindow
	Delay  string `json:"delay,omitempty"`
	Offset string `json:"offset,omitempty"`
}

type DataSource struct {
	ml.DataSource
	Lateness *SourceLateness `json:"lateness,omitempty"`
}

type SourceLateness struct {
	ml.SourceLateness
	SettlingDelay string `json:"settling_delay,omitempty"`
}

// Each wrapper declares its own JSON methods, overriding the ones its embedded API type promotes.

func (c *Function) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c Function) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *AggregationFunction) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c AggregationFunction) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *TimeWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c TimeWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *RollingWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c RollingWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *SawtoothWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c SawtoothWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *SlidingWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c SlidingWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *TumblingWindow) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c TumblingWindow) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *DataSource) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c DataSource) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

func (c *SourceLateness) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c SourceLateness) MarshalJSON() ([]byte, error) {
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
