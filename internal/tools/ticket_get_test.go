package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeTicketsGetter struct {
	body        []byte
	err         error
	gotCriteria string
	gotLimit    int
	calls       int
}

func (f *fakeTicketsGetter) GetTickets(_ context.Context, criteria string, limit int) ([]byte, error) {
	f.calls++
	f.gotCriteria = criteria
	f.gotLimit = limit
	return f.body, f.err
}

func TestNewGetTicketsTool_Schema(t *testing.T) {
	tool := NewGetTicketsTool()
	if tool.Name != ToolNameGetTickets {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	criteriaSchema, ok := tool.InputSchema.Properties["criteria"].(map[string]any)
	if !ok {
		t.Fatalf("expected criteria property, got %T", tool.InputSchema.Properties["criteria"])
	}
	if criteriaSchema["type"] != "string" {
		t.Fatalf("expected criteria.type=string, got %v", criteriaSchema["type"])
	}
	want := map[string]bool{"default": true, "Closed": true}
	switch enum := criteriaSchema["enum"].(type) {
	case []string:
		if len(enum) != 2 {
			t.Fatalf("expected enum of 2 values, got %v", enum)
		}
		for _, v := range enum {
			delete(want, v)
		}
	case []any:
		if len(enum) != 2 {
			t.Fatalf("expected enum of 2 values, got %v", enum)
		}
		for _, v := range enum {
			s, _ := v.(string)
			delete(want, s)
		}
	default:
		t.Fatalf("unexpected enum type %T (%v)", criteriaSchema["enum"], criteriaSchema["enum"])
	}
	if len(want) != 0 {
		t.Fatalf("missing enum values: %v", want)
	}

	limitSchema, ok := tool.InputSchema.Properties["limit"].(map[string]any)
	if !ok {
		t.Fatalf("expected limit property, got %T", tool.InputSchema.Properties["limit"])
	}
	if limitSchema["type"] != "number" {
		t.Fatalf("expected limit.type=number, got %v", limitSchema["type"])
	}
	for _, name := range tool.InputSchema.Required {
		if name == "criteria" || name == "limit" {
			t.Fatalf("%s should not be required", name)
		}
	}
}

func TestGetTicketsHandler_Success(t *testing.T) {
	payload := []byte(`{"total":1,"results":[{"id":"42"}]}`)
	fake := &fakeTicketsGetter{body: payload}
	h := GetTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"criteria": "Closed", "limit": float64(20)}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.gotCriteria != "Closed" {
		t.Fatalf("expected criteria=Closed, got %q", fake.gotCriteria)
	}
	if fake.gotLimit != 20 {
		t.Fatalf("expected limit=20, got %d", fake.gotLimit)
	}
	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("body not JSON: %v (%q)", err, body)
	}
}

func TestGetTicketsHandler_Defaults(t *testing.T) {
	fake := &fakeTicketsGetter{body: []byte(`{}`)}
	h := GetTicketsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	if _, err := h(context.Background(), req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if fake.gotCriteria != ticketsCriteriaDefaultName {
		t.Fatalf("expected default criteria=%q, got %q", ticketsCriteriaDefaultName, fake.gotCriteria)
	}
	if fake.gotLimit != defaultTicketsToolLimit {
		t.Fatalf("expected default limit=%d, got %d", defaultTicketsToolLimit, fake.gotLimit)
	}
}

func TestGetTicketsHandler_APIError(t *testing.T) {
	fake := &fakeTicketsGetter{err: errors.New("upstream 500")}
	h := GetTicketsHandler(fake)

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
