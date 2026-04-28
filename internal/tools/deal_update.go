package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameUpdateDeal is the canonical MCP tool name for updating a deal.
const ToolNameUpdateDeal = "hubspot_update_deal"

type dealUpdater interface {
	UpdateDeal(ctx context.Context, id string, properties map[string]any) ([]byte, error)
}

// NewUpdateDealTool returns the tool definition for hubspot_update_deal.
func NewUpdateDealTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameUpdateDeal,
		mcp.WithDescription("Update properties on an existing HubSpot deal. Only the properties supplied are modified; others are left untouched."),
		mcp.WithString("deal_id",
			mcp.Description("The HubSpot deal ID."),
			mcp.Required(),
		),
		mcp.WithObject("properties",
			mcp.Description("HubSpot deal properties to update."),
			mcp.Required(),
		),
	)
}

// UpdateDealHandler binds a dealUpdater to the hubspot_update_deal tool handler.
func UpdateDealHandler(c dealUpdater) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "deal_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		properties, err := OptionalObjectArg(req, "properties")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(properties) == 0 {
			return mcp.NewToolResultError(`argument "properties" is required`), nil
		}

		body, err := c.UpdateDeal(ctx, id, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
