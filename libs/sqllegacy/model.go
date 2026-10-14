// Code ported from github.com/databricks/databricks-sdk-go /service/sql/model.go.

package sqllegacy

import (
	"fmt"
)

// A JSON representing a dashboard containing widgets of visualizations and text
// boxes.
type Dashboard struct {
	// Whether the authenticated user can edit the query definition.
	CanEdit bool `json:"can_edit,omitempty"`
	// Timestamp when this dashboard was created.
	CreatedAt string `json:"created_at,omitempty"`
	// In the web application, query filters that share a name are coupled to a
	// single selection box if this value is `true`.
	DashboardFiltersEnabled bool `json:"dashboard_filters_enabled,omitempty"`
	// The ID for this dashboard.
	Id string `json:"id,omitempty"`
	// Indicates whether a dashboard is trashed. Trashed dashboards won't appear
	// in list views. If this boolean is `true`, the `options` property for this
	// dashboard includes a `moved_to_trash_at` timestamp. Items in trash are
	// permanently deleted after 30 days.
	IsArchived bool `json:"is_archived,omitempty"`
	// Whether a dashboard is a draft. Draft dashboards only appear in list
	// views for their owners.
	IsDraft bool `json:"is_draft,omitempty"`
	// Indicates whether this query object appears in the current user's
	// favorites list. This flag determines whether the star icon for favorites
	// is selected.
	IsFavorite bool `json:"is_favorite,omitempty"`
	// The title of the dashboard that appears in list views and at the top of
	// the dashboard page.
	Name string `json:"name,omitempty"`

	Options *DashboardOptions `json:"options,omitempty"`
	// The identifier of the workspace folder containing the object.
	Parent string `json:"parent,omitempty"`
	// * `CAN_VIEW`: Can view the query * `CAN_RUN`: Can run the query *
	// `CAN_EDIT`: Can edit the query * `CAN_MANAGE`: Can manage the query
	PermissionTier PermissionLevel `json:"permission_tier,omitempty"`
	// URL slug. Usually mirrors the query name with dashes (`-`) instead of
	// spaces. Appears in the URL for this query.
	Slug string `json:"slug,omitempty"`

	Tags []string `json:"tags,omitempty"`
	// Timestamp when this dashboard was last updated.
	UpdatedAt string `json:"updated_at,omitempty"`

	User *User `json:"user,omitempty"`
	// The ID of the user who owns the dashboard.
	UserId int `json:"user_id,omitempty"`

	Widgets []Widget `json:"widgets,omitempty"`
}

type DashboardOptions struct {
	// The timestamp when this dashboard was moved to trash. Only present when
	// the `is_archived` property is `true`. Trashed items are deleted after
	// thirty days.
	MovedToTrashAt string `json:"moved_to_trash_at,omitempty"`
}

// A JSON object representing a DBSQL data source / SQL warehouse.
type DataSource struct {
	// Data source ID maps to the ID of the data source used by the resource and
	// is distinct from the warehouse ID. [Learn more]
	//
	// [Learn more]: https://docs.databricks.com/api/workspace/datasources/list
	Id string `json:"id,omitempty"`
	// The string name of this data source / SQL warehouse as it appears in the
	// Databricks SQL web application.
	Name string `json:"name,omitempty"`
	// Reserved for internal use.
	PauseReason string `json:"pause_reason,omitempty"`
	// Reserved for internal use.
	Paused int `json:"paused,omitempty"`
	// Reserved for internal use.
	SupportsAutoLimit bool `json:"supports_auto_limit,omitempty"`
	// Reserved for internal use.
	Syntax string `json:"syntax,omitempty"`
	// The type of data source. For SQL warehouses, this will be
	// `databricks_internal`.
	Type string `json:"type,omitempty"`
	// Reserved for internal use.
	ViewOnly bool `json:"view_only,omitempty"`
	// The ID of the associated SQL warehouse, if this data source is backed by
	// a SQL warehouse.
	WarehouseId string `json:"warehouse_id,omitempty"`
}

