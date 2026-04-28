package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetCompanyActivity is the canonical MCP tool name for fetching the
// engagements (notes, calls, emails, tasks, meetings) associated with a company.
const ToolNameGetCompanyActivity = "hubspot_get_company_activity"

type companyActivityGetter interface {
	GetCompanyActivity(ctx context.Context, companyID string) ([]byte, error)
}

// NewGetCompanyActivityTool returns the tool definition for hubspot_get_company_activity.
func NewGetCompanyActivityTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetCompanyActivity,
		mcp.WithDescription("Retrieve the engagements (notes, calls, emails, tasks, meetings) associated with a HubSpot company."),
		mcp.WithString("company_id",
			mcp.Description("The HubSpot company ID."),
			mcp.Required(),
		),
	)
}

// GetCompanyActivityHandler binds a companyActivityGetter to the
// hubspot_get_company_activity tool handler.
func GetCompanyActivityHandler(c companyActivityGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "company_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body, err := c.GetCompanyActivity(ctx, id)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
