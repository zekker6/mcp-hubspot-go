package hubspot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/belong-inc/go-hubspot"
)

const defaultRecentContactsLimit = 10

// GetContact fetches a single contact by its ID, optionally requesting
// additional custom properties on top of the SDK's default field set. The
// returned bytes are JSON, ready to be passed through to MCP clients.
func (c *Client) GetContact(_ context.Context, id string, properties []string) ([]byte, error) {
	opt := &hubspot.RequestQueryOption{}
	if len(properties) > 0 {
		opt.CustomProperties = properties
	}

	props := map[string]any{}
	res, err := c.sdk.CRM.Contact.Get(id, &props, opt)
	if err != nil {
		return nil, fmt.Errorf("get contact %s: %w", id, err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal contact response: %w", err)
	}
	return out, nil
}

// GetRecentContacts returns the most recently modified contacts sorted by
// lastmodifieddate descending. limit defaults to 10 when non-positive.
func (c *Client) GetRecentContacts(_ context.Context, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = defaultRecentContactsLimit
	}

	req := &hubspot.ContactSearchRequest{
		SearchOptions: hubspot.SearchOptions{
			Limit: limit,
			Sorts: []hubspot.Sort{
				{PropertyName: "lastmodifieddate", Direction: hubspot.Desc},
			},
		},
	}

	res, err := c.sdk.CRM.Contact.Search(req)
	if err != nil {
		return nil, fmt.Errorf("get recent contacts: %w", err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal recent contacts response: %w", err)
	}
	return out, nil
}

// CreateContact creates a new HubSpot contact. It performs a pre-flight
// email-exact-match search; if any contact already has the same email, the
// create is skipped and a {"duplicate": true, "matches": [...]} payload is
// returned instead. Search failures are surfaced - they do NOT fall through
// to a create attempt.
func (c *Client) CreateContact(_ context.Context, properties map[string]any) ([]byte, error) {
	if len(properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}
	email, _ := properties["email"].(string)
	if email == "" {
		return nil, fmt.Errorf("properties.email is required")
	}

	existing, err := c.sdk.CRM.Contact.SearchByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("search contact by email %q: %w", email, err)
	}
	if existing != nil && len(existing.Results) > 0 {
		out, err := json.Marshal(map[string]any{
			"duplicate": true,
			"matches":   existing.Results,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal duplicate contact response: %w", err)
		}
		return out, nil
	}

	res, err := c.sdk.CRM.Contact.Create(&properties)
	if err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal create contact response: %w", err)
	}
	return out, nil
}

// UpdateContact updates the supplied properties on an existing HubSpot
// contact. Unspecified properties remain untouched.
func (c *Client) UpdateContact(_ context.Context, id string, properties map[string]any) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("contact id is required")
	}
	if len(properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}

	res, err := c.sdk.CRM.Contact.Update(id, &properties)
	if err != nil {
		return nil, fmt.Errorf("update contact %s: %w", id, err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal update contact response: %w", err)
	}
	return out, nil
}
