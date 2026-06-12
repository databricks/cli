package main

import (
	"fmt"
	"slices"
)

// This file defines the CLI-side view of the "commands" block of the cli.json
// spec. The structs mirror, field-for-field, the dot-paths that the templates
// (templates/*.tmpl, derived from genkit's cliv0 templates) access.
//
// Design rules that keep output byte-for-byte identical to genkit:
//   - Casings (Kebab/Pascal/Snake/Camel/Constant/Title) are derived at render
//     time from the single stored `name` via the ported genkit name functions
//     (names.go); only leaf references (PascalRef, FieldRef, NamedIdMapJSON,
//     LROMethodRef) carry pre-resolved casing strings.
//   - Comment-derived text (Summary, Comment, CliComment, HasComment) is NOT
//     stored. It is derived here from the raw Description with the same pure
//     helpers genkit uses (see comment.go), so it cannot drift.
//   - Cross-references that form cycles in genkit's graph (ParentService,
//     Subservices, RequestBodyField identity) are carried by name/reference and
//     re-linked in Resolve() after decoding.

// CommandsBlock is the batch-level root of the commands section.
type CommandsBlock struct {
	Services            []*ServiceJSON   `json:"services,omitempty"`
	WorkspaceDocsGroups []*DocsGroupJSON `json:"workspace_docs_groups,omitempty"`
	AccountDocsGroups   []*DocsGroupJSON `json:"account_docs_groups,omitempty"`

	byID map[string]*ServiceJSON
}

// DocsGroupJSON is the docs-group shape read by groups-*.go.tmpl.
type DocsGroupJSON struct {
	Key         string `json:"key,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// NameVariants is the value returned by (Service|Method).TrimPrefix; templates
// access its SnakeName/KebabName/PascalName fields.
type NameVariants struct {
	SnakeName  string `json:"snake_name,omitempty"`
	KebabName  string `json:"kebab_name,omitempty"`
	PascalName string `json:"pascal_name,omitempty"`
}

// PackageRef backs .Package.Name.
type PackageRef struct {
	Name string `json:"name,omitempty"`
}

// PascalRef is a thin reference carrying only PascalName (e.g. request types).
type PascalRef struct {
	PascalName string `json:"pascal_name,omitempty"`
}

// NamedIdMapJSON backs $.List.NamedIdMap.PascalName.
type NamedIdMapJSON struct {
	PascalName string `json:"pascal_name,omitempty"`
}

// EnumEntryJSON backs printArray / supportedValues / wait success states.
type EnumEntryJSON struct {
	Content string `json:"content,omitempty"`
}

// ServiceListRefJSON is the slimmed view of a service's List method that the
// template reads via $.List (NamedIdMap, IsLegacyEmptyRequest, Request).
type ServiceListRefJSON struct {
	NamedIdMap           *NamedIdMapJSON `json:"named_id_map,omitempty"`
	IsLegacyEmptyRequest bool            `json:"is_legacy_empty_request,omitempty"`
	Request              *PascalRef      `json:"request,omitempty"`
}

// ServiceJSON is the render context for service.go.tmpl.
type ServiceJSON struct {
	ID string `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	Description string      `json:"description,omitempty"`
	Package     *PackageRef `json:"package,omitempty"`

	IsAccounts     bool `json:"is_accounts,omitempty"`
	HasParent      bool `json:"has_parent,omitempty"`
	IsDataPlane    bool `json:"is_data_plane,omitempty"`
	HasSubservices bool `json:"has_subservices,omitempty"`
	IsHiddenCLI    bool `json:"is_hidden_cli,omitempty"`

	DocsGroup string `json:"docs_group,omitempty"`

	LaunchStage           string `json:"launch_stage,omitempty"`
	CLILaunchStageLabel   string `json:"cli_launch_stage_label,omitempty"`
	CLILaunchStageBanner  string `json:"cli_launch_stage_banner,omitempty"`
	CLILaunchStageDisplay string `json:"cli_launch_stage_display,omitempty"`

	ParentServiceID string              `json:"parent_service_id,omitempty"`
	SubserviceIDs   []string            `json:"subservice_ids,omitempty"`
	List            *ServiceListRefJSON `json:"list,omitempty"`
	Methods         []*MethodJSON       `json:"methods,omitempty"`

	// Re-linked in Resolve().
	ParentService *ServiceJSON   `json:"-"`
	Subservices   []*ServiceJSON `json:"-"`
}

