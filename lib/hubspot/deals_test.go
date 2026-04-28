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

func TestClient_GetDeal_Success(t *testing.T) {
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
			"id": "111",
			"properties": {"dealname": "Big Deal", "amount": "5000", "custom_field": "y"},
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

	body, err := c.GetDeal(context.Background(), "111", []string{"custom_field"})
	if err != nil {
		t.Fatalf("GetDeal: %v", err)
	}

	if gotPath != "/crm/v3/objects/deals/111" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") || !strings.HasSuffix(gotAuth, "token-xyz") {
		t.Fatalf("unexpected Authorization header: %q", gotAuth)
	}
	props := gotQuery.Get("properties")
	if !strings.Contains(props, "custom_field") {
		t.Fatalf("expected custom_field in properties query, got %q", props)
	}
	if !strings.Contains(props, "dealname") {
		t.Fatalf("expected default fields in properties query, got %q", props)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}
	if obj["id"] != "111" {
		t.Fatalf("expected id=111, got %v", obj["id"])
	}
	innerProps, ok := obj["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", obj["properties"])
	}
	if innerProps["dealname"] != "Big Deal" {
		t.Fatalf("expected dealname=Big Deal, got %v", innerProps["dealname"])
	}
	if innerProps["custom_field"] != "y" {
		t.Fatalf("expected custom_field passthrough, got %v", innerProps["custom_field"])
	}
}

func TestClient_GetDeal_EmptyID(t *testing.T) {
	c, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetDeal(context.Background(), "", nil); err == nil {
		t.Fatal("expected error on empty deal id")
	}
}

func TestClient_GetDeal_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","message":"deal not found"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetDeal(context.Background(), "missing", nil)
	if err == nil {
		t.Fatalf("expected error on 404, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_GetRecentDeals_Success(t *testing.T) {
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
				{"id": "1", "properties": {"dealname": "Newest"}, "createdAt": "2024-01-01T00:00:00.000Z", "updatedAt": "2024-03-01T00:00:00.000Z", "archived": false},
				{"id": "2", "properties": {"dealname": "Older"},  "createdAt": "2024-01-01T00:00:00.000Z", "updatedAt": "2024-02-01T00:00:00.000Z", "archived": false}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetRecentDeals(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetRecentDeals: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/deals/search" {
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

func TestClient_GetRecentDeals_DefaultLimit(t *testing.T) {
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

	if _, err := c.GetRecentDeals(context.Background(), 0); err != nil {
		t.Fatalf("GetRecentDeals: %v", err)
	}
	if !bytes.Contains(gotBody, []byte(`"limit":10`)) {
		t.Fatalf("expected default limit=10 in body, got %s", gotBody)
	}
}

func TestClient_GetRecentDeals_Error(t *testing.T) {
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

	body, err := c.GetRecentDeals(context.Background(), 5)
	if err == nil {
		t.Fatalf("expected error on 500, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_GetDealPipelines_Success(t *testing.T) {
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
				{
					"id": "default",
					"label": "Sales Pipeline",
					"displayOrder": 0,
					"stages": [
						{"id": "appointmentscheduled", "label": "Appointment scheduled", "displayOrder": 0},
						{"id": "closedwon", "label": "Closed won", "displayOrder": 5}
					]
				}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetDealPipelines(context.Background())
	if err != nil {
		t.Fatalf("GetDealPipelines: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/pipelines/deals" {
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
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 pipeline, got %v", obj["results"])
	}
	first, _ := results[0].(map[string]any)
	if first["id"] != "default" {
		t.Fatalf("expected pipeline id=default, got %v", first["id"])
	}
	stages, ok := first["stages"].([]any)
	if !ok || len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %v", first["stages"])
	}
}

func TestClient_GetDealPipelines_Error(t *testing.T) {
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

	body, err := c.GetDealPipelines(context.Background())
	if err == nil {
		t.Fatalf("expected error on 500, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_CreateDeal_Success(t *testing.T) {
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
			"id": "555",
			"properties": {"dealname": "Brand New", "amount": "1000"},
			"createdAt": "2024-01-01T00:00:00.000Z",
			"updatedAt": "2024-01-01T00:00:00.000Z",
			"archived": false
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.CreateDeal(context.Background(), map[string]any{"dealname": "Brand New", "amount": "1000"})
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/deals" {
		t.Fatalf("unexpected create path: %q", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"dealname":"Brand New"`)) {
		t.Fatalf("expected create body to contain dealname=Brand New, got %s", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`"properties":`)) {
		t.Fatalf("expected create body to wrap fields under properties, got %s", gotBody)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["id"] != "555" {
		t.Fatalf("expected id=555, got %v", obj["id"])
	}
}

func TestClient_CreateDeal_GuardArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made when properties are empty")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.CreateDeal(context.Background(), nil); err == nil {
		t.Fatal("expected error on nil properties")
	}
	if _, err := c.CreateDeal(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error on empty properties")
	}
}

func TestClient_CreateDeal_Error(t *testing.T) {
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

	body, err := c.CreateDeal(context.Background(), map[string]any{"dealname": "X"})
	if err == nil {
		t.Fatalf("expected error on 500, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_UpdateDeal_Success(t *testing.T) {
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
			"id": "111",
			"properties": {"dealname": "Renamed", "amount": "9999"},
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

	body, err := c.UpdateDeal(context.Background(), "111", map[string]any{"dealname": "Renamed", "amount": "9999"})
	if err != nil {
		t.Fatalf("UpdateDeal: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/deals/111" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"dealname":"Renamed"`)) {
		t.Fatalf("expected update body to contain dealname=Renamed, got %s", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`"properties":`)) {
		t.Fatalf("expected update body to wrap fields under properties, got %s", gotBody)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	if obj["id"] != "111" {
		t.Fatalf("expected id=111, got %v", obj["id"])
	}
}

func TestClient_UpdateDeal_GuardArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made when guard args are invalid")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.UpdateDeal(context.Background(), "", map[string]any{"dealname": "X"}); err == nil {
		t.Fatal("expected error on empty deal id")
	}
	if _, err := c.UpdateDeal(context.Background(), "1", nil); err == nil {
		t.Fatal("expected error on nil properties")
	}
	if _, err := c.UpdateDeal(context.Background(), "1", map[string]any{}); err == nil {
		t.Fatal("expected error on empty properties")
	}
}

func TestClient_UpdateDeal_Error(t *testing.T) {
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

	body, err := c.UpdateDeal(context.Background(), "111", map[string]any{"dealname": "X"})
	if err == nil {
		t.Fatalf("expected error on 400, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}
