package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeRecentCompaniesGetter struct {
	body     []byte
	err      error
	gotLimit int
	calls    int
}

func (f *fakeRecentCompaniesGetter) GetRecentCompanies(_ context.Context, limit int) ([]byte, error) {
	f.calls++
	f.gotLimit = limit
	return f.body, f.err
}

func TestNewGetRecentCompaniesTool_Schema(t *testing.T) {
	tool := NewGetRecentCompaniesTool()
	if tool.Name != ToolNameGetRecentCompanies {
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
	for _, name := range tool.InputSchema.Required {
		if name == "limit" {
			t.Fatal("limit should not be required")
		}
	}
}

func TestGetRecentCompaniesHandler_Success(t *testing.T) {
	payload := []byte(`{"total":1,"results":[{"id":"1","properties":{"name":"Acme"}}]}`)
	fake := &fakeRecentCompaniesGetter{body: payload}
	h := GetRecentCompaniesHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"limit": float64(25)}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.gotLimit != 25 {
		t.Fatalf("expected limit=25, got %d", fake.gotLimit)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("response body was not JSON: %v (body=%q)", err, body)
	}
	if obj["total"].(float64) != 1 {
		t.Fatalf("expected total=1, got %v", obj["total"])
	}
}

func TestGetRecentCompaniesHandler_DefaultLimit(t *testing.T) {
	fake := &fakeRecentCompaniesGetter{body: []byte(`{}`)}
	h := GetRecentCompaniesHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	if _, err := h(context.Background(), req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if fake.gotLimit != defaultRecentCompaniesToolLimit {
		t.Fatalf("expected default limit=%d, got %d", defaultRecentCompaniesToolLimit, fake.gotLimit)
	}
}

func TestGetRecentCompaniesHandler_APIError(t *testing.T) {
	fake := &fakeRecentCompaniesGetter{err: errors.New("upstream 500")}
	h := GetRecentCompaniesHandler(fake)

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
