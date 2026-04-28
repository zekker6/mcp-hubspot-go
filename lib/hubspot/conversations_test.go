package hubspot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetRecentConversations_FirstPage(t *testing.T) {
	var (
		gotThreadsPath  string
		gotThreadsQuery string
		gotMessagePaths []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations/v3/conversations/threads":
			gotThreadsPath = r.URL.Path
			gotThreadsQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "t1", "status": "OPEN", "latestMessageReceivedTimestamp": "2024-03-01T00:00:00Z"},
					{"id": "t2", "status": "CLOSED", "latestMessageReceivedTimestamp": "2024-02-28T00:00:00Z"}
				],
				"paging": {"next": {"after": "cursor-NEXT", "link": "https://api.hubapi.com/x"}}
			}`))
		case "/conversations/v3/conversations/threads/t1/messages":
			gotMessagePaths = append(gotMessagePaths, r.URL.Path)
			_, _ = w.Write([]byte(`{"results":[{"id":"m1","text":"hello"}]}`))
		case "/conversations/v3/conversations/threads/t2/messages":
			gotMessagePaths = append(gotMessagePaths, r.URL.Path)
			_, _ = w.Write([]byte(`{"results":[{"id":"m2","text":"world"}]}`))
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

	body, err := c.GetRecentConversations(context.Background(), 5, "cursor-PREV")
	if err != nil {
		t.Fatalf("GetRecentConversations: %v", err)
	}

	if gotThreadsPath != "/conversations/v3/conversations/threads" {
		t.Fatalf("unexpected threads path: %q", gotThreadsPath)
	}
	if !strings.Contains(gotThreadsQuery, "limit=5") {
		t.Fatalf("expected limit=5 in query, got %q", gotThreadsQuery)
	}
	if !strings.Contains(gotThreadsQuery, "after=cursor-PREV") {
		t.Fatalf("expected after=cursor-PREV in query, got %q", gotThreadsQuery)
	}
	if len(gotMessagePaths) != 2 {
		t.Fatalf("expected 2 message fetches, got %d (%v)", len(gotMessagePaths), gotMessagePaths)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (body=%s)", err, body)
	}

	threads, ok := obj["threads"].([]any)
	if !ok || len(threads) != 2 {
		t.Fatalf("expected 2 threads, got %v", obj["threads"])
	}
	for i, raw := range threads {
		thread, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("thread[%d] not object: %T", i, raw)
		}
		msgs, ok := thread["messages"].(map[string]any)
		if !ok {
			t.Fatalf("thread[%d].messages not embedded: %T", i, thread["messages"])
		}
		if _, ok := msgs["results"].([]any); !ok {
			t.Fatalf("thread[%d].messages.results missing: %v", i, msgs)
		}
	}

	paging, ok := obj["paging"].(map[string]any)
	if !ok {
		t.Fatalf("expected paging key, got %v", obj["paging"])
	}
	next, ok := paging["next"].(map[string]any)
	if !ok {
		t.Fatalf("expected paging.next, got %v", paging["next"])
	}
	if next["after"] != "cursor-NEXT" {
		t.Fatalf("expected paging.next.after=cursor-NEXT, got %v", next["after"])
	}
}

func TestClient_GetRecentConversations_LastPageOmitsPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations/v3/conversations/threads":
			_, _ = w.Write([]byte(`{
				"results": [
					{"id": "t1", "status": "OPEN"}
				]
			}`))
		case "/conversations/v3/conversations/threads/t1/messages":
			_, _ = w.Write([]byte(`{"results":[]}`))
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

	body, err := c.GetRecentConversations(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("GetRecentConversations: %v", err)
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("response was not valid JSON: %v (body=%s)", err, body)
	}
	if _, ok := obj["paging"]; ok {
		t.Fatalf("expected no paging key when cursor absent, got %v", obj["paging"])
	}
	threads, ok := obj["threads"].([]any)
	if !ok || len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %v", obj["threads"])
	}
}

func TestClient_GetRecentConversations_DefaultLimit(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/conversations/v3/conversations/threads" {
			gotQuery = r.URL.RawQuery
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.GetRecentConversations(context.Background(), 0, ""); err != nil {
		t.Fatalf("GetRecentConversations: %v", err)
	}
	if !strings.Contains(gotQuery, "limit=10") {
		t.Fatalf("expected default limit=10 in query, got %q", gotQuery)
	}
	if strings.Contains(gotQuery, "after=") {
		t.Fatalf("expected no after param when caller passed empty, got %q", gotQuery)
	}
}

func TestClient_GetRecentConversations_MessagesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations/v3/conversations/threads":
			_, _ = w.Write([]byte(`{"results":[{"id":"t1"}]}`))
		case "/conversations/v3/conversations/threads/t1/messages":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"error","message":"boom"}`))
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

	body, err := c.GetRecentConversations(context.Background(), 5, "")
	if err == nil {
		t.Fatalf("expected error when message fetch fails, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
	if !strings.Contains(err.Error(), "thread t1") {
		t.Fatalf("expected error to identify failing thread, got %v", err)
	}
}

func TestClient_GetRecentConversations_ListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"status":"error","message":"upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("token", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := c.GetRecentConversations(context.Background(), 5, "")
	if err == nil {
		t.Fatalf("expected error when list call fails, got body=%s", body)
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %s", body)
	}
}
