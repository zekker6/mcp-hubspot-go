package hubspot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/belong-inc/go-hubspot"
)

const (
	defaultRecentCompaniesLimit = 10
	defaultSearchCompaniesLimit = 10
)

// companySearchRequest mirrors the HubSpot companies search request body. The
// SDK's typed CompanySearchResponse strips the paging field, so SearchCompanies
// posts directly via the SDK's generic Post and returns raw JSON to preserve
// paging.next.after for cursor round-trips.
type companySearchRequest struct {
	Query      string   `json:"query,omitempty"`
	Properties []string `json:"properties,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	After      string   `json:"after,omitempty"`
}

// GetCompany fetches a single company by its ID, optionally requesting
// additional custom properties on top of the SDK's default field set. The
// returned bytes are JSON, ready to be passed through to MCP clients.
func (c *Client) GetCompany(_ context.Context, id string, properties []string) ([]byte, error) {
	opt := &hubspot.RequestQueryOption{}
	if len(properties) > 0 {
		opt.CustomProperties = properties
	}

	props := map[string]any{}
	res, err := c.sdk.CRM.Company.Get(id, &props, opt)
	if err != nil {
		return nil, fmt.Errorf("get company %s: %w", id, err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal company response: %w", err)
	}
	return out, nil
}

// GetRecentCompanies returns the most recently modified companies sorted by
// hs_lastmodifieddate descending. limit defaults to 10 when non-positive.
func (c *Client) GetRecentCompanies(_ context.Context, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = defaultRecentCompaniesLimit
	}

	req := &hubspot.CompanySearchRequest{
		SearchOptions: hubspot.SearchOptions{
			Limit: limit,
			Sorts: []hubspot.Sort{
				{PropertyName: "hs_lastmodifieddate", Direction: hubspot.Desc},
			},
		},
	}

	res, err := c.sdk.CRM.Company.Search(req)
	if err != nil {
		return nil, fmt.Errorf("get recent companies: %w", err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal recent companies response: %w", err)
	}
	return out, nil
}

// SearchCompanies runs HubSpot's CRM Search API for companies using a
// free-text query, returning matches in HubSpot's relevance order (no sort).
// limit is clamped to 1..maxSearchLimit (defaulting to
// defaultSearchCompaniesLimit when non-positive). after is forwarded as-is for
// paging - HubSpot's paging.next.after is a string and is round-tripped
// unchanged.
//
// Deviation: bypasses the SDK's typed CompanySearchResponse, which omits the
// paging field, and posts via the SDK's generic Post to surface raw JSON
// (mirrors SearchDeals/SearchTickets so all four search methods preserve the
// cursor end-to-end).
func (c *Client) SearchCompanies(_ context.Context, query string, limit int, properties []string, after string) ([]byte, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if limit <= 0 {
		limit = defaultSearchCompaniesLimit
	} else if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	req := companySearchRequest{
		Query:      query,
		Properties: properties,
		Limit:      limit,
		After:      after,
	}

	var raw json.RawMessage
	if err := c.sdk.Post("/crm/v3/objects/companies/search", req, &raw); err != nil {
		return nil, fmt.Errorf("search companies: %w", err)
	}
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}

// GetCompanyActivity returns the engagements (notes, calls, emails, tasks,
// meetings) associated with the given company ID. The HubSpot SDK has no
// typed wrapper for the engagements API, so this calls the v1 endpoint via
// the SDK's generic Get and passes the raw JSON through unchanged.
func (c *Client) GetCompanyActivity(_ context.Context, companyID string) ([]byte, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company id is required")
	}

	var dst json.RawMessage
	path := "/engagements/v1/engagements/associated/COMPANY/" + companyID + "/paged"
	if err := c.sdk.Get(path, &dst, nil); err != nil {
		return nil, fmt.Errorf("get company %s activity: %w", companyID, err)
	}
	if len(dst) == 0 {
		return []byte("null"), nil
	}
	return dst, nil
}

// CreateCompany creates a new HubSpot company. It performs a pre-flight
// name-exact-match search; if any company already has the same name, the
// create is skipped and a {"duplicate": true, "matches": [...]} payload is
// returned instead. Search failures are surfaced - they do NOT fall through
// to a create attempt.
func (c *Client) CreateCompany(_ context.Context, properties map[string]any) ([]byte, error) {
	if len(properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}
	name, _ := properties["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("properties.name is required")
	}

	existing, err := c.sdk.CRM.Company.SearchByName(name)
	if err != nil {
		return nil, fmt.Errorf("search company by name %q: %w", name, err)
	}
	if existing != nil && len(existing.Results) > 0 {
		out, err := json.Marshal(map[string]any{
			"duplicate": true,
			"matches":   existing.Results,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal duplicate company response: %w", err)
		}
		return out, nil
	}

	res, err := c.sdk.CRM.Company.Create(&properties)
	if err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal create company response: %w", err)
	}
	return out, nil
}

// UpdateCompany updates the supplied properties on an existing HubSpot
// company. Unspecified properties remain untouched.
func (c *Client) UpdateCompany(_ context.Context, id string, properties map[string]any) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("company id is required")
	}
	if len(properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}

	res, err := c.sdk.CRM.Company.Update(id, &properties)
	if err != nil {
		return nil, fmt.Errorf("update company %s: %w", id, err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal update company response: %w", err)
	}
	return out, nil
}
