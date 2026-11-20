// Package knowledge is a client for the Knowledge Pages API (/api/2.0/knowledge/pages).
//
// The API is public but excluded from the Databricks SDKs, so the request and response
// types are declared here. They follow the shape the SDK generator produces, so that this
// package can be replaced with the SDK's `service/knowledge` once it exists.
package knowledge

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go/marshal"
)

// PageResourceNamePrefix is the prefix of a page's resource name: `pages/{page_id}`.
const PageResourceNamePrefix = "pages/"

// A knowledge page: a domain-scoped governance document describing a concept or term in
// the account.
type Page struct {
	// Page body as Markdown text.
	Body string `json:"body,omitempty"`
	// Server-assigned creation time, formatted as an RFC 3339 timestamp.
	CreateTime string `json:"create_time,omitempty"`
	// Short description of the page.
	Description string `json:"description,omitempty"`
	// Human-readable title, 1 to 200 characters. Published titles must be unique
	// within the domain.
	DisplayName string `json:"display_name"`
	// Resource name of the domain this page belongs to; format `domains/{domain_id}`.
	Domain string `json:"domain"`
	// Resource name in the form `pages/{page_id}`, where `page_id` is a
	// server-generated UUID.
	Name string `json:"name,omitempty"`
	// Decimal principal ID of the page's single owner: a user, group, or service
	// principal. Defaults to the caller on create.
	OwnerIds []string `json:"owner_ids,omitempty"`
	// Lifecycle state. Defaults to `DRAFT` on create.
	State State `json:"state,omitempty"`
	// Alternate names or abbreviations. At most 50 entries, each 1 to 200 characters.
	Synonyms []string `json:"synonyms,omitempty"`
	// Server-assigned last-update time, formatted as an RFC 3339 timestamp.
	UpdateTime string `json:"update_time,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func (s *Page) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Page) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// Lifecycle state of a Page.
type State string

// The page is a work-in-progress draft.
const StateDraft State = `DRAFT`

// The page is published and discoverable.
const StatePublished State = `PUBLISHED`

// String representation for [fmt.Print]
func (f *State) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *State) Set(v string) error {
	switch v {
	case `DRAFT`, `PUBLISHED`:
		*f = State(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "DRAFT", "PUBLISHED"`, v)
	}
}

// Values returns all possible values for State.
//
// There is no guarantee on the order of the values in the slice.
func (f *State) Values() []State {
	return []State{
		StateDraft,
		StatePublished,
	}
}

// Type always returns State to satisfy [pflag.Value] interface
func (f *State) Type() string {
	return "State"
}