type DatePrecision string

const DatePrecisionDayPrecision DatePrecision = `DAY_PRECISION`

const DatePrecisionMinutePrecision DatePrecision = `MINUTE_PRECISION`

const DatePrecisionSecondPrecision DatePrecision = `SECOND_PRECISION`

// String representation for [fmt.Print]
func (f *DatePrecision) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *DatePrecision) Set(v string) error {
	switch v {
	case `DAY_PRECISION`, `MINUTE_PRECISION`, `SECOND_PRECISION`:
		*f = DatePrecision(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "DAY_PRECISION", "MINUTE_PRECISION", "SECOND_PRECISION"`, v)
	}
}

// Values returns all possible values for DatePrecision.
//
// There is no guarantee on the order of the values in the slice.
func (f *DatePrecision) Values() []DatePrecision {
	return []DatePrecision{
		DatePrecisionDayPrecision,
		DatePrecisionMinutePrecision,
		DatePrecisionSecondPrecision,
	}
}

// Type always returns DatePrecision to satisfy [pflag.Value] interface
func (f *DatePrecision) Type() string {
	return "DatePrecision"
}

type DateRange struct {
	End string `json:"end"`

	Start string `json:"start"`
}

type DateRangeValue struct {
	// Manually specified date-time range value.
	DateRangeValue *DateRange `json:"date_range_value,omitempty"`
	// Dynamic date-time range value based on current date-time.
	DynamicDateRangeValue DateRangeValueDynamicDateRange `json:"dynamic_date_range_value,omitempty"`
	// Date-time precision to format the value into when the query is run.
	// Defaults to DAY_PRECISION (YYYY-MM-DD).
	Precision DatePrecision `json:"precision,omitempty"`

	StartDayOfWeek int `json:"start_day_of_week,omitempty"`
}

type DateRangeValueDynamicDateRange string

const DateRangeValueDynamicDateRangeLast12Months DateRangeValueDynamicDateRange = `LAST_12_MONTHS`

const DateRangeValueDynamicDateRangeLast14Days DateRangeValueDynamicDateRange = `LAST_14_DAYS`

const DateRangeValueDynamicDateRangeLast24Hours DateRangeValueDynamicDateRange = `LAST_24_HOURS`

const DateRangeValueDynamicDateRangeLast30Days DateRangeValueDynamicDateRange = `LAST_30_DAYS`

const DateRangeValueDynamicDateRangeLast60Days DateRangeValueDynamicDateRange = `LAST_60_DAYS`

const DateRangeValueDynamicDateRangeLast7Days DateRangeValueDynamicDateRange = `LAST_7_DAYS`

const DateRangeValueDynamicDateRangeLast8Hours DateRangeValueDynamicDateRange = `LAST_8_HOURS`

const DateRangeValueDynamicDateRangeLast90Days DateRangeValueDynamicDateRange = `LAST_90_DAYS`

const DateRangeValueDynamicDateRangeLastHour DateRangeValueDynamicDateRange = `LAST_HOUR`

const DateRangeValueDynamicDateRangeLastMonth DateRangeValueDynamicDateRange = `LAST_MONTH`

const DateRangeValueDynamicDateRangeLastWeek DateRangeValueDynamicDateRange = `LAST_WEEK`

const DateRangeValueDynamicDateRangeLastYear DateRangeValueDynamicDateRange = `LAST_YEAR`

const DateRangeValueDynamicDateRangeThisMonth DateRangeValueDynamicDateRange = `THIS_MONTH`

const DateRangeValueDynamicDateRangeThisWeek DateRangeValueDynamicDateRange = `THIS_WEEK`

const DateRangeValueDynamicDateRangeThisYear DateRangeValueDynamicDateRange = `THIS_YEAR`

const DateRangeValueDynamicDateRangeToday DateRangeValueDynamicDateRange = `TODAY`

const DateRangeValueDynamicDateRangeYesterday DateRangeValueDynamicDateRange = `YESTERDAY`

// String representation for [fmt.Print]
func (f *DateRangeValueDynamicDateRange) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *DateRangeValueDynamicDateRange) Set(v string) error {
	switch v {
	case `LAST_12_MONTHS`, `LAST_14_DAYS`, `LAST_24_HOURS`, `LAST_30_DAYS`, `LAST_60_DAYS`, `LAST_7_DAYS`, `LAST_8_HOURS`, `LAST_90_DAYS`, `LAST_HOUR`, `LAST_MONTH`, `LAST_WEEK`, `LAST_YEAR`, `THIS_MONTH`, `THIS_WEEK`, `THIS_YEAR`, `TODAY`, `YESTERDAY`:
		*f = DateRangeValueDynamicDateRange(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "LAST_12_MONTHS", "LAST_14_DAYS", "LAST_24_HOURS", "LAST_30_DAYS", "LAST_60_DAYS", "LAST_7_DAYS", "LAST_8_HOURS", "LAST_90_DAYS", "LAST_HOUR", "LAST_MONTH", "LAST_WEEK", "LAST_YEAR", "THIS_MONTH", "THIS_WEEK", "THIS_YEAR", "TODAY", "YESTERDAY"`, v)
	}
}

