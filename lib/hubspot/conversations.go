package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const defaultRecentConversationsLimit = 10

// conversationsThreadsPagingNext mirrors the `paging.next` shape returned by
// the HubSpot conversations API.
type conversationsThreadsPagingNext struct {
	After string `json:"after"`
	Link  string `json:"link,omitempty"`
}

// conversationsThreadsPaging mirrors the `paging` envelope returned by the
// HubSpot conversations API.
type conversationsThreadsPaging struct {
	Next *conversationsThreadsPagingNext `json:"next,omitempty"`
}

// conversationsThreadsListResponse is the minimal shape we need from the
// threads list endpoint. Each thread is kept as a free-form map so we can
// attach the messages payload and pass everything else through unchanged.
type conversationsThreadsListResponse struct {
	Results []map[string]any            `json:"results"`
	Paging  *conversationsThreadsPaging `json:"paging,omitempty"`
}

// GetRecentConversations lists conversation threads (most recent first) and
// embeds each thread's messages in-place under a `messages` key. The response
// is `{threads, paging}` where `paging` is omitted entirely when HubSpot did
// not return a `paging.next.after` cursor. `after` may be empty for the first
// page.
//
// The HubSpot SDK has no typed wrapper for the conversations API beyond
// VisitorIdentification, so this hits the v3 endpoint via the SDK's generic
// Get and returns aggregated JSON.
func (c *Client) GetRecentConversations(_ context.Context, limit int, after string) ([]byte, error) {
	if limit <= 0 {
		limit = defaultRecentConversationsLimit
	}

	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if after != "" {
		q.Set("after", after)
	}
	threadsPath := "/conversations/v3/conversations/threads?" + q.Encode()

	var listResp conversationsThreadsListResponse
	if err := c.sdk.Get(threadsPath, &listResp, nil); err != nil {
		return nil, fmt.Errorf("list conversation threads: %w", err)
	}

	for _, thread := range listResp.Results {
		id, _ := thread["id"].(string)
		if id == "" {
			continue
		}
		var messages json.RawMessage
		msgPath := "/conversations/v3/conversations/threads/" + url.PathEscape(id) + "/messages"
		if err := c.sdk.Get(msgPath, &messages, nil); err != nil {
			return nil, fmt.Errorf("get messages for thread %s: %w", id, err)
		}
		thread["messages"] = messages
	}

	out := map[string]any{"threads": listResp.Results}
	if listResp.Paging != nil && listResp.Paging.Next != nil && listResp.Paging.Next.After != "" {
		out["paging"] = listResp.Paging
	}

	body, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal conversations response: %w", err)
	}
	return body, nil
}
