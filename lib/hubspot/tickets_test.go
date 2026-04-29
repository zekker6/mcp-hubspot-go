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
	"time"
)

func TestClient_GetTickets_DefaultCriteria(t *testing.T) {
	fixed := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	prev := nowMinus24h
	nowMinus24h = func() time.Time { return fixed }
	t.Cleanup(func() { nowMinus24h = prev })

	expectedTimestamp := "2024-03-15T12:00:00Z"

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
			"total": 1,
			"results": [{"id":"42","properties":{"subject":"Hi"}}],
			"paging": {"next": {"after": "cursor-NEXT"}}
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetTickets(context.Background(), "default", 25)
	if err != nil {
		t.Fatalf("GetTickets: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/tickets/search" {
		t.Fatalf("unexpected path: %q", gotPath)
	}

	var sent ticketSearchRequest
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("request body not JSON: %v (%s)", err, gotBody)
	}
	if len(sent.FilterGroups) != 2 {
		t.Fatalf("expected 2 filter groups for default criteria, got %d (%s)", len(sent.FilterGroups), gotBody)
	}
	if sent.FilterGroups[0].Filters[0].PropertyName != "closedate" || sent.FilterGroups[0].Filters[0].Operator != "GT" {
		t.Fatalf("unexpected filter group 0: %+v", sent.FilterGroups[0].Filters[0])
	}
	if sent.FilterGroups[0].Filters[0].Value != expectedTimestamp {
		t.Fatalf("expected default closedate value=%q, got %q", expectedTimestamp, sent.FilterGroups[0].Filters[0].Value)
	}
	if sent.FilterGroups[1].Filters[0].PropertyName != "hs_lastmodifieddate" {
		t.Fatalf("expected filter group 1 to filter on hs_lastmodifieddate, got %+v", sent.FilterGroups[1].Filters[0])
	}
	if sent.Limit != 25 {
		t.Fatalf("expected limit=25, got %d", sent.Limit)
	}
	if len(sent.Sorts) != 1 || sent.Sorts[0].PropertyName != "hs_lastmodifieddate" || sent.Sorts[0].Direction != "DESCENDING" {
		t.Fatalf("expected hs_lastmodifieddate DESCENDING sort, got %+v", sent.Sorts)
	}
	wantProps := map[string]bool{
		"subject": true, "content": true, "hs_pipeline": true, "hs_pipeline_stage": true,
		"hs_ticket_status": true, "status": true, "hs_ticket_priority": true,
		"createdate": true, "closedate": true, "hs_lastmodifieddate": true,
	}
	for _, p := range sent.Properties {
		delete(wantProps, p)
	}
	if len(wantProps) != 0 {
		t.Fatalf("missing properties in request: %v (got %v)", wantProps, sent.Properties)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response not JSON: %v (%s)", err, body)
	}
	if obj["total"].(float64) != 1 {
		t.Fatalf("expected total=1, got %v", obj["total"])
	}
}

func TestClient_GetTickets_ClosedCriteria(t *testing.T) {
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

	if _, err := c.GetTickets(context.Background(), "Closed", 50); err != nil {
		t.Fatalf("GetTickets: %v", err)
	}

	var sent ticketSearchRequest
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("request body not JSON: %v", err)
	}
	if len(sent.FilterGroups) != 2 {
		t.Fatalf("expected 2 filter groups for Closed criteria, got %d", len(sent.FilterGroups))
	}
	for i, want := range []string{"4", "Closed"} {
		f := sent.FilterGroups[i].Filters[0]
		if f.PropertyName != "hs_pipeline_stage" || f.Operator != "EQ" || f.Value != want {
			t.Fatalf("filter group %d: expected hs_pipeline_stage EQ %q, got %+v", i, want, f)
		}
	}
}