// Values returns all possible values for DateRangeValueDynamicDateRange.
//
// There is no guarantee on the order of the values in the slice.
func (f *DateRangeValueDynamicDateRange) Values() []DateRangeValueDynamicDateRange {
	return []DateRangeValueDynamicDateRange{
		DateRangeValueDynamicDateRangeLast12Months,
		DateRangeValueDynamicDateRangeLast14Days,
		DateRangeValueDynamicDateRangeLast24Hours,
		DateRangeValueDynamicDateRangeLast30Days,
		DateRangeValueDynamicDateRangeLast60Days,
		DateRangeValueDynamicDateRangeLast7Days,
		DateRangeValueDynamicDateRangeLast8Hours,
		DateRangeValueDynamicDateRangeLast90Days,
		DateRangeValueDynamicDateRangeLastHour,
		DateRangeValueDynamicDateRangeLastMonth,
		DateRangeValueDynamicDateRangeLastWeek,
		DateRangeValueDynamicDateRangeLastYear,
		DateRangeValueDynamicDateRangeThisMonth,
		DateRangeValueDynamicDateRangeThisWeek,
		DateRangeValueDynamicDateRangeThisYear,
		DateRangeValueDynamicDateRangeToday,
		DateRangeValueDynamicDateRangeYesterday,
	}
}

// Type always returns DateRangeValueDynamicDateRange to satisfy [pflag.Value] interface
func (f *DateRangeValueDynamicDateRange) Type() string {
	return "DateRangeValueDynamicDateRange"
}

type DateValue struct {
	// Manually specified date-time value.
	DateValue string `json:"date_value,omitempty"`
	// Dynamic date-time value based on current date-time.
	DynamicDateValue DateValueDynamicDate `json:"dynamic_date_value,omitempty"`
	// Date-time precision to format the value into when the query is run.
	// Defaults to DAY_PRECISION (YYYY-MM-DD).
	Precision DatePrecision `json:"precision,omitempty"`
}

type DateValueDynamicDate string

const DateValueDynamicDateNow DateValueDynamicDate = `NOW`

const DateValueDynamicDateYesterday DateValueDynamicDate = `YESTERDAY`

