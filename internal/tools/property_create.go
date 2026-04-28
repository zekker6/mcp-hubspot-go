package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameCreateProperty is the canonical MCP tool name for creating a property.
const ToolNameCreateProperty = "hubspot_create_property"

type propertyCreator interface {
	CreateProperty(ctx context.Context, objectType, name, label, propertyType, fieldType, groupName string, options []any) ([]byte, error)
}

// NewCreatePropertyTool returns the tool definition for hubspot_create_property.
func NewCreatePropertyTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameCreateProperty,
		mcp.WithDescription("Create a new HubSpot property on a companies or contacts object. Supply HubSpot's property definition: internal name, human label, property type (e.g. \"string\", \"enumeration\"), field type (e.g. \"text\", \"select\"), group name, and an optional options array for enumeration/select fields."),
		mcp.WithString("object_type",
			mcp.Description("HubSpot object type. Must be \"companies\" or \"contacts\"."),
			mcp.Required(),
			mcp.Enum(propertyObjectTypeCompanies, propertyObjectTypeContacts),
		),
		mcp.WithString("name",
			mcp.Description("Internal name of the property."),
			mcp.Required(),
		),
		mcp.WithString("label",
			mcp.Description("Human-readable label."),
			mcp.Required(),
		),
		mcp.WithString("property_type",
			mcp.Description("HubSpot property type, e.g. \"string\", \"number\", \"enumeration\", \"date\", \"datetime\", \"bool\"."),
			mcp.Required(),
		),
		mcp.WithString("field_type",
			mcp.Description("HubSpot field type, e.g. \"text\", \"textarea\", \"select\", \"radio\", \"checkbox\", \"booleancheckbox\", \"number\", \"date\", \"file\"."),
			mcp.Required(),
		),
		mcp.WithString("group_name",
			mcp.Description("Property group name to bucket this property under."),
			mcp.Required(),
		),
		mcp.WithArray("options",
			mcp.Description("Optional list of options for enumeration/select-style field types. Each entry is an object with at least \"label\" and \"value\"."),
			mcp.Items(map[string]any{"type": "object"}),
		),
	)
}

// CreatePropertyHandler binds a propertyCreator to the hubspot_create_property tool handler.
func CreatePropertyHandler(c propertyCreator) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		objectType, err := RequiredStringArg(req, "object_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := RequiredStringArg(req, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		label, err := RequiredStringArg(req, "label")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		propertyType, err := RequiredStringArg(req, "property_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fieldType, err := RequiredStringArg(req, "field_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		groupName, err := RequiredStringArg(req, "group_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		options, err := optionalObjectArrayArg(req, "options")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body, err := c.CreateProperty(ctx, objectType, name, label, propertyType, fieldType, groupName, options)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}

// optionalObjectArrayArg returns the value of a JSON-array argument as []any:
//   - nil when absent
//   - the slice when present and an array
//   - error when present but not an array
//
// This is local to property_create because it is the only tool currently
// passing through a free-form array of objects; if more tools need it, lift
// it into common.go.
func optionalObjectArrayArg(req mcp.CallToolRequest, key string) ([]any, error) {
	args := req.GetArguments()
	if args == nil {
		return nil, nil
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("argument %q must be a JSON array", key)
	}
	return arr, nil
}