func TestClient_GetTickets_DefaultLimit(t *testing.T) {
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

	if _, err := c.GetTickets(context.Background(), "", 0); err != nil {
		t.Fatalf("GetTickets: %v", err)
	}
	var sent ticketSearchRequest
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("request body not JSON: %v", err)
	}
	if sent.Limit != defaultTicketsLimit {
		t.Fatalf("expected default limit=%d, got %d", defaultTicketsLimit, sent.Limit)
	}
	// empty criteria should resolve to default (two filter groups)
	if len(sent.FilterGroups) != 2 {
		t.Fatalf("expected 2 filter groups for empty criteria→default, got %d", len(sent.FilterGroups))
	}
}

func TestClient_GetTickets_InvalidCriteria(t *testing.T) {
	c, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.GetTickets(context.Background(), "bogus", 10)
	if err == nil {
		t.Fatal("expected error for invalid criteria")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("expected error to mention invalid criteria, got %v", err)
	}
}

func TestClient_GetTickets_ServerError(t *testing.T) {
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

	body, err := c.GetTickets(context.Background(), "default", 10)
	if err == nil {
		t.Fatalf("expected error, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_GetTicketConversationThreads_Success(t *testing.T) {
	var (
		gotAssocPath    string
		gotMessagePaths []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/crm/v4/objects/tickets/T1/associations/conversation":
			gotAssocPath = r.URL.Path
			_, _ = w.Write([]byte(`{
				"results": [
					{"toObjectId": 1001},
					{"toObjectId": 1002}
				]
			}`))
		case "/conversations/v3/conversations/threads/1001/messages":
			gotMessagePaths = append(gotMessagePaths, r.URL.Path)
			_, _ = w.Write([]byte(`{
				"results": [
					{"id":"m1","type":"MESSAGE","text":"hello"},
					{"id":"s1","type":"SYSTEM_MESSAGE","text":"system"}
				]
			}`))
		case "/conversations/v3/conversations/threads/1002/messages":
			gotMessagePaths = append(gotMessagePaths, r.URL.Path)
			_, _ = w.Write([]byte(`{
				"results": [
					{"id":"m2","type":"MESSAGE","text":"world"}
				]
			}`))
		default:
			t.Errorf("unexpected request path: %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetTicketConversationThreads(context.Background(), "T1")
	if err != nil {
		t.Fatalf("GetTicketConversationThreads: %v", err)
	}
	if gotAssocPath != "/crm/v4/objects/tickets/T1/associations/conversation" {
		t.Fatalf("unexpected associations path: %q", gotAssocPath)
	}
	if len(gotMessagePaths) != 2 {
		t.Fatalf("expected 2 message fetches, got %v", gotMessagePaths)
	}

	var out ticketThreadsOutput
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("response not JSON: %v (%s)", err, body)
	}
	if out.TicketID != "T1" {
		t.Fatalf("expected ticket_id=T1, got %q", out.TicketID)
	}
	if out.TotalThreads != 2 {
		t.Fatalf("expected total_threads=2, got %d", out.TotalThreads)
	}
	// thread 1 should drop the SYSTEM_MESSAGE entry, leaving 1 MESSAGE.
	if len(out.Threads[0].Messages) != 1 {
		t.Fatalf("expected thread[0] to have 1 message after filtering, got %d", len(out.Threads[0].Messages))
	}
	if out.TotalMessages != 2 {
		t.Fatalf("expected total_messages=2 (1+1 after filter), got %d", out.TotalMessages)
	}
	if out.Threads[0].ID != "1001" {
		t.Fatalf("expected first thread id=1001, got %q", out.Threads[0].ID)
	}
}

func TestClient_GetTicketConversationThreads_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/crm/v4/objects/tickets/T9/associations/conversation" {
			_, _ = w.Write([]byte(`{"results":[]}`))
			return
		}
		t.Errorf("unexpected request path: %q", r.URL.Path)
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetTicketConversationThreads(context.Background(), "T9")
	if err != nil {
		t.Fatalf("GetTicketConversationThreads: %v", err)
	}
	var out ticketThreadsOutput
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("response not JSON: %v (%s)", err, body)
	}
	if out.TotalThreads != 0 || out.TotalMessages != 0 {
		t.Fatalf("expected zero-counts, got threads=%d messages=%d", out.TotalThreads, out.TotalMessages)
	}
	if len(out.Threads) != 0 {
		t.Fatalf("expected empty threads slice, got %d", len(out.Threads))
	}
}

