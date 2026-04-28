package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeCompanyActivityGetter struct {
	body  []byte
	err   error
	gotID string
	calls int
}

func (f *fakeCompanyActivityGetter) GetCompanyActivity(_ context.Context, id string) ([]byte, error) {
	f.calls++
	f.gotID = id
	return f.body, f.err
}

func TestNewGetCompanyActivityTool_Schema(t *testing.T) {
	tool := NewGetCompanyActivityTool()
	if tool.Name != ToolNameGetCompanyActivity {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	if _, ok := tool.InputSchema.Properties["company_id"]; !ok {
		t.Fatal("expected company_id property")
	}
	required := false
	for _, name := range tool.InputSchema.Required {
		if name == "company_id" {
			required = true
		}
	}
	if !required {
		t.Fatal("expected company_id to be required")
	}
}

func TestGetCompanyActivityHandler_Success(t *testing.T) {
	payload := []byte(`{"results":[{"engagement":{"id":1,"type":"NOTE"}}],"hasMore":false}`)
	fake := &fakeCompanyActivityGetter{body: payload}
	h := GetCompanyActivityHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"company_id": "42"}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.gotID != "42" {
		t.Fatalf("expected id=42, got %q", fake.gotID)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("response body was not JSON: %v (body=%q)", err, body)
	}
	results, ok := obj["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", obj["results"])
	}
}

func TestGetCompanyActivityHandler_MissingCompanyID(t *testing.T) {
	fake := &fakeCompanyActivityGetter{}
	h := GetCompanyActivityHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing company_id")
	}
	if fake.calls != 0 {
		t.Fatalf("expected getter not to be called, got %d calls", fake.calls)
	}
}

func TestGetCompanyActivityHandler_APIError(t *testing.T) {
	fake := &fakeCompanyActivityGetter{err: errors.New("upstream 500")}
	h := GetCompanyActivityHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"company_id": "99"}
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
