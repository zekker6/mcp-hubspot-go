package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameSearchContacts is the canonical MCP tool name for full-text
// searching HubSpot contacts via the CRM Search API.
const ToolNameSearchContacts = "hubspot_search_contacts"

const defaultSearchContactsToolLimit = 10

type searchContactsGetter interface {
	SearchContacts(ctx context.Context, query string, limit int, properties []string, after string) ([]byte, error)
}

// NewSearchContactsTool returns the tool definition for hubspot_search_contacts.
func NewSearchContactsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameSearchContacts,
		mcp.WithDescription("Full-text search HubSpot contacts by query against indexed properties (firstname, lastname, email, company). Use this when hubspot_get_active_contacts does not return the contact you are looking for, e.g. when searching by email or name. Returns matches sorted by relevance."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Free-text query matched against indexed contact properties (firstname, lastname, email, company)."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of contacts to return. Defaults to 10."),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional contact properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("after",
			mcp.Description("Pagination cursor returned by HubSpot in paging.next.after on a previous response."),
		),
	)
}

// SearchContactsHandler binds a searchContactsGetter to the
// hubspot_search_contacts tool handler.
func SearchContactsHandler(c searchContactsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := RequiredStringArg(req, "query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := OptionalIntArg(req, "limit", defaultSearchContactsToolLimit)
		properties := OptionalStringArrayArg(req, "properties")
		after := OptionalStringArg(req, "after", "")

		body, err := c.SearchContacts(ctx, query, limit, properties, after)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