func TestClient_GetTicketConversationThreads_FallbackID(t *testing.T) {
	// Some upstream payloads use `id` instead of `toObjectId` (string form).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/crm/v4/objects/tickets/T2/associations/conversation":
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "7777"}
				]
			}`))
		case "/conversations/v3/conversations/threads/7777/messages":
			_, _ = w.Write([]byte(`{"results":[{"id":"m1","type":"MESSAGE"}]}`))
		default:
			t.Errorf("unexpected request path: %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetTicketConversationThreads(context.Background(), "T2")
	if err != nil {
		t.Fatalf("GetTicketConversationThreads: %v", err)
	}
	var out ticketThreadsOutput
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("response not JSON: %v (%s)", err, body)
	}
	if out.TotalThreads != 1 || out.Threads[0].ID != "7777" {
		t.Fatalf("expected fallback id=7777, got %+v", out.Threads)
	}
}

func TestClient_GetTicketConversationThreads_EmptyTicketID(t *testing.T) {
	c, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetTicketConversationThreads(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty ticket id")
	}
}

func TestClient_GetTicketConversationThreads_AssocError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","message":"not found"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetTicketConversationThreads(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body, got %s", body)
	}
}

func TestClient_GetTicketConversationThreads_MessagesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/crm/v4/objects/tickets/T3/associations/conversation":
			_, _ = w.Write([]byte(`{"results":[{"toObjectId":111}]}`))
		case "/conversations/v3/conversations/threads/111/messages":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"error","message":"boom"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	body, err := c.GetTicketConversationThreads(context.Background(), "T3")
	if err == nil {
		t.Fatalf("expected error, got body=%s", body)
	}
	if !strings.Contains(err.Error(), "thread 111") {
		t.Fatalf("expected error to identify failing thread, got %v", err)
	}
}

func TestClient_SearchTickets_Success(t *testing.T) {
	type sentBody struct {
		Query        string   `json:"query"`
		Limit        int      `json:"limit"`
		After        string   `json:"after"`
		Properties   []string `json:"properties"`
		Sorts        []any    `json:"sorts"`
		FilterGroups []any    `json:"filterGroups"`
	}

	var (
		gotMethod string
		gotPath   string
		gotBody   sentBody
		raw       []byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ = io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total": 1,
			"results": [
				{"id": "501", "properties": {"subject": "Outage on Acme"}, "createdAt": "2024-01-01T00:00:00.000Z", "updatedAt": "2024-01-02T00:00:00.000Z", "archived": false}
			],
			"paging": {"next": {"after": "10"}}
		}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.SearchTickets(context.Background(), "outage", 25, []string{"subject", "content", "custom"}, "")
	if err != nil {
		t.Fatalf("SearchTickets: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/crm/v3/objects/tickets/search" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if gotBody.Query != "outage" {
		t.Fatalf("expected query=outage, got %q (raw=%s)", gotBody.Query, raw)
	}
	if gotBody.Limit != 25 {
		t.Fatalf("expected limit=25, got %d (raw=%s)", gotBody.Limit, raw)
	}
	if len(gotBody.Sorts) != 0 {
		t.Fatalf("expected no sorts in search request (relevance ordering), got %v (raw=%s)", gotBody.Sorts, raw)
	}
	if len(gotBody.FilterGroups) != 0 {
		t.Fatalf("expected no filterGroups in search request, got %v (raw=%s)", gotBody.FilterGroups, raw)
	}
	if bytes.Contains(raw, []byte(`"sorts"`)) {
		t.Fatalf("expected sorts key to be absent from search request body, got %s", raw)
	}
	if bytes.Contains(raw, []byte(`"filterGroups"`)) {
		t.Fatalf("expected filterGroups key to be absent from search request body, got %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"query"`)) {
		t.Fatalf("expected query key in search request body, got %s", raw)
	}
	if got := gotBody.Properties; len(got) != 3 || got[0] != "subject" || got[1] != "content" || got[2] != "custom" {
		t.Fatalf("expected properties [subject content custom], got %v (raw=%s)", got, raw)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (%s)", err, body)
	}
	results, ok := obj["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", obj["results"])
	}
}