// Casing methods derive from Name via the ported genkit name functions
// (names.go), so the producer doesn't denormalize every variant into cli.json.
func (s *ServiceJSON) KebabName() string  { return kebabName(s.Name) }
func (s *ServiceJSON) SnakeName() string  { return snakeName(s.Name) }
func (s *ServiceJSON) PascalName() string { return pascalName(s.Name) }
func (s *ServiceJSON) TitleName() string  { return titleName(s.Name) }

// TrimPrefix mirrors genkit Named.TrimPrefix: trim the prefix from the
// camel-cased name, then expose the casings of the result.
func (s *ServiceJSON) TrimPrefix(prefix string) NameVariants {
	tn := trimPrefixName(s.Name, prefix)
	return NameVariants{SnakeName: snakeName(tn), KebabName: kebabName(tn), PascalName: pascalName(tn)}
}

// Summary is the first sentence of the description; matches genkit Named.Summary.
func (s *ServiceJSON) Summary() string { return summarize(s.Description) }

// Comment wraps the description into a multi-line comment; matches Named.Comment.
func (s *ServiceJSON) Comment(prefix string, maxLen int) string {
	return commentWrap(s.Description, prefix, maxLen)
}

// HasInputOnlyPaths is true when any of the service's methods has a non-empty
// InputOnlyPaths. service.go.tmpl uses it to gate the libs/inputonly import:
// importing it unconditionally would leave it unused on services that don't
// strip anything.
func (s *ServiceJSON) HasInputOnlyPaths() bool {
	for _, m := range s.Methods {
		if len(m.InputOnlyPaths) > 0 {
			return true
		}
	}
	return false
}

// MethodJSON is the render context for one command (a method of a service).
type MethodJSON struct {
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`
	// Summary is the method's own summary, pre-resolved by the producer (unlike
	// Service/Field, Method.Summary() reads a separate field, not the first
	// sentence of the description).
	Summary string `json:"summary,omitempty"`
	Path    string `json:"path,omitempty"`

	IsLegacyEmptyRequest           bool `json:"is_legacy_empty_request,omitempty"`
	CanUseJson                     bool `json:"can_use_json,omitempty"`
	MustUseJson                    bool `json:"must_use_json,omitempty"`
	IsJsonOnly                     bool `json:"is_json_only,omitempty"`
	IsCrudCreate                   bool `json:"is_crud_create,omitempty"`
	IsCrudRead                     bool `json:"is_crud_read,omitempty"`
	IsHiddenCLI                    bool `json:"is_hidden_cli,omitempty"`
	IsResponseByteStream           bool `json:"is_response_byte_stream,omitempty"`
	HasRequiredPositionalArguments bool `json:"has_required_positional_arguments,omitempty"`

	LaunchStage           string `json:"launch_stage,omitempty"`
	CLILaunchStageLabel   string `json:"cli_launch_stage_label,omitempty"`
	CLILaunchStageBanner  string `json:"cli_launch_stage_banner,omitempty"`
	CLILaunchStageDisplay string `json:"cli_launch_stage_display,omitempty"`

	Request           *EntityJSON `json:"request,omitempty"`
	RequestBodyField  *FieldJSON  `json:"request_body_field,omitempty"`
	Response          *EntityJSON `json:"response,omitempty"`
	ResponseBodyField *PascalRef  `json:"response_body_field,omitempty"`

	AllFields                   []*FieldJSON `json:"all_fields,omitempty"`
	RequiredPositionalArguments []*FieldJSON `json:"required_positional_arguments,omitempty"`

	Pagination           *PaginationJSON `json:"pagination,omitempty"`
	Wait                 *WaitJSON       `json:"wait,omitempty"`
	LongRunningOperation *LROJSON        `json:"long_running_operation,omitempty"`

	// InputOnlyPaths is the sorted set of dotted JSON paths in the method's
	// response type that are annotated INPUT_ONLY in cli.json's schemas
	// block. Populated by populateInputOnlyPaths after Resolve(); empty for
	// methods whose render site is not the standard sync path.
	InputOnlyPaths []string `json:"-"`
}

func (m *MethodJSON) KebabName() string  { return kebabName(m.Name) }
func (m *MethodJSON) SnakeName() string  { return snakeName(m.Name) }
func (m *MethodJSON) PascalName() string { return pascalName(m.Name) }
func (m *MethodJSON) CamelName() string  { return camelName(m.Name) }

// CliComment mirrors genkit Method.CliComment (method.go:139): prepend the
// summary as its own paragraph when it differs from the description. Summary is
// a pre-resolved field (see the struct), not derived from the description.
func (m *MethodJSON) CliComment(prefix string, maxLen int) string {
	description := m.Description
	if m.Summary != "" && m.Summary != m.Description {
		description = fmt.Sprintf("%s\n\n%s", m.Summary, description)
	}
	return commentWrap(description, prefix, maxLen)
}

// FieldJSON is one request field (flag or positional argument).
type FieldJSON struct {
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`

	Required           bool `json:"required,omitempty"`
	IsPath             bool `json:"is_path,omitempty"`
	IsQuery            bool `json:"is_query,omitempty"`
	IsComputed         bool `json:"is_computed,omitempty"`
	IsOutputOnly       bool `json:"is_output_only,omitempty"`
	IsRequestBodyField bool `json:"is_request_body_field,omitempty"`
	IsOptionalObject   bool `json:"is_optional_object,omitempty"`

	Entity *EntityJSON `json:"entity,omitempty"`
}

