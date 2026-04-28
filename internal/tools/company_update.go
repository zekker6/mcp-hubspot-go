package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameUpdateCompany is the canonical MCP tool name for updating a company.
const ToolNameUpdateCompany = "hubspot_update_company"

type companyUpdater interface {
	UpdateCompany(ctx context.Context, id string, properties map[string]any) ([]byte, error)
}

// NewUpdateCompanyTool returns the tool definition for hubspot_update_company.
func NewUpdateCompanyTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameUpdateCompany,
		mcp.WithDescription("Update properties on an existing HubSpot company. Only the properties supplied are modified; others are left untouched."),
		mcp.WithString("company_id",
			mcp.Description("The HubSpot company ID."),
			mcp.Required(),
		),
		mcp.WithObject("properties",
			mcp.Description("HubSpot company properties to update."),
			mcp.Required(),
		),
	)
}

// UpdateCompanyHandler binds a companyUpdater to the hubspot_update_company tool handler.
func UpdateCompanyHandler(c companyUpdater) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "company_id")
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

		body, err := c.UpdateCompany(ctx, id, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
