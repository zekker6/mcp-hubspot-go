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

type fakeCompanyGetter struct {
	body     []byte
	err      error
	gotID    string
	gotProps []string
	calls    int
}

func (f *fakeCompanyGetter) GetCompany(_ context.Context, id string, properties []string) ([]byte, error) {
	f.calls++
	f.gotID = id
	f.gotProps = properties
	return f.body, f.err
}

func TestNewGetCompanyTool_Schema(t *testing.T) {
	tool := NewGetCompanyTool()
	if tool.Name != ToolNameGetCompany {
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
}

func TestGetCompanyHandler_Success(t *testing.T) {
	payload := []byte(`{"id":"42","properties":{"name":"Acme"}}`)
	fake := &fakeCompanyGetter{body: payload}
	h := GetCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolNameGetCompany
	req.Params.Arguments = map[string]any{
		"company_id": "42",
		"properties": []any{"website", "phone"},
	}

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
	if !reflect.DeepEqual(fake.gotProps, []string{"website", "phone"}) {
		t.Fatalf("expected props [website phone], got %v", fake.gotProps)
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

func TestGetCompanyHandler_MissingCompanyID(t *testing.T) {
	fake := &fakeCompanyGetter{}
	h := GetCompanyHandler(fake)

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

func TestGetCompanyHandler_APIError(t *testing.T) {
	fake := &fakeCompanyGetter{err: errors.New("upstream 500")}
	h := GetCompanyHandler(fake)

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

func TestGetCompanyHandler_PropertiesOmittedWhenAbsent(t *testing.T) {
	fake := &fakeCompanyGetter{body: []byte(`{}`)}
	h := GetCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"company_id": "1"}
	if _, err := h(context.Background(), req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if fake.gotProps != nil {
		t.Fatalf("expected nil properties when absent, got %v", fake.gotProps)
	}
}
