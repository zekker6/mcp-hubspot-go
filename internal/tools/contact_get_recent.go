package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetRecentContacts is the canonical MCP tool name for fetching the
// most recently modified HubSpot contacts.
const ToolNameGetRecentContacts = "hubspot_get_active_contacts"

const defaultRecentContactsToolLimit = 10

type recentContactsGetter interface {
	GetRecentContacts(ctx context.Context, limit int) ([]byte, error)
}

// NewGetRecentContactsTool returns the tool definition for hubspot_get_active_contacts.
func NewGetRecentContactsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetRecentContacts,
		mcp.WithDescription("List the most recently modified HubSpot contacts, sorted by lastmodifieddate descending."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of contacts to return. Defaults to 10."),
		),
	)
}

// GetRecentContactsHandler binds a recentContactsGetter to the
// hubspot_get_active_contacts tool handler.
func GetRecentContactsHandler(c recentContactsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := OptionalIntArg(req, "limit", defaultRecentContactsToolLimit)

		body, err := c.GetRecentContacts(ctx, limit)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
