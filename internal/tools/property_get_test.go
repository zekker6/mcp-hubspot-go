package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakePropertyGetter struct {
	body          []byte
	err           error
	gotObjectType string
	gotName       string
	calls         int
}

func (f *fakePropertyGetter) GetProperty(_ context.Context, objectType, propertyName string) ([]byte, error) {
	f.calls++
	f.gotObjectType = objectType
	f.gotName = propertyName
	return f.body, f.err
}

func TestNewGetPropertyTool_Schema(t *testing.T) {
	tool := NewGetPropertyTool()
	if tool.Name != ToolNameGetProperty {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	for _, key := range []string{"object_type", "property_name"} {
		if _, ok := tool.InputSchema.Properties[key]; !ok {
			t.Fatalf("expected %s property in schema", key)
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

	objectTypeSchema, ok := tool.InputSchema.Properties["object_type"].(map[string]any)
	if !ok {
		t.Fatalf("expected object_type schema map, got %T", tool.InputSchema.Properties["object_type"])
	}
	if objectTypeSchema["type"] != "string" {
		t.Fatalf("expected object_type.type=string, got %v", objectTypeSchema["type"])
	}

	want := map[string]bool{"companies": true, "contacts": true}
	switch enum := objectTypeSchema["enum"].(type) {
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
		t.Fatalf("unexpected enum type %T (%v)", objectTypeSchema["enum"], objectTypeSchema["enum"])
	}
	if len(want) != 0 {
		t.Fatalf("missing enum values: %v", want)
	}
}

func TestGetPropertyHandler_Success(t *testing.T) {
	payload := []byte(`{"name":"industry","label":"Industry"}`)
	fake := &fakePropertyGetter{body: payload}
	h := GetPropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"property_name": "industry",
	}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}
	if fake.gotObjectType != "companies" {
		t.Fatalf("expected object_type=companies, got %q", fake.gotObjectType)
	}
	if fake.gotName != "industry" {
		t.Fatalf("expected property_name=industry, got %q", fake.gotName)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("body not JSON: %v (%q)", err, body)
	}
	if obj["name"] != "industry" {
		t.Fatalf("expected name=industry, got %v", obj["name"])
	}
}

func TestGetPropertyHandler_MissingObjectType(t *testing.T) {
	fake := &fakePropertyGetter{}
	h := GetPropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"property_name": "industry"}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing object_type")
	}
	if fake.calls != 0 {
		t.Fatalf("expected getter not called, got %d calls", fake.calls)
	}
}

func TestGetPropertyHandler_MissingPropertyName(t *testing.T) {
	fake := &fakePropertyGetter{}
	h := GetPropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"object_type": "companies"}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on missing property_name")
	}
	if fake.calls != 0 {
		t.Fatalf("expected getter not called, got %d calls", fake.calls)
	}
}

func TestGetPropertyHandler_APIError(t *testing.T) {
	fake := &fakePropertyGetter{err: errors.New("upstream 500")}
	h := GetPropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "contacts",
		"property_name": "email",
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
