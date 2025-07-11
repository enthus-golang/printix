package printix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Job represents a print job.
type Job struct {
	ID          string         `json:"id"`
	PrinterID   string         `json:"printerId"`
	PrinterName string         `json:"printerName,omitempty"`
	Title       string         `json:"title"`
	Status      string         `json:"status"`
	Source      string         `json:"source,omitempty"`
	CreatedAt   string         `json:"createdAt,omitempty"`
	UpdatedAt   string         `json:"updatedAt,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	UserName    string         `json:"userName,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// JobsResponse represents the response from listing jobs.
type JobsResponse struct {
	Response
	Jobs []Job `json:"jobs"`
}

// JobStatus represents possible job statuses.
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusPrinting   = "printing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusCancelled  = "cancelled"
)

// GetJobsOptions represents options for retrieving jobs.
type GetJobsOptions struct {
	PrinterID string
	UserID    string
	Status    string
	Limit     int
	Offset    int
}

// GetJobs retrieves print jobs based on the provided options.
func (c *Client) GetJobs(ctx context.Context, opts *GetJobsOptions) ([]Job, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting jobs")
	}

	endpoint := fmt.Sprintf(jobsEndpoint, c.tenantID)

	if opts != nil {
		params := url.Values{}
		if opts.PrinterID != "" {
			params.Set("printerId", opts.PrinterID)
		}
		if opts.UserID != "" {
			params.Set("userId", opts.UserID)
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", strconv.Itoa(opts.Offset))
		}

		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting jobs: %w", err)
	}

	var jobsResp JobsResponse
	if err := parseResponse(resp, &jobsResp); err != nil {
		return nil, fmt.Errorf("parsing jobs response: %w", err)
	}

	if !jobsResp.Success {
		return nil, fmt.Errorf("get jobs failed: %s (error ID: %s)", jobsResp.ErrorDescription, jobsResp.ErrorID)
	}

	return jobsResp.Jobs, nil
}

// GetJob retrieves details for a specific job.
func (c *Client) GetJob(ctx context.Context, jobID string) (*Job, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting job")
	}

	endpoint := fmt.Sprintf("%s/%s", fmt.Sprintf(jobsEndpoint, c.tenantID), jobID)

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting job: %w", err)
	}

	var jobResp struct {
		Response
		Job Job `json:"job"`
	}

	if err := parseResponse(resp, &jobResp); err != nil {
		return nil, fmt.Errorf("parsing job response: %w", err)
	}

	if !jobResp.Success {
		return nil, fmt.Errorf("get job failed: %s (error ID: %s)", jobResp.ErrorDescription, jobResp.ErrorID)
	}

	return &jobResp.Job, nil
}

// CancelJob cancels a print job.
func (c *Client) CancelJob(ctx context.Context, jobID string) error {
	if c.tenantID == "" {
		return fmt.Errorf("tenant ID is required for cancelling job")
	}

	endpoint := fmt.Sprintf("%s/%s/cancel", fmt.Sprintf(jobsEndpoint, c.tenantID), jobID)

	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}

	var cancelResp Response
	if err := parseResponse(resp, &cancelResp); err != nil {
		return fmt.Errorf("parsing cancel response: %w", err)
	}

	if !cancelResp.Success {
		return fmt.Errorf("cancel job failed: %s (error ID: %s)", cancelResp.ErrorDescription, cancelResp.ErrorID)
	}

	return nil
}

// DeleteJob deletes a print job.
func (c *Client) DeleteJob(ctx context.Context, jobID string) error {
	if c.tenantID == "" {
		return fmt.Errorf("tenant ID is required for deleting job")
	}

	endpoint := fmt.Sprintf("%s/%s", fmt.Sprintf(jobsEndpoint, c.tenantID), jobID)

	resp, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("deleting job: %w", err)
	}

	var deleteResp Response
	if err := parseResponse(resp, &deleteResp); err != nil {
		return fmt.Errorf("parsing delete response: %w", err)
	}

	if !deleteResp.Success {
		return fmt.Errorf("delete job failed: %s (error ID: %s)", deleteResp.ErrorDescription, deleteResp.ErrorID)
	}

	return nil
}
