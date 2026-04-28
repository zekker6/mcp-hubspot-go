package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameUpdateProperty is the canonical MCP tool name for updating a property.
const ToolNameUpdateProperty = "hubspot_update_property"

type propertyUpdater interface {
	UpdateProperty(ctx context.Context, objectType, propertyName string, fields map[string]any) ([]byte, error)
}

// NewUpdatePropertyTool returns the tool definition for hubspot_update_property.
func NewUpdatePropertyTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameUpdateProperty,
		mcp.WithDescription("Update fields on an existing HubSpot property. Only the supplied keys are modified; others are left unchanged. Updatable keys include label, type, fieldType, groupName, options, description, displayOrder, hidden."),
		mcp.WithString("object_type",
			mcp.Description("HubSpot object type. Must be \"companies\" or \"contacts\"."),
			mcp.Required(),
			mcp.Enum(propertyObjectTypeCompanies, propertyObjectTypeContacts),
		),
		mcp.WithString("property_name",
			mcp.Description("Internal name of the property to update."),
			mcp.Required(),
		),
		mcp.WithObject("fields",
			mcp.Description("Property fields to update, passed through to HubSpot verbatim."),
			mcp.Required(),
		),
	)
}

// UpdatePropertyHandler binds a propertyUpdater to the hubspot_update_property tool handler.
func UpdatePropertyHandler(c propertyUpdater) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		objectType, err := RequiredStringArg(req, "object_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		propertyName, err := RequiredStringArg(req, "property_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fields, err := OptionalObjectArg(req, "fields")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(fields) == 0 {
			return mcp.NewToolResultError(`argument "fields" is required`), nil
		}

		body, err := c.UpdateProperty(ctx, objectType, propertyName, fields)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
