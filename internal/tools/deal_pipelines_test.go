package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeDealPipelinesGetter struct {
	body  []byte
	err   error
	calls int
}

func (f *fakeDealPipelinesGetter) GetDealPipelines(_ context.Context) ([]byte, error) {
	f.calls++
	return f.body, f.err
}

func TestNewGetDealPipelinesTool_Schema(t *testing.T) {
	tool := NewGetDealPipelinesTool()
	if tool.Name != ToolNameGetDealPipelines {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}
	if len(tool.InputSchema.Required) != 0 {
		t.Fatalf("expected no required arguments, got %v", tool.InputSchema.Required)
	}
}

func TestGetDealPipelinesHandler_Success(t *testing.T) {
	payload := []byte(`{"results":[{"id":"default","label":"Sales Pipeline","stages":[{"id":"closedwon","label":"Closed won"}]}]}`)
	fake := &fakeDealPipelinesGetter{body: payload}
	h := GetDealPipelinesHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fake.calls)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("response body was not JSON: %v (body=%q)", err, body)
	}
	results, ok := obj["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 pipeline, got %v", obj["results"])
	}
}

func TestGetDealPipelinesHandler_APIError(t *testing.T) {
	fake := &fakeDealPipelinesGetter{err: errors.New("upstream 500")}
	h := GetDealPipelinesHandler(fake)

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
