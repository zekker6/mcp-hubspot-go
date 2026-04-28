package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameCreateCompany is the canonical MCP tool name for creating a company.
const ToolNameCreateCompany = "hubspot_create_company"

type companyCreator interface {
	CreateCompany(ctx context.Context, properties map[string]any) ([]byte, error)
}

// NewCreateCompanyTool returns the tool definition for hubspot_create_company.
func NewCreateCompanyTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameCreateCompany,
		mcp.WithDescription("Create a HubSpot company. Performs a pre-flight name-exact-match search; if a match exists, returns the existing record(s) under \"matches\" with \"duplicate\": true and skips creation."),
		mcp.WithObject("properties",
			mcp.Description("HubSpot company properties. Must include \"name\". Additional properties (domain, website, industry, etc.) are passed through to HubSpot."),
			mcp.Required(),
		),
	)
}

// CreateCompanyHandler binds a companyCreator to the hubspot_create_company tool handler.
func CreateCompanyHandler(c companyCreator) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		properties, err := OptionalObjectArg(req, "properties")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(properties) == 0 {
			return mcp.NewToolResultError(`argument "properties" is required`), nil
		}
		name, _ := properties["name"].(string)
		if name == "" {
			return mcp.NewToolResultError(`argument "properties.name" is required`), nil
		}

		body, err := c.CreateCompany(ctx, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
