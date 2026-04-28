package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetTickets is the canonical MCP tool name for searching HubSpot tickets.
const ToolNameGetTickets = "hubspot_get_tickets"

const (
	defaultTicketsToolLimit    = 50
	ticketsCriteriaDefaultName = "default"
	ticketsCriteriaClosedName  = "Closed"
)

type ticketsGetter interface {
	GetTickets(ctx context.Context, criteria string, limit int) ([]byte, error)
}

// NewGetTicketsTool returns the tool definition for hubspot_get_tickets.
func NewGetTicketsTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetTickets,
		mcp.WithDescription("Search HubSpot tickets by criteria. \"default\" returns tickets whose closedate or hs_lastmodifieddate are within the last 24 hours; \"Closed\" returns tickets in the Closed pipeline stage."),
		mcp.WithString("criteria",
			mcp.Description("Selection criteria. Defaults to \"default\"."),
			mcp.Enum(ticketsCriteriaDefaultName, ticketsCriteriaClosedName),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of tickets to return. Defaults to 50."),
		),
	)
}

// GetTicketsHandler binds a ticketsGetter to the hubspot_get_tickets tool handler.
func GetTicketsHandler(c ticketsGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		criteria := OptionalStringArg(req, "criteria", ticketsCriteriaDefaultName)
		limit := OptionalIntArg(req, "limit", defaultTicketsToolLimit)

		body, err := c.GetTickets(ctx, criteria, limit)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
