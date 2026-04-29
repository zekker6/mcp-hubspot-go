package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-hubspot-go/lib/hubspot"
)

// toolEntry binds an MCP tool definition to a handler factory taking a
// HubSpot client. A single ordered list per category drives both registration
// and the test-visible name lists, so the two can never desync.
type toolEntry struct {
	Name    string
	Tool    func() mcp.Tool
	Handler func(c *hubspot.Client) server.ToolHandlerFunc
}

var readTools = []toolEntry{
	{ToolNameGetCompany, NewGetCompanyTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetCompanyHandler(c) }},
	{ToolNameGetRecentCompanies, NewGetRecentCompaniesTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetRecentCompaniesHandler(c) }},
	{ToolNameSearchCompanies, NewSearchCompaniesTool, func(c *hubspot.Client) server.ToolHandlerFunc { return SearchCompaniesHandler(c) }},
	{ToolNameGetCompanyActivity, NewGetCompanyActivityTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetCompanyActivityHandler(c) }},
	{ToolNameGetContact, NewGetContactTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetContactHandler(c) }},
	{ToolNameGetRecentContacts, NewGetRecentContactsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetRecentContactsHandler(c) }},
	{ToolNameSearchContacts, NewSearchContactsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return SearchContactsHandler(c) }},
	{ToolNameGetRecentConversations, NewGetRecentConversationsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetRecentConversationsHandler(c) }},
	{ToolNameGetTickets, NewGetTicketsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetTicketsHandler(c) }},
	{ToolNameSearchTickets, NewSearchTicketsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return SearchTicketsHandler(c) }},
	{ToolNameGetTicketConversationThreads, NewGetTicketConversationThreadsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetTicketConversationThreadsHandler(c) }},
	{ToolNameGetProperty, NewGetPropertyTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetPropertyHandler(c) }},
	{ToolNameGetDeal, NewGetDealTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetDealHandler(c) }},
	{ToolNameGetRecentDeals, NewGetRecentDealsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetRecentDealsHandler(c) }},
	{ToolNameSearchDeals, NewSearchDealsTool, func(c *hubspot.Client) server.ToolHandlerFunc { return SearchDealsHandler(c) }},
	{ToolNameGetDealPipelines, NewGetDealPipelinesTool, func(c *hubspot.Client) server.ToolHandlerFunc { return GetDealPipelinesHandler(c) }},
}

var writeTools = []toolEntry{
	{ToolNameCreateCompany, NewCreateCompanyTool, func(c *hubspot.Client) server.ToolHandlerFunc { return CreateCompanyHandler(c) }},
	{ToolNameUpdateCompany, NewUpdateCompanyTool, func(c *hubspot.Client) server.ToolHandlerFunc { return UpdateCompanyHandler(c) }},
	{ToolNameCreateContact, NewCreateContactTool, func(c *hubspot.Client) server.ToolHandlerFunc { return CreateContactHandler(c) }},
	{ToolNameUpdateContact, NewUpdateContactTool, func(c *hubspot.Client) server.ToolHandlerFunc { return UpdateContactHandler(c) }},
	{ToolNameCreateProperty, NewCreatePropertyTool, func(c *hubspot.Client) server.ToolHandlerFunc { return CreatePropertyHandler(c) }},
	{ToolNameUpdateProperty, NewUpdatePropertyTool, func(c *hubspot.Client) server.ToolHandlerFunc { return UpdatePropertyHandler(c) }},
	{ToolNameCreateDeal, NewCreateDealTool, func(c *hubspot.Client) server.ToolHandlerFunc { return CreateDealHandler(c) }},
	{ToolNameUpdateDeal, NewUpdateDealTool, func(c *hubspot.Client) server.ToolHandlerFunc { return UpdateDealHandler(c) }},
}

// RegisterTools registers every MCP tool backed by the given HubSpot client.
// Read tools are always registered. Write tools are skipped when readOnly is
// true. Returns the total number of tools registered, useful for startup
// logging.
func RegisterTools(s *server.MCPServer, c *hubspot.Client, readOnly bool) int {
	for _, t := range readTools {
		s.AddTool(t.Tool(), t.Handler(c))
	}
	count := len(readTools)
	if !readOnly {
		for _, t := range writeTools {
			s.AddTool(t.Tool(), t.Handler(c))
		}
		count += len(writeTools)
	}
	return count
}

// ReadToolNames returns the list of read tool names always registered.
func ReadToolNames() []string {
	names := make([]string, len(readTools))
	for i, t := range readTools {
		names[i] = t.Name
	}
	return names
}

// WriteToolNames returns the list of write tool names registered when not in
// read-only mode.
func WriteToolNames() []string {
	names := make([]string, len(writeTools))
	for i, t := range writeTools {
		names[i] = t.Name
	}
	return names
}
