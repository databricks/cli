package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/databricks/cli/libs/knowledge"
)

func pageError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body:       map[string]string{"error_code": errorCode, "message": message},
	}
}

// pageAccountID is the account the fake reports pages to live in.
const pageAccountID = "aaaaaaaa-bbbb-4ccc-dddd-000000000001"

func pageNotFound(id string) Response {
	return pageError(http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Knowledge page with id '%s' does not exist in account: %s", id, pageAccountID))
}

func pageTimestamp() string {
	return time.Unix(0, nowNano()).UTC().Format(time.RFC3339Nano)
}

// normalizePageSynonyms stores synonyms the way the API does: trimmed, deduplicated and
// sorted.
func normalizePageSynonyms(synonyms []string) ([]string, *Response) {
	if len(synonyms) > 50 {
		resp := pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "A page can have at most 50 synonyms.")
		return nil, &resp
	}
	normalized := make([]string, 0, len(synonyms))
	for _, synonym := range synonyms {
		synonym = strings.TrimSpace(synonym)
		if synonym == "" || len(utf16.Encode([]rune(synonym))) > 200 {
			resp := pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "Each synonym must be 1 to 200 characters.")
			return nil, &resp
		}
		normalized = append(normalized, synonym)
	}
	slices.Sort(normalized)
	normalized = slices.Compact(normalized)
	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}

func validatePageDisplayName(displayName string) (string, *Response) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(utf16.Encode([]rune(displayName))) > 200 {
		resp := pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "display_name must be 1 to 200 characters.")
		return "", &resp
	}
	return displayName, nil
}

// pageDomainID returns the ID in a `domains/{domain_id}` resource name. The API takes the
// resource name, not the bare ID.
func (s *FakeWorkspace) pageDomainID(domain string) (string, *Response) {
	id, ok := strings.CutPrefix(domain, domainNamePrefix)
	if !ok || id == "" {
		resp := pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", fmt.Sprintf("Invalid domain '%s': must be of the form 'domains/{domain_id}'.", domain))
		return "", &resp
	}
	if _, ok := s.Domains[id]; !ok {
		resp := domainNotFound(id)
		return "", &resp
	}
	return id, nil
}

// publishedPageExists reports whether another published page in the domain has the title.
// Titles are unique among the published pages of a domain; drafts may share one.
func (s *FakeWorkspace) publishedPageExists(domain, displayName, exceptID string) bool {
	for id, page := range s.Pages {
		if id != exceptID && page.State == knowledge.StatePublished && page.Domain == domain && page.DisplayName == displayName {
			return true
		}
	}
	return false
}

func pageAlreadyExists(displayName string) Response {
	return pageError(http.StatusConflict, "ALREADY_EXISTS", fmt.Sprintf("A published page named '%s' already exists in this domain.", displayName))
}

// The body is the Page itself (not wrapped in "page").
func (s *FakeWorkspace) PageCreate(req Request) Response {
	defer s.LockUnlock()()

	var page knowledge.Page
	if err := json.Unmarshal(req.Body, &page); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if _, errResp := s.pageDomainID(page.Domain); errResp != nil {
		return *errResp
	}

	displayName, errResp := validatePageDisplayName(page.DisplayName)
	if errResp != nil {
		return *errResp
	}
	page.DisplayName = displayName

	page.Synonyms, errResp = normalizePageSynonyms(page.Synonyms)
	if errResp != nil {
		return *errResp
	}

	if page.State == "" {
		page.State = knowledge.StateDraft
	}
	if page.State == knowledge.StatePublished && s.publishedPageExists(page.Domain, page.DisplayName, "") {
		return pageAlreadyExists(page.DisplayName)
	}

	switch len(page.OwnerIds) {
	case 0:
		page.OwnerIds = []string{userForToken(req.Token).Id}
	case 1:
	default:
		return pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "A page can have at most one owner.")
	}

	id := nextUUID()
	page.Name = knowledge.PageResourceNamePrefix + id
	page.CreateTime = pageTimestamp()
	page.UpdateTime = page.CreateTime
	page.ForceSendFields = nil

	s.Pages[id] = page
	return Response{Body: page}
}

func (s *FakeWorkspace) PageGet(id string) Response {
	defer s.LockUnlock()()

	page, ok := s.Pages[id]
	if !ok {
		return pageNotFound(id)
	}
	return Response{Body: page}
}

// update_mask is a query parameter. The API has no wildcard: only the fields it names are
// written, and the body holds just those fields.
func (s *FakeWorkspace) PageUpdate(req Request, id string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Pages[id]
	if !ok {
		return pageNotFound(id)
	}

	var incoming knowledge.Page
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	mask := req.URL.Query().Get("update_mask")
	if mask == "" {
		return pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "update_mask must not be empty.")
	}

	for field := range strings.SplitSeq(mask, ",") {
		switch field {
		case "display_name":
			displayName, errResp := validatePageDisplayName(incoming.DisplayName)
			if errResp != nil {
				return *errResp
			}
			existing.DisplayName = displayName
		case "domain":
			if _, errResp := s.pageDomainID(incoming.Domain); errResp != nil {
				return *errResp
			}
			existing.Domain = incoming.Domain
		case "body":
			existing.Body = incoming.Body
		case "synonyms":
			synonyms, errResp := normalizePageSynonyms(incoming.Synonyms)
			if errResp != nil {
				return *errResp
			}
			existing.Synonyms = synonyms
		case "description":
			existing.Description = incoming.Description
		case "state":
			if incoming.State != knowledge.StateDraft && incoming.State != knowledge.StatePublished {
				return pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "state must be DRAFT or PUBLISHED when selected by update_mask.")
			}
			existing.State = incoming.State
		case "owner_ids":
			if len(incoming.OwnerIds) != 1 {
				return pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "owner_ids must contain exactly one ID when selected by update_mask.")
			}
			existing.OwnerIds = incoming.OwnerIds
		default:
			return pageError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", fmt.Sprintf("Unsupported update_mask path '%s'.", field))
		}
	}

	if existing.State == knowledge.StatePublished && s.publishedPageExists(existing.Domain, existing.DisplayName, id) {
		return pageAlreadyExists(existing.DisplayName)
	}

	existing.UpdateTime = pageTimestamp()
	s.Pages[id] = existing
	return Response{Body: existing}
}

func (s *FakeWorkspace) PageDelete(id string) Response {
	defer s.LockUnlock()()

	if _, ok := s.Pages[id]; !ok {
		return pageNotFound(id)
	}
	delete(s.Pages, id)
	return Response{}
}
