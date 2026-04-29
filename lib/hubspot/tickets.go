package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const (
	defaultTicketsLimit       = 50
	defaultSearchTicketsLimit = 10

	ticketsCriteriaDefault = "default"
	ticketsCriteriaClosed  = "Closed"
)

// nowMinus24h is overridable for tests so the "default" criteria filter value
// is reproducible without time-mocking the whole package.
var nowMinus24h = func() time.Time { return time.Now().Add(-24 * time.Hour) }

// ticketSearchFilter mirrors the HubSpot search API filter shape.
type ticketSearchFilter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

// ticketSearchFilterGroup mirrors a single filter group.
type ticketSearchFilterGroup struct {
	Filters []ticketSearchFilter `json:"filters"`
}

// ticketSearchSort mirrors the HubSpot search sort shape.
type ticketSearchSort struct {
	PropertyName string `json:"propertyName"`
	Direction    string `json:"direction"`
}

// ticketSearchRequest mirrors the HubSpot tickets search request body. The SDK
// in v0.10.1 has a typed CrmTicketSearchRequest, but it does not expose
// top-level sorts/limit/properties - the actual API requires those at the top
// level, so we hit the search endpoint directly via the SDK's generic Post.
type ticketSearchRequest struct {
	FilterGroups []ticketSearchFilterGroup `json:"filterGroups,omitempty"`
	Query        string                    `json:"query,omitempty"`
	Sorts        []ticketSearchSort        `json:"sorts,omitempty"`
	Properties   []string                  `json:"properties,omitempty"`
	Limit        int                       `json:"limit,omitempty"`
	After        string                    `json:"after,omitempty"`
}

var ticketSearchProperties = []string{
	"subject",
	"content",
	"hs_pipeline",
	"hs_pipeline_stage",
	"hs_ticket_status",
	"status",
	"hs_ticket_priority",
	"createdate",
	"closedate",
	"hs_lastmodifieddate",
}

// GetTickets returns HubSpot tickets matching the given criteria. Filter shapes:
//
//   - "default": tickets where closedate OR hs_lastmodifieddate is GT now-24h
//   - "Closed":  tickets where hs_pipeline_stage equals stage ID "4" OR "Closed"
//
// limit defaults to 50 when non-positive.
func (c *Client) GetTickets(_ context.Context, criteria string, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = defaultTicketsLimit
	}

	groups, err := ticketFilterGroups(criteria)
	if err != nil {
		return nil, err
	}

	req := ticketSearchRequest{
		FilterGroups: groups,
		Sorts: []ticketSearchSort{
			{PropertyName: "hs_lastmodifieddate", Direction: "DESCENDING"},
		},
		Properties: ticketSearchProperties,
		Limit:      limit,
	}

	var raw json.RawMessage
	if err := c.sdk.Post("/crm/v3/objects/tickets/search", req, &raw); err != nil {
		return nil, fmt.Errorf("search tickets (criteria=%s): %w", criteria, err)
	}
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}

