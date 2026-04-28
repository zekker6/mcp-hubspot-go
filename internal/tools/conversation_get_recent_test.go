package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeRecentConversationsGetter struct {
	body     []byte
	err      error
	gotLimit int
	gotAfter string
	calls    int
}

func (f *fakeRecentConversationsGetter) GetRecentConversations(_ context.Context, limit int, after string) ([]byte, error) {
	f.calls++
	f.gotLimit = limit
	f.gotAfter = after
	return f.body, f.err
}

func TestNewGetRecentConversationsTool_Schema(t *testing.T) {
	tool := NewGetRecentConversationsTool()
	if tool.Name != ToolNameGetRecentConversations {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	limitSchema, ok := tool.InputSchema.Properties["limit"].(map[string]any)
	if !ok {
		t.Fatalf("expected limit property, got %T", tool.InputSchema.Properties["limit"])
	}
	if limitSchema["type"] != "number" {
		t.Fatalf("expected limit.type=number, got %v", limitSchema["type"])
	}

	afterSchema, ok := tool.InputSchema.Properties["after"].(map[string]any)
	if !ok {
		t.Fatalf("expected after property, got %T", tool.InputSchema.Properties["after"])
	}
	if afterSchema["type"] != "string" {
		t.Fatalf("expected after.type=string, got %v", afterSchema["type"])
	}

	for _, name := range tool.InputSchema.Required {
		if name == "limit" || name == "after" {
			t.Fatalf("%s should not be required", name)
		}
	}
}

func TestGetRecentConversationsHandler_Success(t *testing.T) {
	payload := []byte(`{"threads":[{"id":"t1","messages":{"results":[]}}],"paging":{"next":{"after":"cursor-NEXT"}}}`)
	fake := &fakeRecentConversationsGetter{body: payload}
	h := GetRecentConversationsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"limit": float64(5), "after": "cursor-PREV"}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.gotLimit != 5 {
		t.Fatalf("expected limit=5, got %d", fake.gotLimit)
	}
	if fake.gotAfter != "cursor-PREV" {
		t.Fatalf("expected after=cursor-PREV, got %q", fake.gotAfter)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("response body was not JSON: %v (body=%q)", err, body)
	}
	threads, ok := obj["threads"].([]any)
	if !ok || len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %v", obj["threads"])
	}
	paging, ok := obj["paging"].(map[string]any)
	if !ok {
		t.Fatalf("expected paging passthrough, got %v", obj["paging"])
	}
	next, _ := paging["next"].(map[string]any)
	if next["after"] != "cursor-NEXT" {
		t.Fatalf("expected paging.next.after passthrough, got %v", next)
	}
}

func TestGetRecentConversationsHandler_Defaults(t *testing.T) {
	fake := &fakeRecentConversationsGetter{body: []byte(`{"threads":[]}`)}
	h := GetRecentConversationsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	if _, err := h(context.Background(), req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if fake.gotLimit != defaultRecentConversationsToolLimit {
		t.Fatalf("expected default limit=%d, got %d", defaultRecentConversationsToolLimit, fake.gotLimit)
	}
	if fake.gotAfter != "" {
		t.Fatalf("expected empty after, got %q", fake.gotAfter)
	}
}

func TestGetRecentConversationsHandler_APIError(t *testing.T) {
	fake := &fakeRecentConversationsGetter{err: errors.New("upstream 500")}
	h := GetRecentConversationsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on api failure")
	}
	body := textOf(t, res)
	if !strings.HasPrefix(body, "HubSpot API error: ") {
		t.Fatalf("expected HubSpot API error prefix, got %q", body)
	}
	if !strings.Contains(body, "upstream 500") {
		t.Fatalf("expected underlying error in message, got %q", body)
	}
}
