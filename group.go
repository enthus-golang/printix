package printix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Group represents a Printix group.
type Group struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Members     []string       `json:"members,omitempty"`
	Created     string         `json:"created,omitempty"`
	Updated     string         `json:"updated,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// GroupsResponse represents the response from listing groups.
type GroupsResponse struct {
	Response
	Groups []Group `json:"groups"`
	Page   struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// GetGroupsOptions represents options for retrieving groups.
type GetGroupsOptions struct {
	Name     string
	UserID   string
	Page     int
	PageSize int
}

// GetGroups retrieves groups based on the provided options.
func (c *Client) GetGroups(ctx context.Context, opts *GetGroupsOptions) (*GroupsResponse, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting groups")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups", c.tenantID)

	if opts != nil {
		params := url.Values{}
		if opts.Name != "" {
			params.Set("name", opts.Name)
		}
		if opts.UserID != "" {
			params.Set("userId", opts.UserID)
		}
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PageSize > 0 {
			params.Set("pageSize", strconv.Itoa(opts.PageSize))
		}

		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting groups: %w", err)
	}

	var groupsResp GroupsResponse
	if err := parseResponse(resp, &groupsResp); err != nil {
		return nil, fmt.Errorf("parsing groups response: %w", err)
	}

	if !groupsResp.Success {
		return nil, fmt.Errorf("get groups failed: %s (error ID: %s)", groupsResp.ErrorDescription, groupsResp.ErrorID)
	}

	return &groupsResp, nil
}

// GetGroup retrieves details for a specific group.
func (c *Client) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting group")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups/%s", c.tenantID, groupID)

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}

	var groupResp struct {
		Response
		Group Group `json:"group"`
	}

	if err := parseResponse(resp, &groupResp); err != nil {
		return nil, fmt.Errorf("parsing group response: %w", err)
	}

	if !groupResp.Success {
		return nil, fmt.Errorf("get group failed: %s (error ID: %s)", groupResp.ErrorDescription, groupResp.ErrorID)
	}

	return &groupResp.Group, nil
}

// CreateGroup creates a new group.
func (c *Client) CreateGroup(ctx context.Context, group *Group) (*Group, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for creating group")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups", c.tenantID)

	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, group)
	if err != nil {
		return nil, fmt.Errorf("creating group: %w", err)
	}

	var groupResp struct {
		Response
		Group Group `json:"group"`
	}

	if err := parseResponse(resp, &groupResp); err != nil {
		return nil, fmt.Errorf("parsing group response: %w", err)
	}

	if !groupResp.Success {
		return nil, fmt.Errorf("create group failed: %s (error ID: %s)", groupResp.ErrorDescription, groupResp.ErrorID)
	}

	return &groupResp.Group, nil
}

// UpdateGroup updates an existing group.
func (c *Client) UpdateGroup(ctx context.Context, groupID string, group *Group) (*Group, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for updating group")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups/%s", c.tenantID, groupID)

	resp, err := c.doRequest(ctx, http.MethodPut, endpoint, group)
	if err != nil {
		return nil, fmt.Errorf("updating group: %w", err)
	}

	var groupResp struct {
		Response
		Group Group `json:"group"`
	}

	if err := parseResponse(resp, &groupResp); err != nil {
		return nil, fmt.Errorf("parsing group response: %w", err)
	}

	if !groupResp.Success {
		return nil, fmt.Errorf("update group failed: %s (error ID: %s)", groupResp.ErrorDescription, groupResp.ErrorID)
	}

	return &groupResp.Group, nil
}

// DeleteGroup deletes a group.
func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
	if c.tenantID == "" {
		return fmt.Errorf("tenant ID is required for deleting group")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups/%s", c.tenantID, groupID)

	resp, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("deleting group: %w", err)
	}

	var deleteResp Response
	if err := parseResponse(resp, &deleteResp); err != nil {
		return fmt.Errorf("parsing delete response: %w", err)
	}

	if !deleteResp.Success {
		return fmt.Errorf("delete group failed: %s (error ID: %s)", deleteResp.ErrorDescription, deleteResp.ErrorID)
	}

	return nil
}

// AddGroupMember adds a user to a group.
func (c *Client) AddGroupMember(ctx context.Context, groupID, userID string) error {
	if c.tenantID == "" {
		return fmt.Errorf("tenant ID is required for adding group member")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups/%s/members/%s", c.tenantID, groupID, userID)

	resp, err := c.doRequest(ctx, http.MethodPut, endpoint, nil)
	if err != nil {
		return fmt.Errorf("adding group member: %w", err)
	}

	var addResp Response
	if err := parseResponse(resp, &addResp); err != nil {
		return fmt.Errorf("parsing add member response: %w", err)
	}

	if !addResp.Success {
		return fmt.Errorf("add group member failed: %s (error ID: %s)", addResp.ErrorDescription, addResp.ErrorID)
	}

	return nil
}

// RemoveGroupMember removes a user from a group.
func (c *Client) RemoveGroupMember(ctx context.Context, groupID, userID string) error {
	if c.tenantID == "" {
		return fmt.Errorf("tenant ID is required for removing group member")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/groups/%s/members/%s", c.tenantID, groupID, userID)

	resp, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("removing group member: %w", err)
	}

	var removeResp Response
	if err := parseResponse(resp, &removeResp); err != nil {
		return fmt.Errorf("parsing remove member response: %w", err)
	}

	if !removeResp.Success {
		return fmt.Errorf("remove group member failed: %s (error ID: %s)", removeResp.ErrorDescription, removeResp.ErrorID)
	}

	return nil
}