package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetProperty_Companies(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotAuth   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "industry",
			"label": "Industry",
			"type": "enumeration",
			"fieldType": "select",
			"groupName": "companyinformation",
			"options": [
				{"label": "Technology", "value": "TECH", "displayOrder": 0},
				{"label": "Finance",    "value": "FIN",  "displayOrder": 1}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetProperty(context.Background(), "companies", "industry")
	if err != nil {
		t.Fatalf("GetProperty: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/properties/companies/industry" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["name"] != "industry" {
		t.Fatalf("expected name=industry, got %v", obj["name"])
	}
	if obj["label"] != "Industry" {
		t.Fatalf("expected label=Industry, got %v", obj["label"])
	}
}

func TestClient_GetProperty_Contacts(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "email",
			"label": "Email",
			"type": "string",
			"fieldType": "text"
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetProperty(context.Background(), "contacts", "email")
	if err != nil {
		t.Fatalf("GetProperty: %v", err)
	}
	if gotPath != "/crm/v3/properties/contacts/email" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["name"] != "email" {
		t.Fatalf("expected name=email, got %v", obj["name"])
	}
}

func TestClient_GetProperty_UnsupportedObjectType(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for _, objectType := range []string{"deals", "tickets", "products", ""} {
		body, err := c.GetProperty(context.Background(), objectType, "anything")
		if err == nil {
			t.Fatalf("expected error for unsupported object type %q, got body=%s", objectType, body)
		}
		if body != nil {
			t.Fatalf("expected nil body on error, got %s", body)
		}
		if !strings.Contains(err.Error(), "unsupported object type") {
			t.Fatalf("expected error to mention unsupported object type, got %v", err)
		}
	}
	if called {
		t.Fatal("expected upstream to not be called for unsupported object types")
	}
}

func TestClient_GetProperty_EmptyName(t *testing.T) {
	c, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetProperty(context.Background(), "companies", ""); err == nil {
		t.Fatal("expected error on empty property name")
	}
}

func TestClient_GetProperty_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","message":"property not found"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetProperty(context.Background(), "companies", "nope")
	if err == nil {
		t.Fatalf("expected error on 404, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_CreateProperty_Success(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotBody   []byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "industry_v2",
			"label": "Industry v2",
			"type": "enumeration",
			"fieldType": "select",
			"groupName": "companyinformation",
			"options": [
				{"label": "Technology", "value": "TECH", "displayOrder": 0}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	options := []any{
		map[string]any{"label": "Technology", "value": "TECH", "displayOrder": float64(0)},
	}
	body, err := c.CreateProperty(
		context.Background(),
		"companies",
		"industry_v2",
		"Industry v2",
		"enumeration",
		"select",
		"companyinformation",
		options,
	)
	if err != nil {
		t.Fatalf("CreateProperty: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/properties/companies" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}

	for _, want := range [][]byte{
		[]byte(`"name":"industry_v2"`),
		[]byte(`"label":"Industry v2"`),
		[]byte(`"type":"enumeration"`),
		[]byte(`"fieldType":"select"`),
		[]byte(`"groupName":"companyinformation"`),
		[]byte(`"options":`),
		[]byte(`"value":"TECH"`),
	} {
		if !bytes.Contains(gotBody, want) {
			t.Fatalf("create body missing %s: %s", want, gotBody)
		}
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["name"] != "industry_v2" {
		t.Fatalf("expected name=industry_v2, got %v", obj["name"])
	}
}

func TestClient_CreateProperty_NoOptions(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"notes","label":"Notes","type":"string","fieldType":"text"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.CreateProperty(
		context.Background(),
		"contacts",
		"notes",
		"Notes",
		"string",
		"text",
		"contactinformation",
		nil,
	); err != nil {
		t.Fatalf("CreateProperty: %v", err)
	}

	if bytes.Contains(gotBody, []byte(`"options"`)) {
		t.Fatalf("did not expect options key in body when no options passed: %s", gotBody)
	}
}

func TestClient_CreateProperty_GuardArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made on guard arg failure")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	cases := []struct {
		name                                                     string
		objectType, pname, label, propType, fieldType, groupName string
	}{
		{"unsupported_object_type", "deals", "n", "L", "string", "text", "g"},
		{"empty_name", "companies", "", "L", "string", "text", "g"},
		{"empty_label", "companies", "n", "", "string", "text", "g"},
		{"empty_property_type", "companies", "n", "L", "", "text", "g"},
		{"empty_field_type", "companies", "n", "L", "string", "", "g"},
		{"empty_group_name", "companies", "n", "L", "string", "text", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := c.CreateProperty(context.Background(), tc.objectType, tc.pname, tc.label, tc.propType, tc.fieldType, tc.groupName, nil)
			if err == nil {
				t.Fatalf("expected guard error, got body=%s", body)
			}
			if body != nil {
				t.Fatalf("expected nil body on guard error, got %s", body)
			}
		})
	}
}

func TestClient_CreateProperty_BadFieldTypeOptionCombo(t *testing.T) {
	// HubSpot rejects field_type=select without options with a 400 echo. Ensure
	// the wrapper surfaces the error rather than swallowing it. The wrapper
	// performs no client-side validation of this combo (per plan).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"error","message":"options is required for field type select"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.CreateProperty(
		context.Background(),
		"companies",
		"industry",
		"Industry",
		"enumeration",
		"select",
		"companyinformation",
		nil,
	)
	if err == nil {
		t.Fatalf("expected error on 400, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_UpdateProperty_Success(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotBody   []byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "industry",
			"label": "Industry (renamed)",
			"type": "enumeration",
			"fieldType": "select",
			"groupName": "companyinformation"
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.UpdateProperty(context.Background(), "companies", "industry", map[string]any{
		"label": "Industry (renamed)",
	})
	if err != nil {
		t.Fatalf("UpdateProperty: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/properties/companies/industry" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"label":"Industry (renamed)"`)) {
		t.Fatalf("expected update body to contain label, got %s", gotBody)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["label"] != "Industry (renamed)" {
		t.Fatalf("expected label=Industry (renamed), got %v", obj["label"])
	}
}

func TestClient_UpdateProperty_GuardArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made on guard arg failure")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.UpdateProperty(context.Background(), "deals", "industry", map[string]any{"label": "x"}); err == nil {
		t.Fatal("expected error on unsupported object type")
	}
	if _, err := c.UpdateProperty(context.Background(), "companies", "", map[string]any{"label": "x"}); err == nil {
		t.Fatal("expected error on empty property name")
	}
	if _, err := c.UpdateProperty(context.Background(), "companies", "industry", nil); err == nil {
		t.Fatal("expected error on nil fields")
	}
	if _, err := c.UpdateProperty(context.Background(), "companies", "industry", map[string]any{}); err == nil {
		t.Fatal("expected error on empty fields")
	}
}

func TestClient_UpdateProperty_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"error","message":"validation"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.UpdateProperty(context.Background(), "companies", "industry", map[string]any{"label": "x"})
	if err == nil {
		t.Fatalf("expected error on 400, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
	if !strings.Contains(err.Error(), "industry") {
		t.Fatalf("expected error to reference property name, got %v", err)
	}
}
