package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClient_GetCompany_Success(t *testing.T) {
	var (
		gotPath  string
		gotQuery url.Values
		gotAuth  string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "123",
			"properties": {"name": "Acme", "domain": "acme.com", "custom_field": "x"},
			"createdAt": "2024-01-02T03:04:05.000Z",
			"updatedAt": "2024-02-03T04:05:06.000Z",
			"archived": false
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token-xyz", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetCompany(context.Background(), "123", []string{"custom_field"})
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}

	if gotPath != "/crm/v3/objects/companies/123" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") || !strings.HasSuffix(gotAuth, "token-xyz") {
		t.Fatalf("unexpected Authorization header: %q", gotAuth)
	}
	props := gotQuery.Get("properties")
	if !strings.Contains(props, "custom_field") {
		t.Fatalf("expected custom_field in properties query, got %q", props)
	}
	if !strings.Contains(props, "name") {
		t.Fatalf("expected default fields in properties query, got %q", props)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}
	if obj["id"] != "123" {
		t.Fatalf("expected id 123, got %v", obj["id"])
	}
	innerProps, ok := obj["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", obj["properties"])
	}
	if innerProps["name"] != "Acme" {
		t.Fatalf("expected name=Acme, got %v", innerProps["name"])
	}
	if innerProps["custom_field"] != "x" {
		t.Fatalf("expected custom_field passthrough, got %v", innerProps["custom_field"])
	}
}

func TestClient_GetRecentCompanies_Success(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotBody   []byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total": 2,
			"results": [
				{"id": "1", "properties": {"name": "Newest"}, "createdAt": "2024-01-01T00:00:00.000Z", "updatedAt": "2024-03-01T00:00:00.000Z", "archived": false},
				{"id": "2", "properties": {"name": "Older"},  "createdAt": "2024-01-01T00:00:00.000Z", "updatedAt": "2024-02-01T00:00:00.000Z", "archived": false}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetRecentCompanies(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetRecentCompanies: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/companies/search" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"limit":5`)) {
		t.Fatalf("expected limit=5 in body, got %s", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`"propertyName":"hs_lastmodifieddate"`)) {
		t.Fatalf("expected sort by hs_lastmodifieddate, got %s", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`"direction":"DESCENDING"`)) {
		t.Fatalf("expected DESCENDING direction, got %s", gotBody)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}
	results, ok := obj["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 results, got %v", obj["results"])
	}
}

func TestClient_GetRecentCompanies_DefaultLimit(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.GetRecentCompanies(context.Background(), 0); err != nil {
		t.Fatalf("GetRecentCompanies: %v", err)
	}
	if !bytes.Contains(gotBody, []byte(`"limit":10`)) {
		t.Fatalf("expected default limit=10 in body, got %s", gotBody)
	}
}

func TestClient_GetRecentCompanies_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","message":"boom"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetRecentCompanies(context.Background(), 5)
	if err == nil {
		t.Fatalf("expected error on 500, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_GetCompanyActivity_Success(t *testing.T) {
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
			"results": [
				{"engagement": {"id": 9001, "type": "NOTE"}},
				{"engagement": {"id": 9002, "type": "CALL"}}
			],
			"hasMore": false,
			"offset": 0
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetCompanyActivity(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetCompanyActivity: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %q", gotMethod)
	}
	if gotPath != "/engagements/v1/engagements/associated/COMPANY/42/paged" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	results, ok := obj["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 engagements, got %v", obj["results"])
	}
}

func TestClient_GetCompanyActivity_EmptyID(t *testing.T) {
	c, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetCompanyActivity(context.Background(), ""); err == nil {
		t.Fatal("expected error on empty company id")
	}
}