func TestClient_SearchTickets_EmptyQueryRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP request should be made when query is empty")
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.SearchTickets(context.Background(), "", 10, nil, "")
	if err == nil {
		t.Fatalf("expected error on empty query, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}

func TestClient_SearchTickets_LimitClamping(t *testing.T) {
	cases := []struct {
		name      string
		input     int
		wantLimit int
	}{
		{name: "zero defaults to 10", input: 0, wantLimit: 10},
		{name: "negative defaults to 10", input: -1, wantLimit: 10},
		{name: "passthrough 1", input: 1, wantLimit: 1},
		{name: "passthrough 50", input: 50, wantLimit: 50},
		{name: "passthrough 100", input: 100, wantLimit: 100},
		{name: "passthrough 200", input: 200, wantLimit: 200},
		{name: "clamped to 200", input: 500, wantLimit: 200},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"total":0,"results":[]}`))
			}))
			t.Cleanup(srv.Close)

			c, err := NewClient("token", WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			if _, err := c.SearchTickets(context.Background(), "outage", tc.input, nil, ""); err != nil {
				t.Fatalf("SearchTickets: %v", err)
			}

			var sent struct {
				Limit int `json:"limit"`
			}
			if err := json.Unmarshal(gotBody, &sent); err != nil {
				t.Fatalf("decode body: %v (%s)", err, gotBody)
			}
			if sent.Limit != tc.wantLimit {
				t.Fatalf("expected limit=%d in body, got %d (raw=%s)", tc.wantLimit, sent.Limit, gotBody)
			}
		})
	}
}

func TestClient_SearchTickets_AfterCursorPassedThrough(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.SearchTickets(context.Background(), "outage", 5, nil, "cursor-42"); err != nil {
		t.Fatalf("SearchTickets: %v", err)
	}

	var sent struct {
		After string `json:"after"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("decode body: %v (%s)", err, gotBody)
	}
	if sent.After != "cursor-42" {
		t.Fatalf("expected after=cursor-42, got %q (raw=%s)", sent.After, gotBody)
	}
}

func TestClient_SearchTickets_AfterCursorEmptyOmitted(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.SearchTickets(context.Background(), "outage", 5, nil, ""); err != nil {
		t.Fatalf("SearchTickets: %v", err)
	}

	if bytes.Contains(gotBody, []byte(`"after"`)) {
		t.Fatalf("expected after to be omitted from request body when cursor is empty, got %s", gotBody)
	}
	if bytes.Contains(gotBody, []byte(`"filterGroups"`)) {
		t.Fatalf("expected filterGroups to be omitted from search request body, got %s", gotBody)
	}
}

func TestClient_SearchTickets_PropertiesPassedThrough(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.SearchTickets(context.Background(), "outage", 5, []string{"subject", "content", "hs_pipeline_stage"}, ""); err != nil {
		t.Fatalf("SearchTickets: %v", err)
	}

	var sent struct {
		Properties []string `json:"properties"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("decode body: %v (%s)", err, gotBody)
	}
	if len(sent.Properties) != 3 || sent.Properties[0] != "subject" || sent.Properties[1] != "content" || sent.Properties[2] != "hs_pipeline_stage" {
		t.Fatalf("expected properties [subject content hs_pipeline_stage], got %v (raw=%s)", sent.Properties, gotBody)
	}
}

func TestClient_SearchTickets_Error(t *testing.T) {
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

	body, err := c.SearchTickets(context.Background(), "outage", 5, nil, "")
	if err == nil {
		t.Fatalf("expected error on 500, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}
