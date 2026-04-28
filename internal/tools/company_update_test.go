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

type fakeCompanyUpdater struct {
	body     []byte
	err      error
	gotID    string
	gotProps map[string]any
	calls    int
}

func (f *fakeCompanyUpdater) UpdateCompany(_ context.Context, id string, properties map[string]any) ([]byte, error) {
	f.calls++
	f.gotID = id
	f.gotProps = properties
	return f.body, f.err
}

func TestNewUpdateCompanyTool_Schema(t *testing.T) {
	tool := NewUpdateCompanyTool()
	if tool.Name != ToolNameUpdateCompany {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	if _, ok := tool.InputSchema.Properties["company_id"]; !ok {
		t.Fatal("expected company_id property")
	}
	propsSchema, ok := tool.InputSchema.Properties["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties schema map, got %T", tool.InputSchema.Properties["properties"])
	}
	if propsSchema["type"] != "object" {
		t.Fatalf("expected properties.type=object, got %v", propsSchema["type"])
	}

	required := map[string]bool{}
	for _, name := range tool.InputSchema.Required {
		required[name] = true
	}
	if !required["company_id"] {
		t.Fatal("expected company_id to be required")
	}
	if !required["properties"] {
		t.Fatal("expected properties to be required")
	}
}

func TestUpdateCompanyHandler_Success(t *testing.T) {
	payload := []byte(`{"id":"42","properties":{"name":"Renamed"}}`)
	fake := &fakeCompanyUpdater{body: payload}
	h := UpdateCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolNameUpdateCompany
	req.Params.Arguments = map[string]any{
		"company_id": "42",
		"properties": map[string]any{"name": "Renamed"},
	}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.calls != 1 {
		t.Fatalf("expected updater to be called once, got %d", fake.calls)
	}
	if fake.gotID != "42" {
		t.Fatalf("expected id=42, got %q", fake.gotID)
	}
	want := map[string]any{"name": "Renamed"}
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

func TestUpdateCompanyHandler_MissingCompanyID(t *testing.T) {
	fake := &fakeCompanyUpdater{}
	h := UpdateCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"properties": map[string]any{"name": "X"},
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing company_id")
	}
	if fake.calls != 0 {
		t.Fatalf("expected updater not to be called, got %d calls", fake.calls)
	}
}

func TestUpdateCompanyHandler_MissingProperties(t *testing.T) {
	fake := &fakeCompanyUpdater{}
	h := UpdateCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"company_id": "42",
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing properties")
	}
	if fake.calls != 0 {
		t.Fatalf("expected updater not to be called, got %d calls", fake.calls)
	}
}

func TestUpdateCompanyHandler_PropertiesWrongType(t *testing.T) {
	fake := &fakeCompanyUpdater{}
	h := UpdateCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"company_id": "42",
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
		t.Fatalf("expected updater not to be called, got %d calls", fake.calls)
	}
}

func TestUpdateCompanyHandler_APIError(t *testing.T) {
	fake := &fakeCompanyUpdater{err: errors.New("upstream 500")}
	h := UpdateCompanyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"company_id": "42",
		"properties": map[string]any{"name": "X"},
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
