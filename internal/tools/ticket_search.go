package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameSearchTickets is the canonical MCP tool name for full-text
// searching HubSpot tickets via the CRM Search API.
const ToolNameSearchTickets = "hubspot_search_tickets"

const defaultSearchTicketsToolLimit = 10

type searchTicketsGetter interface {
	SearchTickets(ctx context.Context, query string, limit int, properties []string, after string) ([]byte, error)
}

// NewSearchTicketsTool returns the tool definition for hubspot_search_tickets.
func NewSearchTicketsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameSearchTickets,
		mcp.WithDescription("Full-text search HubSpot tickets by query against indexed properties (subject, content). Use this when hubspot_get_tickets does not return the ticket you are looking for, e.g. when searching by ticket subject or content. Returns matches sorted by relevance."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Free-text query matched against indexed ticket properties (subject, content)."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of tickets to return. Defaults to 10."),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional ticket properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("after",
			mcp.Description("Pagination cursor returned by HubSpot in paging.next.after on a previous response."),
		),
	)
}

// SearchTicketsHandler binds a searchTicketsGetter to the
// hubspot_search_tickets tool handler.
func SearchTicketsHandler(c searchTicketsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := RequiredStringArg(req, "query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := OptionalIntArg(req, "limit", defaultSearchTicketsToolLimit)
		properties := OptionalStringArrayArg(req, "properties")
		after := OptionalStringArg(req, "after", "")

		body, err := c.SearchTickets(ctx, query, limit, properties, after)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
