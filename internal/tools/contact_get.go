package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameGetContact is the canonical MCP tool name for fetching a contact.
const ToolNameGetContact = "hubspot_get_contact"

type contactGetter interface {
	GetContact(ctx context.Context, id string, properties []string) ([]byte, error)
}

// NewGetContactTool returns the tool definition for hubspot_get_contact.
func NewGetContactTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameGetContact,
		mcp.WithDescription("Retrieve a HubSpot contact by its ID. Optionally include extra properties beyond the default field set."),
		mcp.WithString("contact_id",
			mcp.Description("The HubSpot contact ID."),
			mcp.Required(),
		),
		mcp.WithArray("properties",
			mcp.Description("Optional list of additional contact properties to include in the response."),
			mcp.Items(map[string]any{"type": "string"}),
		),
	)
}

// GetContactHandler binds a contactGetter to the hubspot_get_contact tool handler.
func GetContactHandler(c contactGetter) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := RequiredStringArg(req, "contact_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		properties := OptionalStringArrayArg(req, "properties")

		body, err := c.GetContact(ctx, id, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
