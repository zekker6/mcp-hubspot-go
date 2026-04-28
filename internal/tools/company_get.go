package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetCompany is the canonical MCP tool name for fetching a company.
const ToolNameGetCompany = "hubspot_get_company"

type companyGetter interface {
	GetCompany(ctx context.Context, id string, properties []string) ([]byte, error)
}

// NewGetCompanyTool returns the tool definition for hubspot_get_company.
func NewGetCompanyTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetCompany,
		mcp.WithDescription("Retrieve a HubSpot company by its ID. Optionally include extra properties beyond the default field set."),
		mcp.WithString("company_id",
			mcp.Description("The HubSpot company ID."),
			mcp.Required(),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional company properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
	)
}

// GetCompanyHandler binds a companyGetter to the hubspot_get_company tool handler.
func GetCompanyHandler(c companyGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "company_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		properties := OptionalStringArrayArg(req, "properties")

		body, err := c.GetCompany(ctx, id, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
