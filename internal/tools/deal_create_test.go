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

type fakeDealCreator struct {
	body     []byte
	err      error
	gotProps map[string]any
	calls    int
}

func (f *fakeDealCreator) CreateDeal(_ context.Context, properties map[string]any) ([]byte, error) {
	f.calls++
	f.gotProps = properties
	return f.body, f.err
}

func TestNewCreateDealTool_Schema(t *testing.T) {
	tool := NewCreateDealTool()
	if tool.Name != ToolNameCreateDeal {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	propsSchema, ok := tool.InputSchema.Properties["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties schema map, got %T", tool.InputSchema.Properties["properties"])
	}
	if propsSchema["type"] != "object" {
		t.Fatalf("expected properties.type=object, got %v", propsSchema["type"])
	}

	required := false
	for _, name := range tool.InputSchema.Required {
		if name == "properties" {
			required = true
		}
	}
	if !required {
		t.Fatal("expected properties to be required")
	}
}

func TestCreateDealHandler_Success(t *testing.T) {
	payload := []byte(`{"id":"42","properties":{"dealname":"Big"}}`)
	fake := &fakeDealCreator{body: payload}
	h := CreateDealHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolNameCreateDeal
	req.Params.Arguments = map[string]any{
		"properties": map[string]any{"dealname": "Big", "amount": "1000"},
	}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.calls != 1 {
		t.Fatalf("expected creator to be called once, got %d", fake.calls)
	}
	want := map[string]any{"dealname": "Big", "amount": "1000"}
	if !reflect.DeepEqual(fake.gotProps, want) {
		t.Fatalf("got props %v, want %v", fake.gotProps, want)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("response body was not JSON: %v (body=%q)", err, body)
	}
	if obj["id"] != "42" {
		t.Fatalf("expected id=42 in body, got %v", obj["id"])
	}
}

func TestCreateDealHandler_MissingPropertiesArg(t *testing.T) {
	fake := &fakeDealCreator{}
	h := CreateDealHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing properties")
	}
	if fake.calls != 0 {
		t.Fatalf("expected creator not to be called, got %d calls", fake.calls)
	}
}

func TestCreateDealHandler_PropertiesWrongType(t *testing.T) {
	fake := &fakeDealCreator{}
	h := CreateDealHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"properties": "scalar",
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on non-object properties")
	}
	if fake.calls != 0 {
		t.Fatalf("expected creator not to be called, got %d calls", fake.calls)
	}
}

func TestCreateDealHandler_APIError(t *testing.T) {
	fake := &fakeDealCreator{err: errors.New("upstream 500")}
	h := CreateDealHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"properties": map[string]any{"dealname": "X"},
	}
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