func TestClient_GetCompanyActivity_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","message":"company not found"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetCompanyActivity(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error on 404, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_CreateCompany_NoDuplicate(t *testing.T) {
	var (
		searchCalls int
		createCalls int
		createBody  []byte
		createPath  string
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/crm/v3/objects/companies/search", func(w http.ResponseWriter, _ *http.Request) {
		searchCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"results":[]}`))
	})
	mux.HandleFunc("/crm/v3/objects/companies", func(w http.ResponseWriter, r *http.Request) {
		createCalls++
		createPath = r.URL.Path
		createBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "999",
			"properties": {"name": "Acme", "website": "acme.com"},
			"createdAt": "2024-01-01T00:00:00.000Z",
			"updatedAt": "2024-01-01T00:00:00.000Z",
			"archived": false
		}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.CreateCompany(context.Background(), map[string]any{"name": "Acme", "website": "acme.com"})
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	if searchCalls != 1 {
		t.Fatalf("expected 1 search call, got %d", searchCalls)
	}
	if createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", createCalls)
	}
	if createPath != "/crm/v3/objects/companies" {
		t.Fatalf("unexpected create path: %q", createPath)
	}
	if !bytes.Contains(createBody, []byte(`"name":"Acme"`)) {
		t.Fatalf("expected create body to contain name=Acme, got %s", createBody)
	}
	if !bytes.Contains(createBody, []byte(`"properties":`)) {
		t.Fatalf("expected create body to wrap fields under properties, got %s", createBody)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["id"] != "999" {
		t.Fatalf("expected id=999, got %v", obj["id"])
	}
	if dup, ok := obj["duplicate"].(bool); ok && dup {
		t.Fatalf("expected non-duplicate response, got duplicate=true")
	}
}

func TestClient_CreateCompany_DuplicateExists(t *testing.T) {
	var (
		searchCalls int
		createCalls int
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/crm/v3/objects/companies/search", func(w http.ResponseWriter, _ *http.Request) {
		searchCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total": 1,
			"results": [
				{"id": "777", "properties": {"name": "Acme"}, "createdAt": "2024-01-01T00:00:00.000Z", "updatedAt": "2024-01-01T00:00:00.000Z", "archived": false}
			]
		}`))
	})
	mux.HandleFunc("/crm/v3/objects/companies", func(w http.ResponseWriter, _ *http.Request) {
		createCalls++
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.CreateCompany(context.Background(), map[string]any{"name": "Acme"})
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	if searchCalls != 1 {
		t.Fatalf("expected 1 search call, got %d", searchCalls)
	}
	if createCalls != 0 {
		t.Fatalf("expected 0 create calls when duplicate exists, got %d", createCalls)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if dup, _ := obj["duplicate"].(bool); !dup {
		t.Fatalf("expected duplicate=true, got %v", obj["duplicate"])
	}
	matches, ok := obj["matches"].([]any)
	if !ok || len(matches) != 1 {
		t.Fatalf("expected matches with 1 entry, got %v", obj["matches"])
	}
	first, _ := matches[0].(map[string]any)
	if first["id"] != "777" {
		t.Fatalf("expected match id=777, got %v", first["id"])
	}
}

func TestClient_CreateCompany_SearchFails(t *testing.T) {
	var (
		searchCalls int
		createCalls int
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/crm/v3/objects/companies/search", func(w http.ResponseWriter, _ *http.Request) {
		searchCalls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","message":"hubspot down"}`))
	})
	mux.HandleFunc("/crm/v3/objects/companies", func(w http.ResponseWriter, _ *http.Request) {
		createCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"never-created"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.CreateCompany(context.Background(), map[string]any{"name": "Acme"})
	if err == nil {
		t.Fatalf("expected error when search fails, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
	if searchCalls != 1 {
		t.Fatalf("expected 1 search call, got %d", searchCalls)
	}
	if createCalls != 0 {
		t.Fatalf("must NOT call create when search fails, got %d create calls", createCalls)
	}
}

func TestClient_CreateCompany_MissingName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made when properties.name is missing")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.CreateCompany(context.Background(), map[string]any{"website": "acme.com"}); err == nil {
		t.Fatal("expected error on missing properties.name")
	}
	if _, err := c.CreateCompany(context.Background(), nil); err == nil {
		t.Fatal("expected error on nil properties")
	}
}

func TestClient_UpdateCompany_Success(t *testing.T) {
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
			"id": "42",
			"properties": {"name": "Renamed"},
			"createdAt": "2024-01-01T00:00:00.000Z",
			"updatedAt": "2024-02-02T00:00:00.000Z",
			"archived": false
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.UpdateCompany(context.Background(), "42", map[string]any{"name": "Renamed"})
	if err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/companies/42" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"name":"Renamed"`)) {
		t.Fatalf("expected update body to contain name=Renamed, got %s", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`"properties":`)) {
		t.Fatalf("expected update body to wrap fields under properties, got %s", gotBody)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["id"] != "42" {
		t.Fatalf("expected id=42, got %v", obj["id"])
	}
}

func TestClient_UpdateCompany_GuardArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made when guard args are invalid")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.UpdateCompany(context.Background(), "", map[string]any{"name": "X"}); err == nil {
		t.Fatal("expected error on empty company id")
	}
	if _, err := c.UpdateCompany(context.Background(), "1", nil); err == nil {
		t.Fatal("expected error on nil properties")
	}
}

func TestClient_UpdateCompany_Error(t *testing.T) {
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

	body, err := c.UpdateCompany(context.Background(), "42", map[string]any{"name": "X"})
	if err == nil {
		t.Fatalf("expected error on 400, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_GetCompany_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","message":"company not found"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetCompany(context.Background(), "missing", nil)
	if err == nil {
		t.Fatalf("expected error on 404, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}
