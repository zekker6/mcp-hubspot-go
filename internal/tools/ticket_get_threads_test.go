package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeTicketThreadsGetter struct {
	body        []byte
	err         error
	gotTicketID string
	calls       int
}

func (f *fakeTicketThreadsGetter) GetTicketConversationThreads(_ context.Context, ticketID string) ([]byte, error) {
	f.calls++
	f.gotTicketID = ticketID
	return f.body, f.err
}

func TestNewGetTicketConversationThreadsTool_Schema(t *testing.T) {
	tool := NewGetTicketConversationThreadsTool()
	if tool.Name != ToolNameGetTicketConversationThreads {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	idSchema, ok := tool.InputSchema.Properties["ticket_id"].(map[string]any)
	if !ok {
		t.Fatalf("expected ticket_id property, got %T", tool.InputSchema.Properties["ticket_id"])
	}
	if idSchema["type"] != "string" {
		t.Fatalf("expected ticket_id.type=string, got %v", idSchema["type"])
	}

	hasRequired := false
	for _, name := range tool.InputSchema.Required {
		if name == "ticket_id" {
			hasRequired = true
		}
	}
	if !hasRequired {
		t.Fatal("expected ticket_id to be marked required")
	}
}

func TestGetTicketConversationThreadsHandler_Success(t *testing.T) {
	payload := []byte(`{"ticket_id":"T1","threads":[{"id":"1001","messages":[{"id":"m1","type":"MESSAGE"}]}],"total_threads":1,"total_messages":1}`)
	fake := &fakeTicketThreadsGetter{body: payload}
	h := GetTicketConversationThreadsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"ticket_id": "T1"}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.gotTicketID != "T1" {
		t.Fatalf("expected ticket_id=T1, got %q", fake.gotTicketID)
	}
	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("body not JSON: %v (%q)", err, body)
	}
	if obj["ticket_id"] != "T1" {
		t.Fatalf("expected ticket_id=T1, got %v", obj["ticket_id"])
	}
}

func TestGetTicketConversationThreadsHandler_MissingID(t *testing.T) {
	fake := &fakeTicketThreadsGetter{}
	h := GetTicketConversationThreadsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true when ticket_id missing")
	}
	if fake.calls != 0 {
		t.Fatalf("expected wrapper not called when validation fails, got %d calls", fake.calls)
	}
}

func TestGetTicketConversationThreadsHandler_APIError(t *testing.T) {
	fake := &fakeTicketThreadsGetter{err: errors.New("upstream 500")}
	h := GetTicketConversationThreadsHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"ticket_id": "T1"}
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
