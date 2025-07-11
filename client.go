package printix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL         = "https://api.printix.net"
	defaultAuthURL         = "https://auth.printix.net/oauth/token"
	testAuthURL            = "https://auth.testenv.printix.net/oauth/token"
	submitEndpoint         = "/cloudprint/tenants/%s/printers/%s/jobs"
	completeUploadEndpoint = "/cloudprint/completeUpload"
	printersEndpoint       = "/cloudprint/tenants/%s/printers"
	jobsEndpoint           = "/cloudprint/tenants/%s/jobs"
	tokenExpirySeconds     = 3599 // 1 hour
	tokenRenewalBuffer     = 600  // Renew 10 minutes before expiry
)

// Client represents a Printix API client.
type Client struct {
	httpClient      *http.Client
	baseURL         string
	authURL         string
	clientID        string
	clientSecret    string
	tenantID        string
	accessToken     string
	tokenExpiry     time.Time
	testMode        bool
	rateLimitRemain int
	rateLimitReset  time.Time
}

// Option is a function that configures the client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithBaseURL sets a custom base URL for the API.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithTestMode enables test mode for the client.
func WithTestMode() Option {
	return func(c *Client) {
		c.testMode = true
		c.authURL = testAuthURL
	}
}

// WithTenantID sets the tenant ID for the client.
func WithTenantID(tenantID string) Option {
	return func(c *Client) {
		c.tenantID = tenantID
	}
}

// WithAuthURL sets a custom auth URL for the client.
func WithAuthURL(authURL string) Option {
	return func(c *Client) {
		c.authURL = authURL
	}
}

// New creates a new Printix client.
func New(clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      defaultBaseURL,
		authURL:      defaultAuthURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// authenticate gets or refreshes the OAuth access token.
func (c *Client) authenticate(ctx context.Context) error {
	// Check if token is still valid with renewal buffer
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-tokenRenewalBuffer*time.Second)) {
		return nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("authentication failed with status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}

	c.accessToken = authResp.AccessToken
	// Use the exact expiry time from response
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	return nil
}

// doRequest performs an authenticated HTTP request.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	// For absolute URLs (like HAL links), use them directly
	fullURL := endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		fullURL = c.baseURL + endpoint
	}

	if err := c.authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	// Extract rate limit headers
	if remaining := resp.Header.Get("X-Rate-Limit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimitRemain = val
		}
	}
	if reset := resp.Header.Get("X-Rate-Limit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimitReset = time.Unix(val, 0)
		}
	}

	return resp, nil
}

// Response represents a generic API response.
type Response struct {
	Success          bool   `json:"success"`
	ErrorDescription string `json:"errorDescription,omitempty"`
	ErrorID          string `json:"errorId,omitempty"`
}

// parseResponse reads and parses the API response.
func parseResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("request failed with status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// GetRateLimitInfo returns the current rate limit status.
func (c *Client) GetRateLimitInfo() (remaining int, reset time.Time) {
	return c.rateLimitRemain, c.rateLimitReset
}

// GetTenantID returns the tenant ID.
func (c *Client) GetTenantID() string {
	return c.tenantID
}
