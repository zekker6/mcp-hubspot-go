package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolNameCreateContact is the canonical MCP tool name for creating a contact.
const ToolNameCreateContact = "hubspot_create_contact"

type contactCreator interface {
	CreateContact(ctx context.Context, properties map[string]any) ([]byte, error)
}

// NewCreateContactTool returns the tool definition for hubspot_create_contact.
func NewCreateContactTool() mcp.Tool {
	return mcp.NewTool(
		ToolNameCreateContact,
		mcp.WithDescription("Create a HubSpot contact. Performs a pre-flight email-exact-match search; if a match exists, returns the existing record(s) under \"matches\" with \"duplicate\": true and skips creation."),
		mcp.WithObject("properties",
			mcp.Description("HubSpot contact properties. Must include \"email\". Additional properties (firstname, lastname, phone, etc.) are passed through to HubSpot."),
			mcp.Required(),
		),
	)
}

// CreateContactHandler binds a contactCreator to the hubspot_create_contact tool handler.
func CreateContactHandler(c contactCreator) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		properties, err := OptionalObjectArg(req, "properties")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(properties) == 0 {
			return mcp.NewToolResultError(`argument "properties" is required`), nil
		}
		email, _ := properties["email"].(string)
		if email == "" {
			return mcp.NewToolResultError(`argument "properties.email" is required`), nil
		}

		body, err := c.CreateContact(ctx, properties)
		if err != nil {
			return APIError(err), nil
		}
		return JSONResult(body), nil
	}
}
