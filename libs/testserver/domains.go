package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/domains"
)

const domainNamePrefix = "domains/"

// domainIDPattern is the format the API requires of a client-supplied domain ID.
var domainIDPattern = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

// domainImmutableFields are rejected in an update mask. The `*` wildcard expands to them,
// so the API rejects it as well.
var domainImmutableFields = []string{"tag_key", "parent_domain_id", "name"}

func domainError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body:       map[string]string{"error_code": errorCode, "message": message},
	}
}

func domainNotFound(id string) Response {
	return domainError(http.StatusNotFound, "NOT_FOUND", "Domain not found: "+id)
}

// The SDK sends the Domain body directly (not wrapped in "domain"); the ID to create the
// domain with is the domain_id query parameter.
func (s *FakeWorkspace) DomainCreate(req Request) Response {
	defer s.LockUnlock()()

	var domain domains.Domain
	if err := json.Unmarshal(req.Body, &domain); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if domain.TagKey == "" {
		return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "tag_key is required.")
	}

	for _, existing := range s.Domains {
		if existing.TagKey == domain.TagKey {
			return domainError(http.StatusConflict, "ALREADY_EXISTS", fmt.Sprintf("Domain already exists for tag key '%s'.", domain.TagKey))
		}
	}

	// A subdomain's tag key is its parent's tag key plus one segment.
	if domain.ParentDomainId != "" {
		parent, ok := s.Domains[domain.ParentDomainId]
		if !ok {
			return domainNotFound(domain.ParentDomainId)
		}
		segment, ok := strings.CutPrefix(domain.TagKey, parent.TagKey+"/")
		if !ok || segment == "" || strings.Contains(segment, "/") {
			return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", fmt.Sprintf("Subdomain tag key must be of the form '%s/<name>'.", parent.TagKey))
		}
	} else if strings.Contains(domain.TagKey, "/") {
		return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "Top-level domain tag key must not contain '/'.")
	}

	id := req.URL.Query().Get("domain_id")
	if id == "" {
		id = nextUUID()
	} else {
		if !domainIDPattern.MatchString(id) || strings.HasPrefix(id, "system-") {
			return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", fmt.Sprintf("Invalid domain_id '%s'.", id))
		}
		if _, ok := s.Domains[id]; ok {
			return domainError(http.StatusConflict, "ALREADY_EXISTS", fmt.Sprintf("Domain with id '%s' already exists.", id))
		}
	}

	now := sdktime.New(time.Unix(0, nowNano()).UTC())
	domain.DomainId = id
	domain.Name = domainNamePrefix + id
	domain.CreateTime = now
	domain.UpdateTime = now
	domain.EffectiveDraft = domain.Draft

	s.Domains[id] = domain
	return Response{Body: domainResponse(domain)}
}

// domainResponse drops draft: the API accepts it on write and returns the resolved value
// as effective_draft only.
func domainResponse(domain domains.Domain) domains.Domain {
	domain.Draft = false
	domain.ForceSendFields = slices.DeleteFunc(slices.Clone(domain.ForceSendFields), func(field string) bool {
		return field == "Draft"
	})
	if !domain.EffectiveDraft {
		domain.ForceSendFields = append(domain.ForceSendFields, "EffectiveDraft")
	}
	return domain
}

func (s *FakeWorkspace) DomainGet(id string) Response {
	defer s.LockUnlock()()

	domain, ok := s.Domains[id]
	if !ok {
		return domainNotFound(id)
	}
	return Response{Body: domainResponse(domain)}
}

func (s *FakeWorkspace) DomainList(req Request) Response {
	defer s.LockUnlock()()

	parent := req.URL.Query().Get("parent_domain_id")
	var result []domains.Domain
	for _, domain := range s.Domains {
		if parent != "" && domain.ParentDomainId != parent {
			continue
		}
		result = append(result, domainResponse(domain))
	}
	slices.SortFunc(result, func(a, b domains.Domain) int {
		return strings.Compare(a.TagKey, b.TagKey)
	})
	return Response{Body: domains.ListDomainsResponse{Domains: result}}
}

// update_mask is a query parameter. Only the fields it names are written, and it must not
// name an immutable field.
func (s *FakeWorkspace) DomainUpdate(req Request, id string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Domains[id]
	if !ok {
		return domainNotFound(id)
	}

	var incoming domains.Domain
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	mask := req.URL.Query().Get("update_mask")
	if mask == "" {
		return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", "update_mask is required.")
	}

	for field := range strings.SplitSeq(mask, ",") {
		switch field {
		case "subtitle":
			existing.Subtitle = incoming.Subtitle
		case "description":
			existing.Description = incoming.Description
		case "icon":
			existing.Icon = incoming.Icon
		case "technical_owner_ids":
			existing.TechnicalOwnerIds = incoming.TechnicalOwnerIds
		case "business_owner_ids":
			existing.BusinessOwnerIds = incoming.BusinessOwnerIds
		case "draft":
			existing.Draft = incoming.Draft
			existing.EffectiveDraft = incoming.Draft
		default:
			if field == "*" || slices.Contains(domainImmutableFields, field) {
				return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", fmt.Sprintf("Field '%s' is immutable and cannot be updated.", field))
			}
			return domainError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE", fmt.Sprintf("Invalid update_mask path '%s'.", field))
		}
	}

	existing.UpdateTime = sdktime.New(time.Unix(0, nowNano()).UTC())
	s.Domains[id] = existing
	return Response{Body: domainResponse(existing)}
}

// Deleting a domain deletes its subdomains. It is refused while the domain or one of its
// subdomains still has pages, unless force is set, which deletes those pages too.
func (s *FakeWorkspace) DomainDelete(req Request, id string) Response {
	defer s.LockUnlock()()

	if _, ok := s.Domains[id]; !ok {
		return domainNotFound(id)
	}

	subtree := s.domainSubtree(id)
	var pages []string
	for pageID, page := range s.Pages {
		if slices.Contains(subtree, strings.TrimPrefix(page.Domain, domainNamePrefix)) {
			pages = append(pages, pageID)
		}
	}

	if len(pages) > 0 && req.URL.Query().Get("force") != "true" {
		return domainError(http.StatusBadRequest, "BAD_REQUEST", "Cannot delete a domain that has pages. Delete the pages first or set force to true.")
	}

	for _, pageID := range pages {
		delete(s.Pages, pageID)
	}
	for _, domainID := range subtree {
		delete(s.Domains, domainID)
	}
	return Response{}
}

// domainSubtree returns the ID of the domain and of all its subdomains.
func (s *FakeWorkspace) domainSubtree(id string) []string {
	subtree := []string{id}
	for i := 0; i < len(subtree); i++ {
		for domainID, domain := range s.Domains {
			if domain.ParentDomainId == subtree[i] {
				subtree = append(subtree, domainID)
			}
		}
	}
	return subtree
}
