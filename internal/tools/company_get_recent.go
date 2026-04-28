package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetRecentCompanies is the canonical MCP tool name for fetching the
// most recently modified HubSpot companies.
const ToolNameGetRecentCompanies = "hubspot_get_active_companies"

const defaultRecentCompaniesToolLimit = 10

type recentCompaniesGetter interface {
	GetRecentCompanies(ctx context.Context, limit int) ([]byte, error)
}

// NewGetRecentCompaniesTool returns the tool definition for hubspot_get_active_companies.
func NewGetRecentCompaniesTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetRecentCompanies,
		mcp.WithDescription("List the most recently modified HubSpot companies, sorted by hs_lastmodifieddate descending."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of companies to return. Defaults to 10."),
		),
	)
}

// GetRecentCompaniesHandler binds a recentCompaniesGetter to the
// hubspot_get_active_companies tool handler.
func GetRecentCompaniesHandler(c recentCompaniesGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := OptionalIntArg(req, "limit", defaultRecentCompaniesToolLimit)

		body, err := c.GetRecentCompanies(ctx, limit)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