// String representation for [fmt.Print]
func (f *DateValueDynamicDate) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *DateValueDynamicDate) Set(v string) error {
	switch v {
	case `NOW`, `YESTERDAY`:
		*f = DateValueDynamicDate(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "NOW", "YESTERDAY"`, v)
	}
}

// Values returns all possible values for DateValueDynamicDate.
//
// There is no guarantee on the order of the values in the slice.
func (f *DateValueDynamicDate) Values() []DateValueDynamicDate {
	return []DateValueDynamicDate{
		DateValueDynamicDateNow,
		DateValueDynamicDateYesterday,
	}
}

// Type always returns DateValueDynamicDate to satisfy [pflag.Value] interface
func (f *DateValueDynamicDate) Type() string {
	return "DateValueDynamicDate"
}

type EnumValue struct {
	// List of valid query parameter values, newline delimited.
	EnumOptions string `json:"enum_options,omitempty"`
	// If specified, allows multiple values to be selected for this parameter.
	MultiValuesOptions *MultiValuesOptions `json:"multi_values_options,omitempty"`
	// List of selected query parameter values.
	Values []string `json:"values,omitempty"`
}

type LegacyQuery struct {
	// Describes whether the authenticated user is allowed to edit the
	// definition of this query.
	CanEdit bool `json:"can_edit,omitempty"`
	// The timestamp when this query was created.
	CreatedAt string `json:"created_at,omitempty"`
	// Data source ID maps to the ID of the data source used by the resource and
	// is distinct from the warehouse ID. [Learn more]
	//
	// [Learn more]: https://docs.databricks.com/api/workspace/datasources/list
	DataSourceId string `json:"data_source_id,omitempty"`
	// General description that conveys additional information about this query
	// such as usage notes.
	Description string `json:"description,omitempty"`
	// Query ID.
	Id string `json:"id,omitempty"`
	// Indicates whether the query is trashed. Trashed queries can't be used in
	// dashboards, or appear in search results. If this boolean is `true`, the
	// `options` property for this query includes a `moved_to_trash_at`
	// timestamp. Trashed queries are permanently deleted after 30 days.
	IsArchived bool `json:"is_archived,omitempty"`
	// Whether the query is a draft. Draft queries only appear in list views for
	// their owners. Visualizations from draft queries cannot appear on
	// dashboards.
	IsDraft bool `json:"is_draft,omitempty"`
	// Whether this query object appears in the current user's favorites list.
	// This flag determines whether the star icon for favorites is selected.
	IsFavorite bool `json:"is_favorite,omitempty"`
	// Text parameter types are not safe from SQL injection for all types of
	// data source. Set this Boolean parameter to `true` if a query either does
	// not use any text type parameters or uses a data source type where text
	// type parameters are handled safely.
	IsSafe bool `json:"is_safe,omitempty"`

	LastModifiedBy *User `json:"last_modified_by,omitempty"`
	// The ID of the user who last saved changes to this query.
	LastModifiedById int `json:"last_modified_by_id,omitempty"`
	// If there is a cached result for this query and user, this field includes
	// the query result ID. If this query uses parameters, this field is always
	// null.
	LatestQueryDataId string `json:"latest_query_data_id,omitempty"`
	// The title of this query that appears in list views, widget headings, and
	// on the query page.
	Name string `json:"name,omitempty"`

	Options *QueryOptions `json:"options,omitempty"`
	// The identifier of the workspace folder containing the object.
	Parent string `json:"parent,omitempty"`
	// * `CAN_VIEW`: Can view the query * `CAN_RUN`: Can run the query *
	// `CAN_EDIT`: Can edit the query * `CAN_MANAGE`: Can manage the query
	PermissionTier PermissionLevel `json:"permission_tier,omitempty"`
	// The text of the query to be run.
	Query string `json:"query,omitempty"`
	// A SHA-256 hash of the query text along with the authenticated user ID.
	QueryHash string `json:"query_hash,omitempty"`
	// Sets the **Run as** role for the object. Must be set to one of `"viewer"`
	// (signifying "run as viewer" behavior) or `"owner"` (signifying "run as
	// owner" behavior)
	RunAsRole RunAsRole `json:"run_as_role,omitempty"`

	Tags []string `json:"tags,omitempty"`
	// The timestamp at which this query was last updated.
	UpdatedAt string `json:"updated_at,omitempty"`

	User *User `json:"user,omitempty"`
	// The ID of the user who owns the query.
	UserId int `json:"user_id,omitempty"`

	Visualizations []LegacyVisualization `json:"visualizations,omitempty"`
}

// The visualization description API changes frequently and is unsupported. You
// can duplicate a visualization by copying description objects received _from
// the API_ and then using them to create a new one with a POST request to the
// same endpoint. Databricks does not recommend constructing ad-hoc
// visualizations entirely in JSON.
type LegacyVisualization struct {
	CreatedAt string `json:"created_at,omitempty"`
	// A short description of this visualization. This is not displayed in the
	// UI.
	Description string `json:"description,omitempty"`
	// The UUID for this visualization.
	Id string `json:"id,omitempty"`
	// The name of the visualization that appears on dashboards and the query
	// screen.
	Name string `json:"name,omitempty"`
	// The options object varies widely from one visualization type to the next
	// and is unsupported. Databricks does not recommend modifying visualization
	// settings in JSON.
	Options any `json:"options,omitempty"`

	Query *LegacyQuery `json:"query,omitempty"`
	// The type of visualization: chart, table, pivot table, and so on.
	Type string `json:"type,omitempty"`

	UpdatedAt string `json:"updated_at,omitempty"`
}

type LifecycleState string

const LifecycleStateActive LifecycleState = `ACTIVE`

const LifecycleStateTrashed LifecycleState = `TRASHED`

// String representation for [fmt.Print]
func (f *LifecycleState) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *LifecycleState) Set(v string) error {
	switch v {
	case `ACTIVE`, `TRASHED`:
		*f = LifecycleState(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "ACTIVE", "TRASHED"`, v)
	}
}

