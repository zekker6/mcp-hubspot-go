package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameSearchCompanies is the canonical MCP tool name for full-text
// searching HubSpot companies via the CRM Search API.
const ToolNameSearchCompanies = "hubspot_search_companies"

const defaultSearchCompaniesToolLimit = 10

type searchCompaniesGetter interface {
	SearchCompanies(ctx context.Context, query string, limit int, properties []string, after string) ([]byte, error)
}

// NewSearchCompaniesTool returns the tool definition for hubspot_search_companies.
func NewSearchCompaniesTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameSearchCompanies,
		mcp.WithDescription("Full-text search HubSpot companies by query against indexed properties (name, domain, website, phone). Use this when hubspot_get_active_companies does not return the company you are looking for, e.g. when searching by name or domain. Returns matches sorted by relevance."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Free-text query matched against indexed company properties (name, domain, website, phone)."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of companies to return. Defaults to 10."),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional company properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("after",
			mcp.Description("Pagination cursor returned by HubSpot in paging.next.after on a previous response."),
		),
	)
}

// SearchCompaniesHandler binds a searchCompaniesGetter to the
// hubspot_search_companies tool handler.
func SearchCompaniesHandler(c searchCompaniesGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := RequiredStringArg(req, "query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := OptionalIntArg(req, "limit", defaultSearchCompaniesToolLimit)
		properties := OptionalStringArrayArg(req, "properties")
		after := OptionalStringArg(req, "after", "")

		body, err := c.SearchCompanies(ctx, query, limit, properties, after)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
