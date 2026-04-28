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

type fakePropertyCreator struct {
	body            []byte
	err             error
	gotObjectType   string
	gotName         string
	gotLabel        string
	gotPropertyType string
	gotFieldType    string
	gotGroupName    string
	gotOptions      []any
	calls           int
}

func (f *fakePropertyCreator) CreateProperty(_ context.Context, objectType, name, label, propertyType, fieldType, groupName string, options []any) ([]byte, error) {
	f.calls++
	f.gotObjectType = objectType
	f.gotName = name
	f.gotLabel = label
	f.gotPropertyType = propertyType
	f.gotFieldType = fieldType
	f.gotGroupName = groupName
	f.gotOptions = options
	return f.body, f.err
}

func TestNewCreatePropertyTool_Schema(t *testing.T) {
	tool := NewCreatePropertyTool()
	if tool.Name != ToolNameCreateProperty {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}

	for _, key := range []string{"object_type", "name", "label", "property_type", "field_type", "group_name"} {
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

	// options is optional
	if _, ok := tool.InputSchema.Properties["options"]; !ok {
		t.Fatal("expected options in schema")
	}
	for _, name := range tool.InputSchema.Required {
		if name == "options" {
			t.Fatal("options must not be required")
		}
	}
	optSchema, ok := tool.InputSchema.Properties["options"].(map[string]any)
	if !ok {
		t.Fatalf("expected options schema map, got %T", tool.InputSchema.Properties["options"])
	}
	if optSchema["type"] != "array" {
		t.Fatalf("expected options.type=array, got %v", optSchema["type"])
	}

	// object_type enum is companies + contacts
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

func TestCreatePropertyHandler_SuccessWithOptions(t *testing.T) {
	payload := []byte(`{"name":"industry","label":"Industry","type":"enumeration"}`)
	fake := &fakePropertyCreator{body: payload}
	h := CreatePropertyHandler(fake)

	options := []any{
		map[string]any{"label": "Technology", "value": "TECH"},
		map[string]any{"label": "Finance", "value": "FIN"},
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolNameCreateProperty
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"name":          "industry",
		"label":         "Industry",
		"property_type": "enumeration",
		"field_type":    "select",
		"group_name":    "companyinformation",
		"options":       options,
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
	if fake.gotObjectType != "companies" {
		t.Fatalf("got object_type %q", fake.gotObjectType)
	}
	if fake.gotName != "industry" {
		t.Fatalf("got name %q", fake.gotName)
	}
	if fake.gotLabel != "Industry" {
		t.Fatalf("got label %q", fake.gotLabel)
	}
	if fake.gotPropertyType != "enumeration" {
		t.Fatalf("got property_type %q", fake.gotPropertyType)
	}
	if fake.gotFieldType != "select" {
		t.Fatalf("got field_type %q", fake.gotFieldType)
	}
	if fake.gotGroupName != "companyinformation" {
		t.Fatalf("got group_name %q", fake.gotGroupName)
	}
	if !reflect.DeepEqual(fake.gotOptions, options) {
		t.Fatalf("got options %v, want %v", fake.gotOptions, options)
	}

	body := textOf(t, res)
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("body not JSON: %v (%q)", err, body)
	}
}

func TestCreatePropertyHandler_SuccessNoOptions(t *testing.T) {
	payload := []byte(`{"name":"notes","label":"Notes","type":"string"}`)
	fake := &fakePropertyCreator{body: payload}
	h := CreatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "contacts",
		"name":          "notes",
		"label":         "Notes",
		"property_type": "string",
		"field_type":    "text",
		"group_name":    "contactinformation",
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
	if fake.gotOptions != nil {
		t.Fatalf("expected nil options when absent, got %v", fake.gotOptions)
	}
}

func TestCreatePropertyHandler_MissingRequired(t *testing.T) {
	cases := []struct {
		omit string
		args map[string]any
	}{
		{"object_type", map[string]any{
			"name": "n", "label": "L", "property_type": "string", "field_type": "text", "group_name": "g",
		}},
		{"name", map[string]any{
			"object_type": "companies", "label": "L", "property_type": "string", "field_type": "text", "group_name": "g",
		}},
		{"label", map[string]any{
			"object_type": "companies", "name": "n", "property_type": "string", "field_type": "text", "group_name": "g",
		}},
		{"property_type", map[string]any{
			"object_type": "companies", "name": "n", "label": "L", "field_type": "text", "group_name": "g",
		}},
		{"field_type", map[string]any{
			"object_type": "companies", "name": "n", "label": "L", "property_type": "string", "group_name": "g",
		}},
		{"group_name", map[string]any{
			"object_type": "companies", "name": "n", "label": "L", "property_type": "string", "field_type": "text",
		}},
	}
	for _, tc := range cases {
		t.Run("missing_"+tc.omit, func(t *testing.T) {
			fake := &fakePropertyCreator{}
			h := CreatePropertyHandler(fake)
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tc.args
			res, err := h(context.Background(), req)
			if err != nil {
				t.Fatalf("handler should not error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError=true on missing %s", tc.omit)
			}
			if fake.calls != 0 {
				t.Fatalf("expected creator not called, got %d", fake.calls)
			}
		})
	}
}

func TestCreatePropertyHandler_OptionsWrongType(t *testing.T) {
	fake := &fakePropertyCreator{}
	h := CreatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"name":          "n",
		"label":         "L",
		"property_type": "string",
		"field_type":    "text",
		"group_name":    "g",
		"options":       "not-an-array",
	}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on non-array options")
	}
	if fake.calls != 0 {
		t.Fatalf("expected creator not called, got %d calls", fake.calls)
	}
}

func TestCreatePropertyHandler_APIError(t *testing.T) {
	fake := &fakePropertyCreator{err: errors.New("upstream 400")}
	h := CreatePropertyHandler(fake)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"object_type":   "companies",
		"name":          "n",
		"label":         "L",
		"property_type": "string",
		"field_type":    "text",
		"group_name":    "g",
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
	if !strings.Contains(body, "upstream 400") {
		t.Fatalf("expected underlying error in message, got %q", body)
	}
}