// Values returns all possible values for LifecycleState.
//
// There is no guarantee on the order of the values in the slice.
func (f *LifecycleState) Values() []LifecycleState {
	return []LifecycleState{
		LifecycleStateActive,
		LifecycleStateTrashed,
	}
}

// Type always returns LifecycleState to satisfy [pflag.Value] interface
func (f *LifecycleState) Type() string {
	return "LifecycleState"
}

type ListDashboardsRequest struct {
	// Name of dashboard attribute to order by.
	Order ListOrder `json:"-" url:"order,omitempty"`
	// Page number to retrieve.
	Page int `json:"-" url:"page,omitempty"`
	// Number of dashboards to return per page.
	PageSize int `json:"-" url:"page_size,omitempty"`
	// Full text search term.
	Q string `json:"-" url:"q,omitempty"`
}

type ListOrder string

const ListOrderCreatedAt ListOrder = `created_at`

const ListOrderName ListOrder = `name`

// String representation for [fmt.Print]
func (f *ListOrder) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *ListOrder) Set(v string) error {
	switch v {
	case `created_at`, `name`:
		*f = ListOrder(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "created_at", "name"`, v)
	}
}

// Values returns all possible values for ListOrder.
//
// There is no guarantee on the order of the values in the slice.
func (f *ListOrder) Values() []ListOrder {
	return []ListOrder{
		ListOrderCreatedAt,
		ListOrderName,
	}
}

// Type always returns ListOrder to satisfy [pflag.Value] interface
func (f *ListOrder) Type() string {
	return "ListOrder"
}

type ListResponse struct {
	// The total number of dashboards.
	Count int `json:"count,omitempty"`
	// The current page being displayed.
	Page int `json:"page,omitempty"`
	// The number of dashboards per page.
	PageSize int `json:"page_size,omitempty"`
	// List of dashboards returned.
	Results []Dashboard `json:"results,omitempty"`
}

type MultiValuesOptions struct {
	// Character that prefixes each selected parameter value.
	Prefix string `json:"prefix,omitempty"`
	// Character that separates each selected parameter value. Defaults to a
	// comma.
	Separator string `json:"separator,omitempty"`
	// Character that suffixes each selected parameter value.
	Suffix string `json:"suffix,omitempty"`
}

type NumericValue struct {
	Value float64 `json:"value,omitempty"`
}

type Parameter struct {
	// List of valid parameter values, newline delimited. Only applies for
	// dropdown list parameters.
	EnumOptions string `json:"enumOptions,omitempty"`
	// If specified, allows multiple values to be selected for this parameter.
	// Only applies to dropdown list and query-based dropdown list parameters.
	MultiValuesOptions *MultiValuesOptions `json:"multiValuesOptions,omitempty"`
	// The literal parameter marker that appears between double curly braces in
	// the query text.
	Name string `json:"name,omitempty"`
	// The UUID of the query that provides the parameter values. Only applies
	// for query-based dropdown list parameters.
	QueryId string `json:"queryId,omitempty"`
	// The text displayed in a parameter picking widget.
	Title string `json:"title,omitempty"`
	// Parameters can have several different types.
	Type ParameterType `json:"type,omitempty"`
	// The default value for this parameter.
	Value any `json:"value,omitempty"`
}

type ParameterType string

const ParameterTypeDatetime ParameterType = `datetime`

const ParameterTypeEnum ParameterType = `enum`

const ParameterTypeNumber ParameterType = `number`

const ParameterTypeQuery ParameterType = `query`

const ParameterTypeText ParameterType = `text`

// String representation for [fmt.Print]
func (f *ParameterType) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *ParameterType) Set(v string) error {
	switch v {
	case `datetime`, `enum`, `number`, `query`, `text`:
		*f = ParameterType(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "datetime", "enum", "number", "query", "text"`, v)
	}
}

