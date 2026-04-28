package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	propertyObjectTypeCompanies = "companies"
	propertyObjectTypeContacts  = "contacts"
)

// supportedPropertyObjectTypes lists the object types accepted by GetProperty.
// HubSpot supports more object types (deals, tickets, etc.), but the tool
// surface scoped to v1 covers companies and contacts only.
var supportedPropertyObjectTypes = map[string]bool{
	propertyObjectTypeCompanies: true,
	propertyObjectTypeContacts:  true,
}

// GetProperty fetches metadata for a single property on the given object type.
// objectType must be either "companies" or "contacts"; other types return an
// error without hitting the API. Uses the SDK's typed CRM.Properties client.
func (c *Client) GetProperty(_ context.Context, objectType, propertyName string) ([]byte, error) {
	if propertyName == "" {
		return nil, fmt.Errorf("property name is required")
	}
	if !supportedPropertyObjectTypes[objectType] {
		return nil, fmt.Errorf("unsupported object type %q: must be %q or %q",
			objectType, propertyObjectTypeCompanies, propertyObjectTypeContacts)
	}

	res, err := c.sdk.CRM.Properties.Get(objectType, propertyName)
	if err != nil {
		return nil, fmt.Errorf("get %s property %s: %w", objectType, propertyName, err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal property response: %w", err)
	}
	return out, nil
}

// CreateProperty creates a new HubSpot property on the given object type.
// objectType must be one of the supported types ("companies" or "contacts").
// All of name, label, propertyType, fieldType, groupName are required and
// passed through to HubSpot verbatim. options is forwarded as-is when
// non-empty; HubSpot decides whether the field type requires options (no
// client-side validation).
func (c *Client) CreateProperty(_ context.Context, objectType, name, label, propertyType, fieldType, groupName string, options []any) ([]byte, error) {
	if !supportedPropertyObjectTypes[objectType] {
		return nil, fmt.Errorf("unsupported object type %q: must be %q or %q",
			objectType, propertyObjectTypeCompanies, propertyObjectTypeContacts)
	}
	if name == "" {
		return nil, fmt.Errorf("property name is required")
	}
	if label == "" {
		return nil, fmt.Errorf("property label is required")
	}
	if propertyType == "" {
		return nil, fmt.Errorf("property type is required")
	}
	if fieldType == "" {
		return nil, fmt.Errorf("property field type is required")
	}
	if groupName == "" {
		return nil, fmt.Errorf("property group name is required")
	}

	reqData := map[string]any{
		"name":      name,
		"label":     label,
		"type":      propertyType,
		"fieldType": fieldType,
		"groupName": groupName,
	}
	if len(options) > 0 {
		reqData["options"] = options
	}

	res, err := c.sdk.CRM.Properties.Create(objectType, reqData)
	if err != nil {
		return nil, fmt.Errorf("create %s property %s: %w", objectType, name, err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal create property response: %w", err)
	}
	return out, nil
}

// UpdateProperty updates the supplied fields on an existing HubSpot property.
// Only the keys present in fields are modified; others are left untouched.
func (c *Client) UpdateProperty(_ context.Context, objectType, propertyName string, fields map[string]any) ([]byte, error) {
	if !supportedPropertyObjectTypes[objectType] {
		return nil, fmt.Errorf("unsupported object type %q: must be %q or %q",
			objectType, propertyObjectTypeCompanies, propertyObjectTypeContacts)
	}
	if propertyName == "" {
		return nil, fmt.Errorf("property name is required")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}

	res, err := c.sdk.CRM.Properties.Update(objectType, propertyName, fields)
	if err != nil {
		return nil, fmt.Errorf("update %s property %s: %w", objectType, propertyName, err)
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal update property response: %w", err)
	}
	return out, nil
}
