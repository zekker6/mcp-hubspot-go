package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetRecentConversations is the canonical MCP tool name for fetching
// the most recent HubSpot conversation threads with their messages embedded.
const ToolNameGetRecentConversations = "hubspot_get_recent_conversations"

const defaultRecentConversationsToolLimit = 10

type recentConversationsGetter interface {
	GetRecentConversations(ctx context.Context, limit int, after string) ([]byte, error)
}

// NewGetRecentConversationsTool returns the tool definition for
// hubspot_get_recent_conversations.
func NewGetRecentConversationsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetRecentConversations,
		mcp.WithDescription("List the most recent HubSpot conversation threads with their messages embedded. Returns {threads, paging}; paging.next.after passes through unchanged when more pages remain."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of threads to return. Defaults to 10."),
		),
		mcp.WithString("after",
			mcp.Description("Pagination cursor from a previous response's paging.next.after."),
		),
	)
}

// GetRecentConversationsHandler binds a recentConversationsGetter to the
// hubspot_get_recent_conversations tool handler.
func GetRecentConversationsHandler(c recentConversationsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := OptionalIntArg(req, "limit", defaultRecentConversationsToolLimit)
		after := OptionalStringArg(req, "after", "")

		body, err := c.GetRecentConversations(ctx, limit, after)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