// Values returns all possible values for ParameterType.
//
// There is no guarantee on the order of the values in the slice.
func (f *ParameterType) Values() []ParameterType {
	return []ParameterType{
		ParameterTypeDatetime,
		ParameterTypeEnum,
		ParameterTypeNumber,
		ParameterTypeQuery,
		ParameterTypeText,
	}
}

// Type always returns ParameterType to satisfy [pflag.Value] interface
func (f *ParameterType) Type() string {
	return "ParameterType"
}

// * `CAN_VIEW`: Can view the query * `CAN_RUN`: Can run the query * `CAN_EDIT`:
// Can edit the query * `CAN_MANAGE`: Can manage the query
type PermissionLevel string

// Can edit the query
const PermissionLevelCanEdit PermissionLevel = `CAN_EDIT`

// Can manage the query
const PermissionLevelCanManage PermissionLevel = `CAN_MANAGE`

// Can run the query
const PermissionLevelCanRun PermissionLevel = `CAN_RUN`

// Can view the query
const PermissionLevelCanView PermissionLevel = `CAN_VIEW`

// String representation for [fmt.Print]
func (f *PermissionLevel) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *PermissionLevel) Set(v string) error {
	switch v {
	case `CAN_EDIT`, `CAN_MANAGE`, `CAN_RUN`, `CAN_VIEW`:
		*f = PermissionLevel(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "CAN_EDIT", "CAN_MANAGE", "CAN_RUN", "CAN_VIEW"`, v)
	}
}

// Values returns all possible values for PermissionLevel.
//
// There is no guarantee on the order of the values in the slice.
func (f *PermissionLevel) Values() []PermissionLevel {
	return []PermissionLevel{
		PermissionLevelCanEdit,
		PermissionLevelCanManage,
		PermissionLevelCanRun,
		PermissionLevelCanView,
	}
}

// Type always returns PermissionLevel to satisfy [pflag.Value] interface
func (f *PermissionLevel) Type() string {
	return "PermissionLevel"
}

type Query struct {
	// Whether to apply a 1000 row limit to the query result.
	ApplyAutoLimit bool `json:"apply_auto_limit,omitempty"`
	// Name of the catalog where this query will be executed.
	Catalog string `json:"catalog,omitempty"`
	// Timestamp when this query was created.
	CreateTime string `json:"create_time,omitempty"`
	// General description that conveys additional information about this query
	// such as usage notes.
	Description string `json:"description,omitempty"`
	// Display name of the query that appears in list views, widget headings,
	// and on the query page.
	DisplayName string `json:"display_name,omitempty"`
	// UUID identifying the query.
	Id string `json:"id,omitempty"`
	// Username of the user who last saved changes to this query.
	LastModifierUserName string `json:"last_modifier_user_name,omitempty"`
	// Indicates whether the query is trashed.
	LifecycleState LifecycleState `json:"lifecycle_state,omitempty"`
	// Username of the user that owns the query.
	OwnerUserName string `json:"owner_user_name,omitempty"`
	// List of query parameter definitions.
	Parameters []QueryParameter `json:"parameters,omitempty"`
	// Workspace path of the workspace folder containing the object.
	ParentPath string `json:"parent_path,omitempty"`
	// Text of the query to be run.
	QueryText string `json:"query_text,omitempty"`
	// Sets the "Run as" role for the object.
	RunAsMode RunAsMode `json:"run_as_mode,omitempty"`
	// Name of the schema where this query will be executed.
	Schema string `json:"schema,omitempty"`

	Tags []string `json:"tags,omitempty"`
	// Timestamp when this query was last updated.
	UpdateTime string `json:"update_time,omitempty"`
	// ID of the SQL warehouse attached to the query.
	WarehouseId string `json:"warehouse_id,omitempty"`
}

type QueryBackedValue struct {
	// If specified, allows multiple values to be selected for this parameter.
	MultiValuesOptions *MultiValuesOptions `json:"multi_values_options,omitempty"`
	// UUID of the query that provides the parameter values.
	QueryId string `json:"query_id,omitempty"`
	// List of selected query parameter values.
	Values []string `json:"values,omitempty"`
}

type QueryOptions struct {
	// The name of the catalog to execute this query in.
	Catalog string `json:"catalog,omitempty"`
	// The timestamp when this query was moved to trash. Only present when the
	// `is_archived` property is `true`. Trashed items are deleted after thirty
	// days.
	MovedToTrashAt string `json:"moved_to_trash_at,omitempty"`

	Parameters []Parameter `json:"parameters,omitempty"`
	// The name of the schema to execute this query in.
	Schema string `json:"schema,omitempty"`
}

type QueryParameter struct {
	// Date-range query parameter value. Can only specify one of
	// `dynamic_date_range_value` or `date_range_value`.
	DateRangeValue *DateRangeValue `json:"date_range_value,omitempty"`
	// Date query parameter value. Can only specify one of `dynamic_date_value`
	// or `date_value`.
	DateValue *DateValue `json:"date_value,omitempty"`
	// Dropdown query parameter value.
	EnumValue *EnumValue `json:"enum_value,omitempty"`
	// Literal parameter marker that appears between double curly braces in the
	// query text.
	Name string `json:"name,omitempty"`
	// Numeric query parameter value.
	NumericValue *NumericValue `json:"numeric_value,omitempty"`
	// Query-based dropdown query parameter value.
	QueryBackedValue *QueryBackedValue `json:"query_backed_value,omitempty"`
	// Text query parameter value.
	TextValue *TextValue `json:"text_value,omitempty"`
	// Text displayed in the user-facing parameter widget in the UI.
	Title string `json:"title,omitempty"`
}

type RunAsMode string

const RunAsModeOwner RunAsMode = `OWNER`

const RunAsModeViewer RunAsMode = `VIEWER`

// String representation for [fmt.Print]
func (f *RunAsMode) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *RunAsMode) Set(v string) error {
	switch v {
	case `OWNER`, `VIEWER`:
		*f = RunAsMode(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "OWNER", "VIEWER"`, v)
	}
}

