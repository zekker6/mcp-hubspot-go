package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetRecentDeals is the canonical MCP tool name for fetching the most
// recently modified HubSpot deals.
const ToolNameGetRecentDeals = "hubspot_get_active_deals"

const defaultRecentDealsToolLimit = 10

type recentDealsGetter interface {
	GetRecentDeals(ctx context.Context, limit int) ([]byte, error)
}

// NewGetRecentDealsTool returns the tool definition for hubspot_get_active_deals.
func NewGetRecentDealsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetRecentDeals,
		mcp.WithDescription("List the most recently modified HubSpot deals, sorted by hs_lastmodifieddate descending."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of deals to return. Defaults to 10."),
		),
	)
}

// GetRecentDealsHandler binds a recentDealsGetter to the
// hubspot_get_active_deals tool handler.
func GetRecentDealsHandler(c recentDealsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := OptionalIntArg(req, "limit", defaultRecentDealsToolLimit)

		body, err := c.GetRecentDeals(ctx, limit)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
