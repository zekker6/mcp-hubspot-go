package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetDeal is the canonical MCP tool name for fetching a deal.
const ToolNameGetDeal = "hubspot_get_deal"

type dealGetter interface {
	GetDeal(ctx context.Context, id string, properties []string) ([]byte, error)
}

// NewGetDealTool returns the tool definition for hubspot_get_deal.
func NewGetDealTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetDeal,
		mcp.WithDescription("Retrieve a HubSpot deal by its ID. Optionally include extra properties beyond the default field set."),
		mcp.WithString("deal_id",
			mcp.Description("The HubSpot deal ID."),
			mcp.Required(),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional deal properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
	)
}

// GetDealHandler binds a dealGetter to the hubspot_get_deal tool handler.
func GetDealHandler(c dealGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "deal_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		properties := OptionalStringArrayArg(req, "properties")

		body, err := c.GetDeal(ctx, id, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
