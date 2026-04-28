package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameCreateDeal is the canonical MCP tool name for creating a deal.
const ToolNameCreateDeal = "hubspot_create_deal"

type dealCreator interface {
	CreateDeal(ctx context.Context, properties map[string]any) ([]byte, error)
}

// NewCreateDealTool returns the tool definition for hubspot_create_deal.
func NewCreateDealTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameCreateDeal,
		mcp.WithDescription("Create a HubSpot deal. No duplicate pre-flight is performed - deal names are not unique by convention."),
		mcp.WithObject("properties",
			mcp.Description("HubSpot deal properties (e.g. dealname, amount, dealstage, pipeline). Passed through to HubSpot as-is."),
			mcp.Required(),
		),
	)
}

// CreateDealHandler binds a dealCreator to the hubspot_create_deal tool handler.
func CreateDealHandler(c dealCreator) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		properties, err := OptionalObjectArg(req, "properties")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(properties) == 0 {
			return mcp.NewToolResultError(`argument "properties" is required`), nil
		}

		body, err := c.CreateDeal(ctx, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
