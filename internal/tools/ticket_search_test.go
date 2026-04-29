package tools

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeSearchTicketsGetter struct {
	body     []byte
	err      error
	gotQuery string
	gotLimit int
	gotProps []string
	gotAfter string
	calls    int
}

func (f *fakeSearchTicketsGetter) SearchTickets(_ context.Context, query string, limit int, properties []string, after string) ([]byte, error) {
	f.calls++
	f.gotQuery = query
	f.gotLimit = limit
	f.gotProps = properties
	f.gotAfter = after
	return f.body, f.err
}

func TestNewSearchTicketsTool_Schema(t *testing.T) {
	tool := NewSearchTicketsTool()
	if tool.Name != ToolNameSearchTickets {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}
	if !strings.Contains(tool.Description, "hubspot_get_tickets") {
		t.Fatalf("expected description to cross-reference hubspot_get_tickets, got %q", tool.Description)
	}

	if _, ok := tool.InputSchema.Properties["query"]; !ok {
		t.Fatal("expected query property")
	}
	required := false
	for _, name := range tool.InputSchema.Required {
		if name == "query" {
			required = true
		}
	}
	if !required {
		t.Fatal("expected query to be required")
	}

	limitSchema, ok := tool.InputSchema.Properties["limit"].(map[string]any)
	if !ok {
		t.Fatalf("expected limit property, got %T", tool.InputSchema.Properties["limit"])
	}
	if limitSchema["type"] != "number" {
		t.Fatalf("expected limit.type=number, got %v", limitSchema["type"])
	}

	propsSchema, ok := tool.InputSchema.Properties["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties schema map, got %T", tool.InputSchema.Properties["properties"])
	}
	if propsSchema["type"] != "array" {
		t.Fatalf("expected properties.type=array, got %v", propsSchema["type"])
	}
	items, ok := propsSchema["items"].(map[string]any)
	if !ok {
		t.Fatalf("expected items schema, got %T", propsSchema["items"])
	}
	if items["type"] != "string" {
		t.Fatalf("expected items.type=string, got %v", items["type"])
	}

	afterSchema, ok := tool.InputSchema.Properties["after"].(map[string]any)
	if !ok {
		t.Fatalf("expected after property, got %T", tool.InputSchema.Properties["after"])
	}
	if afterSchema["type"] != "string" {
		t.Fatalf("expected after.type=string, got %v", afterSchema["type"])
	}

	for _, name := range tool.InputSchema.Required {
		if name == "limit" || name == "properties" || name == "after" {
			t.Fatalf("%s should not be required", name)
		}
	}
}

func TestSearchTicketsHandler_Success(t *testing.T) {
	payload := []byte(`{"total":1,"results":[{"id":"7","properties":{"subject":"Login issue"}}],"paging":{"next":{"after":"10"}}}`)
	fake := &fakeSearchTicketsGetter{body: payload}
	h := SearchTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolNameSearchTickets
	req.Params.Arguments = map[string]any{
		"query":      "login",
		"limit":      float64(25),
		"properties": []any{"subject", "content"},
		"after":      "5",
	}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.calls != 1 {
		t.Fatalf("expected getter to be called once, got %d", fake.calls)
	}
	if fake.gotQuery != "login" {
		t.Fatalf("expected query=login, got %q", fake.gotQuery)
	}
	if fake.gotLimit != 25 {
		t.Fatalf("expected limit=25, got %d", fake.gotLimit)
	}
	if !reflect.DeepEqual(fake.gotProps, []string{"subject", "content"}) {
		t.Fatalf("expected props [subject content], got %v", fake.gotProps)
	}
	if fake.gotAfter != "5" {
		t.Fatalf("expected after=5, got %q", fake.gotAfter)
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

func TestSearchTicketsHandler_MissingQuery(t *testing.T) {
	fake := &fakeSearchTicketsGetter{}
	h := SearchTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing query")
	}
	if fake.calls != 0 {
		t.Fatalf("expected getter not to be called, got %d calls", fake.calls)
	}
}

func TestSearchTicketsHandler_APIError(t *testing.T) {
	fake := &fakeSearchTicketsGetter{err: errors.New("upstream 500")}
	h := SearchTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "login"}
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

func TestSearchTicketsHandler_OptionalArgsAbsent(t *testing.T) {
	fake := &fakeSearchTicketsGetter{body: []byte(`{}`)}
	h := SearchTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "login"}
	if _, err := h(context.Background(), req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if fake.gotLimit != defaultSearchTicketsToolLimit {
		t.Fatalf("expected default limit=%d, got %d", defaultSearchTicketsToolLimit, fake.gotLimit)
	}
	if fake.gotProps != nil {
		t.Fatalf("expected nil properties when absent, got %v", fake.gotProps)
	}
	if fake.gotAfter != "" {
		t.Fatalf("expected empty after when absent, got %q", fake.gotAfter)
	}
}

func TestSearchTicketsHandler_OptionalArgsPresent(t *testing.T) {
	fake := &fakeSearchTicketsGetter{body: []byte(`{}`)}
	h := SearchTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":      "login",
		"limit":      float64(50),
		"properties": []any{"subject"},
		"after":      "abc",
	}
	if _, err := h(context.Background(), req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if fake.gotLimit != 50 {
		t.Fatalf("expected limit=50, got %d", fake.gotLimit)
	}
	if !reflect.DeepEqual(fake.gotProps, []string{"subject"}) {
		t.Fatalf("expected props [subject], got %v", fake.gotProps)
	}
	if fake.gotAfter != "abc" {
		t.Fatalf("expected after=abc, got %q", fake.gotAfter)
	}
}
