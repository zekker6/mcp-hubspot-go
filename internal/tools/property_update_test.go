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

type fakePropertyUpdater struct {
	body          []byte
	err           error
	gotObjectType string
	gotName       string
	gotFields     map[string]any
	calls         int
}

func (f *fakePropertyUpdater) UpdateProperty(_ context.Context, objectType, propertyName string, fields map[string]any) ([]byte, error) {
	f.calls++
	f.gotObjectType = objectType
	f.gotName = propertyName
	f.gotFields = fields
	return f.body, f.err
}

func TestNewUpdatePropertyTool_Schema(t *testing.T) {
	tool := NewUpdatePropertyTool()
	if tool.Name != ToolNameUpdateProperty {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	for _, key := range []string{"object_type", "property_name", "fields"} {
		if _, ok := tool.InputSchema.Properties[key]; !ok {
			t.Fatalf("expected %s in schema", key)
		}
		required := false
		for _, name := range tool.InputSchema.Required {
			if name == key {
				required = true
			}
		}
		if !required {
			t.Fatalf("expected %s to be required", key)
		}
	}

	fieldsSchema, ok := tool.InputSchema.Properties["fields"].(map[string]any)
	if !ok {
		t.Fatalf("expected fields schema map, got %T", tool.InputSchema.Properties["fields"])
	}
	if fieldsSchema["type"] != "object" {
		t.Fatalf("expected fields.type=object, got %v", fieldsSchema["type"])
	}

	objectTypeSchema, ok := tool.InputSchema.Properties["object_type"].(map[string]any)
	if !ok {
		t.Fatalf("expected object_type schema map")
	}
	want := map[string]bool{"companies": true, "contacts": true}
	switch enum := objectTypeSchema["enum"].(type) {
	case []string:
		for _, v := range enum {
			delete(want, v)
		}
	case []any:
		for _, v := range enum {
			s, _ := v.(string)
			delete(want, s)
		}
	default:
		t.Fatalf("unexpected enum type %T", objectTypeSchema["enum"])
	}
	if len(want) != 0 {
		t.Fatalf("missing enum values: %v", want)
	}
}

func TestUpdatePropertyHandler_Success(t *testing.T) {
	payload := []byte(`{"name":"industry","label":"Industry (renamed)"}`)
	fake := &fakePropertyUpdater{body: payload}
	h := UpdatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolNameUpdateProperty
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"property_name": "industry",
		"fields":        map[string]any{"label": "Industry (renamed)"},
	}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.calls != 1 {
		t.Fatalf("expected updater called once, got %d", fake.calls)
	}
	if fake.gotObjectType != "companies" {
		t.Fatalf("got object_type %q", fake.gotObjectType)
	}
	if fake.gotName != "industry" {
		t.Fatalf("got property_name %q", fake.gotName)
	}
	want := map[string]any{"label": "Industry (renamed)"}
	if !reflect.DeepEqual(fake.gotFields, want) {
		t.Fatalf("got fields %v, want %v", fake.gotFields, want)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("body not JSON: %v (%q)", err, body)
	}
	if obj["label"] != "Industry (renamed)" {
		t.Fatalf("expected label in response, got %v", obj["label"])
	}
}

func TestUpdatePropertyHandler_MissingObjectType(t *testing.T) {
	fake := &fakePropertyUpdater{}
	h := UpdatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"property_name": "industry",
		"fields":        map[string]any{"label": "X"},
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing object_type")
	}
	if fake.calls != 0 {
		t.Fatalf("expected updater not called, got %d", fake.calls)
	}
}

func TestUpdatePropertyHandler_MissingPropertyName(t *testing.T) {
	fake := &fakePropertyUpdater{}
	h := UpdatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type": "companies",
		"fields":      map[string]any{"label": "X"},
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing property_name")
	}
	if fake.calls != 0 {
		t.Fatalf("expected updater not called, got %d", fake.calls)
	}
}

func TestUpdatePropertyHandler_MissingFields(t *testing.T) {
	fake := &fakePropertyUpdater{}
	h := UpdatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"property_name": "industry",
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing fields")
	}
	if fake.calls != 0 {
		t.Fatalf("expected updater not called, got %d", fake.calls)
	}
}

func TestUpdatePropertyHandler_FieldsWrongType(t *testing.T) {
	fake := &fakePropertyUpdater{}
	h := UpdatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"property_name": "industry",
		"fields":        "scalar",
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on non-object fields")
	}
	if fake.calls != 0 {
		t.Fatalf("expected updater not called, got %d", fake.calls)
	}
}

func TestUpdatePropertyHandler_APIError(t *testing.T) {
	fake := &fakePropertyUpdater{err: errors.New("upstream 500")}
	h := UpdatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"property_name": "industry",
		"fields":        map[string]any{"label": "X"},
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