func (f *FieldJSON) KebabName() string    { return kebabName(f.Name) }
func (f *FieldJSON) PascalName() string   { return pascalName(f.Name) }
func (f *FieldJSON) CamelName() string    { return camelName(f.Name) }
func (f *FieldJSON) ConstantName() string { return constantName(f.Name) }

// Summary matches genkit Named.Summary.
func (f *FieldJSON) Summary() string { return summarize(f.Description) }

// HasComment matches genkit Named.HasComment.
func (f *FieldJSON) HasComment() bool { return f.Description != "" }

// Comment wraps the description; matches Named.Comment.
func (f *FieldJSON) Comment(prefix string, maxLen int) string {
	return commentWrap(f.Description, prefix, maxLen)
}

// EntityJSON is a value type (message, primitive, enum, array, map). The same
// struct backs both request entities (which expose required-field collections)
// and leaf field types (which expose only type predicates).
type EntityJSON struct {
	PascalName string `json:"pascal_name,omitempty"`

	IsObject    bool `json:"is_object,omitempty"`
	IsAny       bool `json:"is_any,omitempty"`
	IsString    bool `json:"is_string,omitempty"`
	IsBool      bool `json:"is_bool,omitempty"`
	IsInt       bool `json:"is_int,omitempty"`
	IsInt64     bool `json:"is_int64,omitempty"`
	IsFloat64   bool `json:"is_float64,omitempty"`
	IsDuration  bool `json:"is_duration,omitempty"`
	IsTimestamp bool `json:"is_timestamp,omitempty"`
	IsFieldMask bool `json:"is_field_mask,omitempty"`

	// IsEmptyResponse marks a response that carries nothing renderable
	// (google.protobuf.Empty or a legacy named-but-fieldless response).
	IsEmptyResponse bool `json:"is_empty_response,omitempty"`

	ArrayValue *EntityJSON      `json:"array_value,omitempty"`
	MapValue   *EntityJSON      `json:"map_value,omitempty"`
	Enum       []*EnumEntryJSON `json:"enum,omitempty"`

	HasFieldMask                 bool `json:"has_field_mask,omitempty"`
	HasRequiredRequestBodyFields bool `json:"has_required_request_body_fields,omitempty"`

	RequiredFields            []*FieldJSON `json:"required_fields,omitempty"`
	RequiredInUrlFields       []*FieldJSON `json:"required_in_url_fields,omitempty"`
	RequiredRequestBodyFields []*FieldJSON `json:"required_request_body_fields,omitempty"`
}

// PaginationJSON backs .Pagination. Pagination fields are matched against
// AllFields by Name in the template, so no pointer identity is required.
type PaginationJSON struct {
	Token      *TokenJSON `json:"token,omitempty"`
	Offset     *FieldJSON `json:"offset,omitempty"`
	Limit      *FieldJSON `json:"limit,omitempty"`
	MaxResults *FieldJSON `json:"max_results,omitempty"`
}

