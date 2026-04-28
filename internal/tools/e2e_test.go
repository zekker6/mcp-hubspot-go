package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-hubspot-go/internal/tools"
	"github.com/zekker6/mcp-hubspot-go/lib/hubspot"
)

// fakeHubSpot wraps an httptest server that mimics the HubSpot REST API for
// the endpoints exercised by the registered MCP tools. The handlers serve
// fixtures from the testdata/ directory; behavior switches based on request
// shape (filter groups, paths, query params).
type fakeHubSpot struct {
	srv *httptest.Server
}

func newFakeHubSpot(t *testing.T) *fakeHubSpot {
	t.Helper()
	f := &fakeHubSpot{}
	mux := http.NewServeMux()

	// --- companies ---
	mux.HandleFunc("/crm/v3/objects/companies/search", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		// SearchByName posts filterGroups with a "name" filter; recent posts
		// only sorts. Differentiate by inspecting the body. The "Existing Co"
		// sentinel name flips the search to return a duplicate match so the
		// duplicate-passthrough path can be exercised end-to-end.
		if hasFilterOnProperty(body, "name") {
			if hasFilterValue(body, "name", "Existing Co") {
				writeFixture(t, w, "company_search_match.json")
				return
			}
			writeFixture(t, w, "company_search_empty.json")
			return
		}
		writeFixture(t, w, "recent_companies.json")
	})
	mux.HandleFunc("/crm/v3/objects/companies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeFixture(t, w, "company_create_echo.json")
	})
	mux.HandleFunc("/crm/v3/objects/companies/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			id := strings.TrimPrefix(r.URL.Path, "/crm/v3/objects/companies/")
			if id == "force-500" {
				w.WriteHeader(http.StatusInternalServerError)
				writeFixture(t, w, "error_500.json")
				return
			}
			writeFixture(t, w, "company_by_id.json")
		case http.MethodPatch:
			writeFixture(t, w, "company_update_echo.json")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// --- engagements (company activity) ---
	mux.HandleFunc("/engagements/v1/engagements/associated/COMPANY/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeFixture(t, w, "company_activity.json")
	})

	// --- contacts ---
	mux.HandleFunc("/crm/v3/objects/contacts/search", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if hasFilterOnProperty(body, "email") {
			if hasFilterValue(body, "email", "existing@acme.example") {
				writeFixture(t, w, "contact_search_match.json")
				return
			}
			writeFixture(t, w, "contact_search_empty.json")
			return
		}
		writeFixture(t, w, "recent_contacts.json")
	})
	mux.HandleFunc("/crm/v3/objects/contacts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeFixture(t, w, "contact_create_echo.json")
	})
	mux.HandleFunc("/crm/v3/objects/contacts/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			writeFixture(t, w, "contact_by_id.json")
		case http.MethodPatch:
			writeFixture(t, w, "contact_update_echo.json")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// --- conversations ---
	mux.HandleFunc("/conversations/v3/conversations/threads", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("after") == "cursor-NEXT" {
			writeFixture(t, w, "conversations_page_last.json")
			return
		}
		writeFixture(t, w, "conversations_page_with_cursor.json")
	})
	mux.HandleFunc("/conversations/v3/conversations/threads/", func(w http.ResponseWriter, r *http.Request) {
		// Path is /conversations/v3/conversations/threads/<id>/messages.
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/messages") {
			http.NotFound(w, r)
			return
		}
		writeFixture(t, w, "conversation_thread_messages.json")
	})

	// --- tickets ---
	mux.HandleFunc("/crm/v3/objects/tickets/search", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if hasFilterOnProperty(body, "hs_pipeline_stage") {
			writeFixture(t, w, "tickets_closed.json")
			return
		}
		writeFixture(t, w, "tickets_default.json")
	})
	mux.HandleFunc("/crm/v4/objects/tickets/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/associations/conversation") {
			http.NotFound(w, r)
			return
		}
		writeFixture(t, w, "ticket_associations.json")
	})

	// --- properties ---
	mux.HandleFunc("/crm/v3/properties/companies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// POST to create.
		if r.Method == http.MethodPost {
			writeFixture(t, w, "property_create_echo.json")
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/crm/v3/properties/companies/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			writeFixture(t, w, "property_companies_name.json")
		case http.MethodPatch:
			writeFixture(t, w, "property_update_echo.json")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// --- deals ---
	mux.HandleFunc("/crm/v3/objects/deals/search", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeFixture(t, w, "recent_deals.json")
	})
	mux.HandleFunc("/crm/v3/objects/deals", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeFixture(t, w, "deal_create_echo.json")
	})
	mux.HandleFunc("/crm/v3/objects/deals/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			writeFixture(t, w, "deal_by_id.json")
		case http.MethodPatch:
			writeFixture(t, w, "deal_update_echo.json")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// --- deal pipelines ---
	mux.HandleFunc("/crm/v3/pipelines/deals", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeFixture(t, w, "deal_pipelines.json")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("fakeHubSpot: unexpected request %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	})

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// hasFilterOnProperty reports whether a HubSpot search request body has a
// filterGroup containing a filter on the named property. Used to differentiate
// search-by-name/email pre-flights from sort-only "recent" searches that share
// the same endpoint.
func hasFilterOnProperty(body []byte, propName string) bool {
	var req struct {
		FilterGroups []struct {
			Filters []struct {
				PropertyName string `json:"propertyName"`
			} `json:"filters"`
		} `json:"filterGroups"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	for _, g := range req.FilterGroups {
		for _, f := range g.Filters {
			if f.PropertyName == propName {
				return true
			}
		}
	}
	return false
}

// hasFilterValue is hasFilterOnProperty with a value match - used to flip the
// fake response between "no match" and "duplicate match" based on the search
// term posted by the create tool.
func hasFilterValue(body []byte, propName, value string) bool {
	var req struct {
		FilterGroups []struct {
			Filters []struct {
				PropertyName string `json:"propertyName"`
				Value        string `json:"value"`
			} `json:"filters"`
		} `json:"filterGroups"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	for _, g := range req.FilterGroups {
		for _, f := range g.Filters {
			if f.PropertyName == propName && f.Value == value {
				return true
			}
		}
	}
	return false
}

func writeFixture(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Errorf("read fixture %q: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

// newServer builds an MCP server wired to the fake HubSpot instance, with read
// tools always registered and write tools gated by readOnly.
func newServer(t *testing.T, fake *fakeHubSpot, readOnly bool) *server.MCPServer {
	t.Helper()
	hsClient, err := hubspot.NewClient("test-token", hubspot.WithBaseURL(fake.srv.URL))
	if err != nil {
		t.Fatalf("hubspot.NewClient: %v", err)
	}
	s := server.NewMCPServer(
		"hubspot-manager",
		"v0.1.0-test",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	tools.RegisterTools(s, hsClient, readOnly)
	return s
}

// initializeClient starts an in-process client against s and performs the MCP
// initialize handshake. Returns the connected client; the caller is
// responsible for Close.
func initializeClient(t *testing.T, s *server.MCPServer) *mcpclient.Client {
	t.Helper()
	c, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("client.Start: %v", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "e2e-test", Version: "1.0.0"}
	if _, err := c.Initialize(t.Context(), initReq); err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}
	return c
}

func toolNameSet(tools []mcp.Tool) map[string]bool {
	m := make(map[string]bool, len(tools))
	for _, t := range tools {
		m[t.Name] = true
	}
	return m
}

// callTool drives a tool/call and returns the result. The test fails on
// transport errors; tool-level errors come back via result.IsError.
func callTool(t *testing.T, c *mcpclient.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(t.Context(), req)
	if err != nil {
		t.Fatalf("CallTool(%q): %v", name, err)
	}
	if res == nil {
		t.Fatalf("CallTool(%q): nil result", name)
	}
	return res
}

func textContent(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("expected exactly 1 content item, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected mcp.TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

func TestE2E_ReadOnly_RegistersOnlyReadTools(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, true)
	c := initializeClient(t, s)

	listResp, err := c.ListTools(t.Context(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(listResp.Tools) != 12 {
		names := make([]string, 0, len(listResp.Tools))
		for _, tool := range listResp.Tools {
			names = append(names, tool.Name)
		}
		t.Fatalf("expected 12 read-only tools, got %d: %v", len(listResp.Tools), names)
	}

	got := toolNameSet(listResp.Tools)
	for _, want := range tools.ReadToolNames() {
		if !got[want] {
			t.Errorf("missing read tool: %s", want)
		}
	}
	for _, banned := range tools.WriteToolNames() {
		if got[banned] {
			t.Errorf("read-only mode should NOT register write tool: %s", banned)
		}
	}
}

func TestE2E_ReadOnly_RejectsAllWriteToolCalls(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, true)
	c := initializeClient(t, s)

	// Provide minimum valid args for each write tool so the failure mode is
	// "tool not found", not "missing arg".
	args := map[string]map[string]any{
		"hubspot_create_company":  {"properties": map[string]any{"name": "X"}},
		"hubspot_update_company":  {"company_id": "1", "properties": map[string]any{"name": "X"}},
		"hubspot_create_contact":  {"properties": map[string]any{"email": "x@y.example"}},
		"hubspot_update_contact":  {"contact_id": "1", "properties": map[string]any{"email": "x@y.example"}},
		"hubspot_create_property": {"object_type": "companies", "name": "n", "label": "L", "property_type": "string", "field_type": "text", "group_name": "g"},
		"hubspot_update_property": {"object_type": "companies", "property_name": "n", "fields": map[string]any{"label": "X"}},
		"hubspot_create_deal":     {"properties": map[string]any{"dealname": "X"}},
		"hubspot_update_deal":     {"deal_id": "1", "properties": map[string]any{"dealname": "X"}},
	}
	for _, name := range tools.WriteToolNames() {
		t.Run(name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Name = name
			req.Params.Arguments = args[name]
			_, err := c.CallTool(t.Context(), req)
			if err == nil {
				t.Fatalf("expected unknown-tool error for %q in read-only mode", name)
			}
			// mark3labs returns an error containing "tool '<name>' not found".
			if !strings.Contains(err.Error(), "not found") {
				t.Fatalf("expected 'not found' error for %q, got %v", name, err)
			}
		})
	}
}

func TestE2E_FullMode_ListsAllTwentyTools(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, false)
	c := initializeClient(t, s)

	listResp, err := c.ListTools(t.Context(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(listResp.Tools) != 20 {
		names := make([]string, 0, len(listResp.Tools))
		for _, tool := range listResp.Tools {
			names = append(names, tool.Name)
		}
		t.Fatalf("expected 20 tools in full mode, got %d: %v", len(listResp.Tools), names)
	}

	got := toolNameSet(listResp.Tools)
	for _, want := range append(tools.ReadToolNames(), tools.WriteToolNames()...) {
		if !got[want] {
			t.Errorf("full mode is missing tool: %s", want)
		}
	}
}

func TestE2E_FullMode_RoundTripsEveryTool(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, false)
	c := initializeClient(t, s)

	// Per-tool minimal valid args. Each call must succeed (IsError=false) and
	// return a single TextContent whose body parses as JSON.
	cases := []struct {
		name string
		args map[string]any
	}{
		{tools.ToolNameGetCompany, map[string]any{"company_id": "1001"}},
		{tools.ToolNameGetRecentCompanies, map[string]any{"limit": 5}},
		{tools.ToolNameGetCompanyActivity, map[string]any{"company_id": "1001"}},
		{tools.ToolNameGetContact, map[string]any{"contact_id": "2001"}},
		{tools.ToolNameGetRecentContacts, map[string]any{"limit": 5}},
		{tools.ToolNameGetRecentConversations, map[string]any{"limit": 2}},
		{tools.ToolNameGetTickets, map[string]any{"criteria": "default", "limit": 10}},
		{tools.ToolNameGetTicketConversationThreads, map[string]any{"ticket_id": "T1"}},
		{tools.ToolNameGetProperty, map[string]any{"object_type": "companies", "property_name": "name"}},
		{tools.ToolNameGetDeal, map[string]any{"deal_id": "3001"}},
		{tools.ToolNameGetRecentDeals, map[string]any{"limit": 5}},
		{tools.ToolNameGetDealPipelines, map[string]any{}},
		{tools.ToolNameCreateCompany, map[string]any{"properties": map[string]any{"name": "New Co", "domain": "newco.example"}}},
		{tools.ToolNameUpdateCompany, map[string]any{"company_id": "1001", "properties": map[string]any{"name": "Acme Corp Updated"}}},
		{tools.ToolNameCreateContact, map[string]any{"properties": map[string]any{"email": "new@acme.example"}}},
		{tools.ToolNameUpdateContact, map[string]any{"contact_id": "2001", "properties": map[string]any{"firstname": "Alicia"}}},
		{tools.ToolNameCreateProperty, map[string]any{"object_type": "companies", "name": "loyalty_tier", "label": "Loyalty Tier", "property_type": "enumeration", "field_type": "select", "group_name": "companyinformation"}},
		{tools.ToolNameUpdateProperty, map[string]any{"object_type": "companies", "property_name": "loyalty_tier", "fields": map[string]any{"label": "Loyalty Tier (Renamed)"}}},
		{tools.ToolNameCreateDeal, map[string]any{"properties": map[string]any{"dealname": "New Deal"}}},
		{tools.ToolNameUpdateDeal, map[string]any{"deal_id": "3001", "properties": map[string]any{"amount": "13500"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := callTool(t, c, tc.name, tc.args)
			if res.IsError {
				t.Fatalf("tool %q returned error: %s", tc.name, textContent(t, res))
			}
			body := textContent(t, res)
			var v any
			if err := json.Unmarshal([]byte(body), &v); err != nil {
				t.Fatalf("tool %q output not JSON: %v\nbody=%s", tc.name, err, body)
			}
		})
	}
}

func TestE2E_DuplicatePassthrough(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, false)
	c := initializeClient(t, s)

	cases := []struct {
		name string
		tool string
		args map[string]any
	}{
		{
			name: "company",
			tool: tools.ToolNameCreateCompany,
			args: map[string]any{"properties": map[string]any{"name": "Existing Co"}},
		},
		{
			name: "contact",
			tool: tools.ToolNameCreateContact,
			args: map[string]any{"properties": map[string]any{"email": "existing@acme.example"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := callTool(t, c, tc.tool, tc.args)
			if res.IsError {
				t.Fatalf("%s create returned error: %s", tc.name, textContent(t, res))
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(textContent(t, res)), &body); err != nil {
				t.Fatalf("%s response not JSON: %v", tc.name, err)
			}
			if dup, ok := body["duplicate"].(bool); !ok || !dup {
				t.Fatalf("expected duplicate:true in response, got %v", body)
			}
			matches, ok := body["matches"].([]any)
			if !ok || len(matches) == 0 {
				t.Fatalf("expected non-empty matches array, got %v", body)
			}
		})
	}
}

func TestE2E_PaginationPassthrough(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, false)
	c := initializeClient(t, s)

	// First page: HubSpot returns paging.next.after=cursor-NEXT - verify the
	// tool surfaces it unchanged.
	res := callTool(t, c, tools.ToolNameGetRecentConversations, map[string]any{"limit": 5})
	if res.IsError {
		t.Fatalf("first-page conversations call errored: %s", textContent(t, res))
	}
	var firstPage map[string]any
	if err := json.Unmarshal([]byte(textContent(t, res)), &firstPage); err != nil {
		t.Fatalf("first page not JSON: %v", err)
	}
	paging, ok := firstPage["paging"].(map[string]any)
	if !ok {
		t.Fatalf("first page missing paging key: %v", firstPage)
	}
	next, ok := paging["next"].(map[string]any)
	if !ok {
		t.Fatalf("first page missing paging.next: %v", paging)
	}
	if next["after"] != "cursor-NEXT" {
		t.Fatalf("expected paging.next.after=cursor-NEXT, got %v", next["after"])
	}

	// Last page: feed cursor back; HubSpot omits paging - verify the tool
	// output also omits the paging key entirely.
	res = callTool(t, c, tools.ToolNameGetRecentConversations, map[string]any{"limit": 5, "after": "cursor-NEXT"})
	if res.IsError {
		t.Fatalf("last-page conversations call errored: %s", textContent(t, res))
	}
	var lastPage map[string]any
	if err := json.Unmarshal([]byte(textContent(t, res)), &lastPage); err != nil {
		t.Fatalf("last page not JSON: %v", err)
	}
	if _, present := lastPage["paging"]; present {
		t.Fatalf("last page should NOT contain paging key, got %v", lastPage["paging"])
	}
}

func TestE2E_ErrorRoundTrip(t *testing.T) {
	fake := newFakeHubSpot(t)
	s := newServer(t, fake, false)
	c := initializeClient(t, s)

	res := callTool(t, c, tools.ToolNameGetCompany, map[string]any{"company_id": "force-500"})
	if !res.IsError {
		t.Fatalf("expected IsError=true on 500 fixture, got %s", textContent(t, res))
	}
	body := textContent(t, res)
	if !strings.HasPrefix(body, "HubSpot API error: ") {
		t.Fatalf("expected error body to start with %q, got %q", "HubSpot API error: ", body)
	}
}

func TestE2E_TransportSSEHandshake(t *testing.T) {
	s := server.NewMCPServer("test", "v0.0.1-test", server.WithToolCapabilities(false))
	sseServer := server.NewSSEServer(s, server.WithKeepAliveInterval(30*time.Second))
	listener := httptest.NewServer(sseServer)
	t.Cleanup(listener.Close)

	url := listener.URL + sseServer.CompleteSsePath()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Cancellation while reading the never-ending SSE body is expected;
		// only fail if the connection itself failed.
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Do: %v", err)
		}
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content-type, got %q", ct)
	}
}

func TestE2E_TransportHTTPHandshake(t *testing.T) {
	s := server.NewMCPServer("test", "v0.0.1-test", server.WithToolCapabilities(false))
	httpServer := server.NewStreamableHTTPServer(s, server.WithHeartbeatInterval(30*time.Second))
	// httptest.NewServer mounts the StreamableHTTPServer as the root handler,
	// matching how Start() routes its endpointPath. We POST to the root URL -
	// the production wiring routes the same handler at the configured path.
	listener := httptest.NewServer(httpServer)
	t.Cleanup(listener.Close)

	initBody := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "` + mcp.LATEST_PROTOCOL_VERSION + `",
			"clientInfo": {"name":"e2e-test","version":"1.0.0"},
			"capabilities": {}
		}
	}`
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, listener.URL, strings.NewReader(initBody))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	body, err := readJSONRPCBody(resp)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	var rpc struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &rpc); err != nil {
		t.Fatalf("response body not JSON-RPC: %v\nbody=%s", err, body)
	}
	if rpc.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc=2.0, got %q", rpc.JSONRPC)
	}
	if rpc.ID != 1 {
		t.Fatalf("expected id=1, got %d", rpc.ID)
	}
	if len(rpc.Error) > 0 && string(rpc.Error) != "null" {
		t.Fatalf("unexpected JSON-RPC error: %s", rpc.Error)
	}
	if len(rpc.Result) == 0 {
		t.Fatalf("expected JSON-RPC result, got empty")
	}
}

// readJSONRPCBody reads either an application/json body or extracts the first
// `data:` line from a text/event-stream response. The streamable HTTP
// transport may respond with either depending on negotiation.
func readJSONRPCBody(resp *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "text/event-stream") {
		// SSE frame: split by lines, return first line that starts with "data:".
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				return []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), nil
			}
		}
		return nil, fmt.Errorf("no data line in SSE response: %s", raw)
	}
	return raw, nil
}
