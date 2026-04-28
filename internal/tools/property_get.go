package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetProperty is the canonical MCP tool name for fetching a property.
const ToolNameGetProperty = "hubspot_get_property"

const (
	propertyObjectTypeCompanies = "companies"
	propertyObjectTypeContacts  = "contacts"
)

type propertyGetter interface {
	GetProperty(ctx context.Context, objectType, propertyName string) ([]byte, error)
}

// NewGetPropertyTool returns the tool definition for hubspot_get_property.
func NewGetPropertyTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetProperty,
		mcp.WithDescription("Retrieve metadata for a single HubSpot property by name. Restricted to companies and contacts object types."),
		mcp.WithString("object_type",
			mcp.Description("HubSpot object type. Must be \"companies\" or \"contacts\"."),
			mcp.Required(),
			mcp.Enum(propertyObjectTypeCompanies, propertyObjectTypeContacts),
		),
		mcp.WithString("property_name",
			mcp.Description("The property name (internal name) to fetch."),
			mcp.Required(),
		),
	)
}

// GetPropertyHandler binds a propertyGetter to the hubspot_get_property tool handler.
func GetPropertyHandler(c propertyGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		objectType, err := RequiredStringArg(req, "object_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		propertyName, err := RequiredStringArg(req, "property_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body, err := c.GetProperty(ctx, objectType, propertyName)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
