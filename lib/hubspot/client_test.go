package hubspot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNewClient_EmptyTokenReturnsError(t *testing.T) {
	c, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if c != nil {
		t.Fatalf("expected nil client on error, got %#v", c)
	}
}

func TestNewClient_DefaultConstruction(t *testing.T) {
	c, err := NewClient("token-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.sdk == nil {
		t.Fatal("expected non-nil client and underlying sdk")
	}
}

func TestNewClient_InvalidBaseURLReturnsError(t *testing.T) {
	c, err := NewClient("token-123", WithBaseURL("://bad-url"))
	if err == nil {
		t.Fatal("expected error for invalid base url, got nil")
	}
	if c != nil {
		t.Fatal("expected nil client on error")
	}
}

func TestNewClient_OptionsRouteRequestsThroughOverrides(t *testing.T) {
	var hits int32
	var gotAuth string
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	httpClient := &http.Client{Timeout: 0}

	c, err := NewClient("token-abc",
		WithBaseURL(srv.URL),
		WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var dst map[string]any
	if err := c.sdk.Get("/probe", &dst, nil); err != nil {
		t.Fatalf("sdk.Get failed: %v", err)
	}

	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected exactly one request to fixture server, got %d", got)
	}
	if v, ok := dst["ok"].(bool); !ok || !v {
		t.Fatalf("unexpected response payload: %#v", dst)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") || !strings.HasSuffix(gotAuth, "token-abc") {
		t.Fatalf("unexpected Authorization header: %q", gotAuth)
	}
	if gotPath != "/probe" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
}
