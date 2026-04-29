package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameSearchDeals is the canonical MCP tool name for full-text
// searching HubSpot deals via the CRM Search API.
const ToolNameSearchDeals = "hubspot_search_deals"

const defaultSearchDealsToolLimit = 10

type searchDealsGetter interface {
	SearchDeals(ctx context.Context, query string, limit int, properties []string, after string) ([]byte, error)
}

// NewSearchDealsTool returns the tool definition for hubspot_search_deals.
func NewSearchDealsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameSearchDeals,
		mcp.WithDescription("Full-text search HubSpot deals by query against indexed properties (dealname). Use this when hubspot_get_active_deals does not return the deal you are looking for, e.g. when searching by deal name. Returns matches sorted by relevance."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Free-text query matched against indexed deal properties (dealname)."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of deals to return. Defaults to 10."),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional deal properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("after",
			mcp.Description("Pagination cursor returned by HubSpot in paging.next.after on a previous response."),
		),
	)
}

// SearchDealsHandler binds a searchDealsGetter to the
// hubspot_search_deals tool handler.
func SearchDealsHandler(c searchDealsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := RequiredStringArg(req, "query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := OptionalIntArg(req, "limit", defaultSearchDealsToolLimit)
		properties := OptionalStringArrayArg(req, "properties")
		after := OptionalStringArg(req, "after", "")

		body, err := c.SearchDeals(ctx, query, limit, properties, after)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
