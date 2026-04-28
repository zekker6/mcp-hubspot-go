package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameUpdateContact is the canonical MCP tool name for updating a contact.
const ToolNameUpdateContact = "hubspot_update_contact"

type contactUpdater interface {
	UpdateContact(ctx context.Context, id string, properties map[string]any) ([]byte, error)
}

// NewUpdateContactTool returns the tool definition for hubspot_update_contact.
func NewUpdateContactTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameUpdateContact,
		mcp.WithDescription("Update properties on an existing HubSpot contact. Only the properties supplied are modified; others are left untouched."),
		mcp.WithString("contact_id",
			mcp.Description("The HubSpot contact ID."),
			mcp.Required(),
		),
		mcp.WithObject("properties",
			mcp.Description("HubSpot contact properties to update."),
			mcp.Required(),
		),
	)
}

// UpdateContactHandler binds a contactUpdater to the hubspot_update_contact tool handler.
func UpdateContactHandler(c contactUpdater) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "contact_id")
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

		body, err := c.UpdateContact(ctx, id, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
