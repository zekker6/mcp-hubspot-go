package hubspot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/belong-inc/go-hubspot"
)

const defaultRecentDealsLimit = 10

// dealSearchSort mirrors the HubSpot search sort shape.
type dealSearchSort struct {
	PropertyName string `json:"propertyName"`
	Direction    string `json:"direction"`
}

// dealSearchRequest mirrors the HubSpot deals search request body. The SDK in
// v0.10.1 has a typed DealSearchRequest, but its Search method posts to the
// wrong path ("deals/search" instead of "crm/v3/objects/deals/search"), so we
// hit the search endpoint directly via the SDK's generic Post.
type dealSearchRequest struct {
	Sorts []dealSearchSort `json:"sorts,omitempty"`
	Limit int              `json:"limit,omitempty"`
}

// GetDeal fetches a single deal by its ID, optionally requesting additional
// custom properties on top of the SDK's default field set. The returned bytes
// are JSON, ready to be passed through to MCP clients.
func (c *Client) GetDeal(_ context.Context, id string, properties []string) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("deal id is required")
	}

	opt := &hubspot.RequestQueryOption{}
	if len(properties) > 0 {
		opt.CustomProperties = properties
	}

	props := map[string]any{}
	res, err := c.sdk.CRM.Deal.Get(id, &props, opt)
	if err != nil {
		return nil, fmt.Errorf("get deal %s: %w", id, err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal deal response: %w", err)
	}
	return out, nil
}

// GetRecentDeals returns the most recently modified deals sorted by
// hs_lastmodifieddate descending. limit defaults to 10 when non-positive.
//
// Deviation: uses raw HTTP because belong-inc/go-hubspot@v0.10.1's
// DealServiceOp.Search posts to "deals/search" instead of the CRM-prefixed
// "crm/v3/objects/deals/search".
func (c *Client) GetRecentDeals(_ context.Context, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = defaultRecentDealsLimit
	}

	req := dealSearchRequest{
		Sorts: []dealSearchSort{
			{PropertyName: "hs_lastmodifieddate", Direction: "DESCENDING"},
		},
		Limit: limit,
	}

	var raw json.RawMessage
	if err := c.sdk.Post("/crm/v3/objects/deals/search", req, &raw); err != nil {
		return nil, fmt.Errorf("get recent deals: %w", err)
	}
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}

// GetDealPipelines returns all deal pipelines (with their stages) configured
// in HubSpot. Required to interpret raw deal stage IDs surfaced by GetDeal /
// GetRecentDeals. The SDK has no Pipelines client so we hit the v3 endpoint
// directly via the SDK's generic Get.
func (c *Client) GetDealPipelines(_ context.Context) ([]byte, error) {
	var raw json.RawMessage
	if err := c.sdk.Get("/crm/v3/pipelines/deals", &raw, nil); err != nil {
		return nil, fmt.Errorf("get deal pipelines: %w", err)
	}
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}

// CreateDeal creates a new HubSpot deal with the supplied properties. Unlike
// CreateCompany / CreateContact, no duplicate pre-flight is performed - deal
// names are not unique by convention.
func (c *Client) CreateDeal(_ context.Context, properties map[string]any) ([]byte, error) {
	if len(properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}

	res, err := c.sdk.CRM.Deal.Create(&properties)
	if err != nil {
		return nil, fmt.Errorf("create deal: %w", err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal create deal response: %w", err)
	}
	return out, nil
}

// UpdateDeal updates the supplied properties on an existing HubSpot deal.
// Unspecified properties remain untouched.
func (c *Client) UpdateDeal(_ context.Context, id string, properties map[string]any) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("deal id is required")
	}
	if len(properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}

	res, err := c.sdk.CRM.Deal.Update(id, &properties)
	if err != nil {
		return nil, fmt.Errorf("update deal %s: %w", id, err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal update deal response: %w", err)
	}
	return out, nil
}
