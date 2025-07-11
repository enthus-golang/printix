package printix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// User represents a Printix user.
type User struct {
	ID          string         `json:"id"`
	Email       string         `json:"email"`
	Name        string         `json:"name,omitempty"`
	FullName    string         `json:"fullName,omitempty"`  // For guest users
	UserName    string         `json:"userName,omitempty"`
	DisplayName string         `json:"displayName,omitempty"`
	Role        string         `json:"role,omitempty"`      // e.g., "GUEST_USER"
	PIN         string         `json:"pin,omitempty"`       // 4-digit PIN for guest users
	Password    string         `json:"password,omitempty"`  // Password for guest users
	Active      bool           `json:"active"`
	Created     string         `json:"created,omitempty"`
	Updated     string         `json:"updated,omitempty"`
	Groups      []string       `json:"groups,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// UsersResponse represents the response from listing users.
type UsersResponse struct {
	Response
	Users []User `json:"users"`
	Page  struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// GetUsersOptions represents options for retrieving users.
type GetUsersOptions struct {
	Email    string
	UserName string
	Active   *bool
	GroupID  string
	Page     int
	PageSize int
}

// GetUsers retrieves users based on the provided options.
func (c *Client) GetUsers(ctx context.Context, opts *GetUsersOptions) (*UsersResponse, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting users")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/users", c.tenantID)

	if opts != nil {
		params := url.Values{}
		if opts.Email != "" {
			params.Set("email", opts.Email)
		}
		if opts.UserName != "" {
			params.Set("userName", opts.UserName)
		}
		if opts.Active != nil {
			params.Set("active", strconv.FormatBool(*opts.Active))
		}
		if opts.GroupID != "" {
			params.Set("groupId", opts.GroupID)
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
		return nil, fmt.Errorf("getting users: %w", err)
	}

	var usersResp UsersResponse
	if err := parseResponse(resp, &usersResp); err != nil {
		return nil, fmt.Errorf("parsing users response: %w", err)
	}

	if !usersResp.Success {
		return nil, fmt.Errorf("get users failed: %s (error ID: %s)", usersResp.ErrorDescription, usersResp.ErrorID)
	}

	return &usersResp, nil
}

// GetUser retrieves details for a specific user.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting user")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/users/%s", c.tenantID, userID)

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	var userResp struct {
		Response
		User User `json:"user"`
	}

	if err := parseResponse(resp, &userResp); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}

	if !userResp.Success {
		return nil, fmt.Errorf("get user failed: %s (error ID: %s)", userResp.ErrorDescription, userResp.ErrorID)
	}

	return &userResp.User, nil
}

// CreateUser creates a new user.
func (c *Client) CreateUser(ctx context.Context, user *User) (*User, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for creating user")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/users", c.tenantID)

	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, user)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	var userResp struct {
		Response
		User User `json:"user"`
	}

	if err := parseResponse(resp, &userResp); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}

	if !userResp.Success {
		return nil, fmt.Errorf("create user failed: %s (error ID: %s)", userResp.ErrorDescription, userResp.ErrorID)
	}

	return &userResp.User, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, userID string, user *User) (*User, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for updating user")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/users/%s", c.tenantID, userID)

	resp, err := c.doRequest(ctx, http.MethodPut, endpoint, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	var userResp struct {
		Response
		User User `json:"user"`
	}

	if err := parseResponse(resp, &userResp); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}

	if !userResp.Success {
		return nil, fmt.Errorf("update user failed: %s (error ID: %s)", userResp.ErrorDescription, userResp.ErrorID)
	}

	return &userResp.User, nil
}

// DeleteUser deletes a user.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	if c.tenantID == "" {
		return fmt.Errorf("tenant ID is required for deleting user")
	}

	endpoint := fmt.Sprintf("/cloudprint/tenants/%s/users/%s", c.tenantID, userID)

	resp, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	var deleteResp Response
	if err := parseResponse(resp, &deleteResp); err != nil {
		return fmt.Errorf("parsing delete response: %w", err)
	}

	if !deleteResp.Success {
		return fmt.Errorf("delete user failed: %s (error ID: %s)", deleteResp.ErrorDescription, deleteResp.ErrorID)
	}

	return nil
}