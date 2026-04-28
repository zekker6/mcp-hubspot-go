package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetDealPipelines is the canonical MCP tool name for fetching all
// HubSpot deal pipelines (with their stages).
const ToolNameGetDealPipelines = "hubspot_get_deal_pipelines"

type dealPipelinesGetter interface {
	GetDealPipelines(ctx context.Context) ([]byte, error)
}

// NewGetDealPipelinesTool returns the tool definition for hubspot_get_deal_pipelines.
func NewGetDealPipelinesTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetDealPipelines,
		mcp.WithDescription("List all HubSpot deal pipelines along with their stages. Useful for interpreting raw deal stage IDs returned by hubspot_get_deal and hubspot_get_active_deals."),
	)
}

// GetDealPipelinesHandler binds a dealPipelinesGetter to the
// hubspot_get_deal_pipelines tool handler.
func GetDealPipelinesHandler(c dealPipelinesGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := c.GetDealPipelines(ctx)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