// TokenJSON backs .Pagination.Token; only PollField is read.
type TokenJSON struct {
	PollField *FieldJSON `json:"poll_field,omitempty"`
}

// FieldRef is a thin reference for wait status/message path elements.
type FieldRef struct {
	PascalName string `json:"pascal_name,omitempty"`
}

// WaitJSON backs .Wait. Cross-references to the poll method are flattened to
// the only datum templates read (its response type's PascalName).
type WaitJSON struct {
	Success            []*EnumEntryJSON `json:"success,omitempty"`
	Timeout            int              `json:"timeout,omitempty"`
	ComplexMessagePath bool             `json:"complex_message_path,omitempty"`
	Poll               *PollJSON        `json:"poll,omitempty"`
	StatusPath         []*FieldRef      `json:"status_path,omitempty"`
	MessagePath        []*FieldRef      `json:"message_path,omitempty"`
	MessagePathHead    *FieldRef        `json:"message_path_head,omitempty"`
}

// PollJSON backs .Wait.Poll.Response.PascalName.
type PollJSON struct {
	Response *PascalRef `json:"response,omitempty"`
}

// LROJSON backs .LongRunningOperation.
type LROJSON struct {
	GetOperation *LROMethodRef `json:"get_operation,omitempty"`
	ResponseType *EntityRef    `json:"response_type,omitempty"`
}

// LROMethodRef backs .LongRunningOperation.GetOperation.{KebabName,PascalName,Request.PascalName}.
type LROMethodRef struct {
	KebabName  string     `json:"kebab_name,omitempty"`
	PascalName string     `json:"pascal_name,omitempty"`
	Request    *PascalRef `json:"request,omitempty"`
}

// EntityRef backs .LongRunningOperation.ResponseType.IsEmptyResponse.
//
// The producer computes IsEmptyResponse as "google.protobuf.Empty OR legacy
// named-but-fieldless response", while genkit's LRO branch checked only the
// former. No current LRO response is a legacy empty message (regeneration is
// byte-identical), but an LRO that gains one would render its empty-response
// branch where genkit would not.
type EntityRef struct {
	IsEmptyResponse bool `json:"is_empty_response,omitempty"`
}

// Resolve re-links cross-references after JSON decoding: builds the service
// index, wires ParentService/Subservices pointers, and restores the pointer
// identity between each method's RequestBodyField and its AllFields entry (the
// service.go.tmpl `eq . $method.RequestBodyField` check relies on identity).
// Any unresolved reference is an error: silently dropping a subservice or
// keeping the decoded duplicate RequestBodyField pointer would render wrong
// output (missing subcommands, request-body flags generated as plain flags)
// without failing generation.
func (c *CommandsBlock) Resolve() error {
	c.byID = make(map[string]*ServiceJSON, len(c.Services))
	for _, s := range c.Services {
		if _, ok := c.byID[s.ID]; ok {
			return fmt.Errorf("duplicate service id %q", s.ID)
		}
		c.byID[s.ID] = s
	}
	for _, s := range c.Services {
		if s.ParentServiceID != "" {
			s.ParentService = c.byID[s.ParentServiceID]
			if s.ParentService == nil {
				return fmt.Errorf("service %s: unknown parent service id %q", s.Name, s.ParentServiceID)
			}
		}
		s.Subservices = s.Subservices[:0]
		for _, id := range s.SubserviceIDs {
			sub := c.byID[id]
			if sub == nil {
				return fmt.Errorf("service %s: unknown subservice id %q", s.Name, id)
			}
			s.Subservices = append(s.Subservices, sub)
		}
		for _, m := range s.Methods {
			if m.RequestBodyField == nil {
				continue
			}
			i := slices.IndexFunc(m.AllFields, func(f *FieldJSON) bool {
				return f.Name == m.RequestBodyField.Name
			})
			if i < 0 {
				return fmt.Errorf("service %s, method %s: request body field %q has no all_fields entry", s.Name, m.Name, m.RequestBodyField.Name)
			}
			m.RequestBodyField = m.AllFields[i]
		}
	}
	return nil
}
