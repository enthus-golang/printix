package printix

import (
	"context"
	"fmt"
	"net/http"
)

// Tenant represents a Printix tenant.
type Tenant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]any         `json:"properties,omitempty"`
	Links       map[string]interface{} `json:"_links,omitempty"`
}

// TenantsResponse represents the HAL+JSON response from the root endpoint.
type TenantsResponse struct {
	Links    map[string]interface{} `json:"_links"`
	Success  bool                   `json:"success"`
	Message  string                 `json:"message,omitempty"`
	Tenants  []Tenant               `json:"tenants"`
}

// GetTenants retrieves the list of accessible tenants for the authenticated client.
// This is typically used when a client has access to multiple tenants.
func (c *Client) GetTenants(ctx context.Context) (*TenantsResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/cloudprint", nil)
	if err != nil {
		return nil, fmt.Errorf("getting tenants: %w", err)
	}

	var tenantsResp TenantsResponse
	if err := parseResponse(resp, &tenantsResp); err != nil {
		return nil, fmt.Errorf("parsing tenants response: %w", err)
	}

	if !tenantsResp.Success {
		return nil, fmt.Errorf("get tenants failed: %s", tenantsResp.Message)
	}

	return &tenantsResp, nil
}

// SetTenant sets the active tenant for subsequent API calls.
// This is useful when the client has access to multiple tenants.
func (c *Client) SetTenant(tenantID string) {
	c.tenantID = tenantID
}