// Values returns all possible values for RunAsMode.
//
// There is no guarantee on the order of the values in the slice.
func (f *RunAsMode) Values() []RunAsMode {
	return []RunAsMode{
		RunAsModeOwner,
		RunAsModeViewer,
	}
}

// Type always returns RunAsMode to satisfy [pflag.Value] interface
func (f *RunAsMode) Type() string {
	return "RunAsMode"
}

type RunAsRole string

const RunAsRoleOwner RunAsRole = `owner`

const RunAsRoleViewer RunAsRole = `viewer`

// String representation for [fmt.Print]
func (f *RunAsRole) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *RunAsRole) Set(v string) error {
	switch v {
	case `owner`, `viewer`:
		*f = RunAsRole(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "owner", "viewer"`, v)
	}
}

// Values returns all possible values for RunAsRole.
//
// There is no guarantee on the order of the values in the slice.
func (f *RunAsRole) Values() []RunAsRole {
	return []RunAsRole{
		RunAsRoleOwner,
		RunAsRoleViewer,
	}
}

// Type always returns RunAsRole to satisfy [pflag.Value] interface
func (f *RunAsRole) Type() string {
	return "RunAsRole"
}

type TextValue struct {
	Value string `json:"value,omitempty"`
}

type User struct {
	Email string `json:"email,omitempty"`

	Id int `json:"id,omitempty"`

	Name string `json:"name,omitempty"`
}

type Visualization struct {
	// The timestamp indicating when the visualization was created.
	CreateTime string `json:"create_time,omitempty"`
	// The display name of the visualization.
	DisplayName string `json:"display_name,omitempty"`
	// UUID identifying the visualization.
	Id string `json:"id,omitempty"`
	// UUID of the query that the visualization is attached to.
	QueryId string `json:"query_id,omitempty"`
	// The visualization options varies widely from one visualization type to
	// the next and is unsupported. Databricks does not recommend modifying
	// visualization options directly.
	SerializedOptions string `json:"serialized_options,omitempty"`
	// The visualization query plan varies widely from one visualization type to
	// the next and is unsupported. Databricks does not recommend modifying the
	// visualization query plan directly.
	SerializedQueryPlan string `json:"serialized_query_plan,omitempty"`
	// The type of visualization: counter, table, funnel, and so on.
	Type string `json:"type,omitempty"`
	// The timestamp indicating when the visualization was updated.
	UpdateTime string `json:"update_time,omitempty"`
}

type Widget struct {
	// The unique ID for this widget.
	Id string `json:"id,omitempty"`

	Options *WidgetOptions `json:"options,omitempty"`
	// The visualization description API changes frequently and is unsupported.
	// You can duplicate a visualization by copying description objects received
	// _from the API_ and then using them to create a new one with a POST
	// request to the same endpoint. Databricks does not recommend constructing
	// ad-hoc visualizations entirely in JSON.
	Visualization *LegacyVisualization `json:"visualization,omitempty"`
	// Unused field.
	Width int `json:"width,omitempty"`
}

type WidgetOptions struct {
	// Timestamp when this object was created
	CreatedAt string `json:"created_at,omitempty"`
	// Custom description of the widget
	Description string `json:"description,omitempty"`
	// Whether this widget is hidden on the dashboard.
	IsHidden bool `json:"isHidden,omitempty"`
	// How parameters used by the visualization in this widget relate to other
	// widgets on the dashboard. Databricks does not recommend modifying this
	// definition in JSON.
	ParameterMappings any `json:"parameterMappings,omitempty"`
	// Coordinates of this widget on a dashboard. This portion of the API
	// changes frequently and is unsupported.
	Position *WidgetPosition `json:"position,omitempty"`
	// Custom title of the widget
	Title string `json:"title,omitempty"`
	// Timestamp of the last time this object was updated.
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Coordinates of this widget on a dashboard. This portion of the API changes
// frequently and is unsupported.
type WidgetPosition struct {
	// reserved for internal use
	AutoHeight bool `json:"autoHeight,omitempty"`
	// column in the dashboard grid. Values start with 0
	Col int `json:"col,omitempty"`
	// row in the dashboard grid. Values start with 0
	Row int `json:"row,omitempty"`
	// width of the widget measured in dashboard grid cells
	SizeX int `json:"sizeX,omitempty"`
	// height of the widget measured in dashboard grid cells
	SizeY int `json:"sizeY,omitempty"`
}
