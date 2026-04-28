package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetTicketConversationThreads is the canonical MCP tool name for
// fetching conversation threads associated with a ticket.
const ToolNameGetTicketConversationThreads = "hubspot_get_ticket_conversation_threads"

type ticketThreadsGetter interface {
	GetTicketConversationThreads(ctx context.Context, ticketID string) ([]byte, error)
}

// NewGetTicketConversationThreadsTool returns the tool definition for
// hubspot_get_ticket_conversation_threads.
func NewGetTicketConversationThreadsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetTicketConversationThreads,
		mcp.WithDescription("Return the conversation threads (with their messages) associated with a HubSpot ticket. System messages are filtered out; only entries with type=MESSAGE are returned."),
		mcp.WithString("ticket_id",
			mcp.Description("The HubSpot ticket ID."),
			mcp.Required(),
		),
	)
}

// GetTicketConversationThreadsHandler binds a ticketThreadsGetter to the tool handler.
func GetTicketConversationThreadsHandler(c ticketThreadsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "ticket_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := c.GetTicketConversationThreads(ctx, id)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