// SearchTickets runs a free-text search against the HubSpot tickets search
// endpoint and returns the raw JSON response. No filter groups and no sort are
// set so HubSpot orders results by relevance. limit is clamped: <=0 falls back
// to defaultSearchTicketsLimit, values above maxSearchLimit are capped. after
// is forwarded as-is for paging.
//
// Deviation: uses raw HTTP because belong-inc/go-hubspot@v0.10.1 does not
// expose a typed search method for tickets that posts to the CRM-prefixed
// search path.
func (c *Client) SearchTickets(_ context.Context, query string, limit int, properties []string, after string) ([]byte, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if limit <= 0 {
		limit = defaultSearchTicketsLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	req := ticketSearchRequest{
		Query:      query,
		Properties: properties,
		Limit:      limit,
		After:      after,
	}

	var raw json.RawMessage
	if err := c.sdk.Post("/crm/v3/objects/tickets/search", req, &raw); err != nil {
		return nil, fmt.Errorf("search tickets: %w", err)
	}
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}

func ticketFilterGroups(criteria string) ([]ticketSearchFilterGroup, error) {
	switch criteria {
	case ticketsCriteriaDefault, "":
		oneDayAgo := nowMinus24h().UTC().Format("2006-01-02T15:04:05Z")
		return []ticketSearchFilterGroup{
			{Filters: []ticketSearchFilter{{PropertyName: "closedate", Operator: "GT", Value: oneDayAgo}}},
			{Filters: []ticketSearchFilter{{PropertyName: "hs_lastmodifieddate", Operator: "GT", Value: oneDayAgo}}},
		}, nil
	case ticketsCriteriaClosed:
		return []ticketSearchFilterGroup{
			{Filters: []ticketSearchFilter{{PropertyName: "hs_pipeline_stage", Operator: "EQ", Value: "4"}}},
			{Filters: []ticketSearchFilter{{PropertyName: "hs_pipeline_stage", Operator: "EQ", Value: "Closed"}}},
		}, nil
	default:
		return nil, fmt.Errorf("invalid criteria %q: must be %q or %q", criteria, ticketsCriteriaDefault, ticketsCriteriaClosed)
	}
}

// ticketAssociationsResponse mirrors the v4 associations API shape we care
// about. HubSpot returns `toObjectId` (numeric) for v4 results; older shapes
// use `id`, so we accept both.
type ticketAssociationsResponse struct {
	Results []struct {
		ToObjectID json.RawMessage `json:"toObjectId,omitempty"`
		ID         json.RawMessage `json:"id,omitempty"`
	} `json:"results"`
}

type ticketThreadMessage = map[string]any

type ticketThreadMessagesResponse struct {
	Results []ticketThreadMessage `json:"results"`
}

type ticketThreadOutput struct {
	ID       string                `json:"id"`
	Messages []ticketThreadMessage `json:"messages"`
}

type ticketThreadsOutput struct {
	TicketID      string               `json:"ticket_id"`
	Threads       []ticketThreadOutput `json:"threads"`
	TotalThreads  int                  `json:"total_threads"`
	TotalMessages int                  `json:"total_messages"`
}

// GetTicketConversationThreads returns conversation threads (with their
// messages) associated with a ticket:
//
//  1. GET /crm/v4/objects/tickets/<id>/associations/conversation - list
//     associated conversation IDs (`toObjectId` in the v4 response).
//  2. GET /conversations/v3/conversations/threads/<id>/messages - fetch
//     messages for each thread; only entries whose `type == "MESSAGE"` are
//     kept (system messages are filtered out).
//
// Returned shape: {ticket_id, threads:[{id,messages:[...]}], total_threads, total_messages}.
func (c *Client) GetTicketConversationThreads(_ context.Context, ticketID string) ([]byte, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket id is required")
	}

	assocPath := "/crm/v4/objects/tickets/" + url.PathEscape(ticketID) + "/associations/conversation"
	var assoc ticketAssociationsResponse
	if err := c.sdk.Get(assocPath, &assoc, nil); err != nil {
		return nil, fmt.Errorf("list ticket %s conversation associations: %w", ticketID, err)
	}

	threadIDs := extractThreadIDs(assoc)

	out := ticketThreadsOutput{
		TicketID: ticketID,
		Threads:  []ticketThreadOutput{},
	}

	for _, threadID := range threadIDs {
		msgPath := "/conversations/v3/conversations/threads/" + url.PathEscape(threadID) + "/messages"
		var msgs ticketThreadMessagesResponse
		if err := c.sdk.Get(msgPath, &msgs, nil); err != nil {
			return nil, fmt.Errorf("get messages for thread %s on ticket %s: %w", threadID, ticketID, err)
		}

		thread := ticketThreadOutput{ID: threadID, Messages: []ticketThreadMessage{}}
		for _, m := range msgs.Results {
			if t, _ := m["type"].(string); t == "MESSAGE" {
				thread.Messages = append(thread.Messages, m)
			}
		}
		out.Threads = append(out.Threads, thread)
		out.TotalMessages += len(thread.Messages)
	}
	out.TotalThreads = len(out.Threads)

	body, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal ticket threads response: %w", err)
	}
	return body, nil
}

// extractThreadIDs pulls thread IDs out of the v4 associations response,
// preferring `toObjectId` and falling back to `id`. Numeric IDs are
// stringified; non-string/non-numeric raw values are skipped.
func extractThreadIDs(resp ticketAssociationsResponse) []string {
	ids := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		if id := decodeRawJSONID(r.ToObjectID); id != "" {
			ids = append(ids, id)
			continue
		}
		if id := decodeRawJSONID(r.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// decodeRawJSONID extracts an ID from a json.RawMessage that may be a JSON
// string ("12345") or a JSON number (12345). Returns "" for anything else.
func decodeRawJSONID(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(trimmed, &s); err == nil {
		return s
	}
	// Numeric: the raw JSON number text is already a valid string ID for
	// HubSpot. Don't bother with a strconv round-trip.
	first := trimmed[0]
	if first >= '0' && first <= '9' {
		return string(trimmed)
	}
	if first == '-' && len(trimmed) > 1 && trimmed[1] >= '0' && trimmed[1] <= '9' {
		return string(trimmed)
	}
	return ""
}
