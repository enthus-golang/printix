package printix

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// PrintJob represents a print job submission.
type PrintJob struct {
	PrinterID     string         `json:"printerId"`
	Title         string         `json:"title"`
	ContentType   string         `json:"contentType,omitempty"`
	Source        string         `json:"source,omitempty"`
	JobProperties map[string]any `json:"jobProperties,omitempty"`
	TestMode      bool           `json:"-"` // Not sent to API
}

// SubmitResponse represents the response from submitting a print job.
type SubmitResponse struct {
	Response
	JobID       string `json:"jobId"`
	UploadLinks []struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Type    string            `json:"type"` // "Azure" or "GCP"
	} `json:"uploadLinks"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		UploadCompleted struct {
			Href string `json:"href"`
		} `json:"uploadCompleted"`
	} `json:"_links"`
}

// CompleteUploadRequest represents the request to complete an upload.
type CompleteUploadRequest struct {
	JobID string `json:"jobId"`
}

// PrintOptions represents print job options.
type PrintOptions struct {
	Copies      int    `json:"copies,omitempty"`
	Color       bool   `json:"color,omitempty"`
	Duplex      string `json:"duplex,omitempty"` // "none", "long-edge", "short-edge"
	PageRange   string `json:"pageRange,omitempty"`
	Orientation string `json:"orientation,omitempty"` // "portrait", "landscape"
}

// Submit creates a new print job.
func (c *Client) Submit(ctx context.Context, job *PrintJob) (*SubmitResponse, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for job submission")
	}

	endpoint := fmt.Sprintf(submitEndpoint, c.tenantID, job.PrinterID)
	if c.testMode || job.TestMode {
		endpoint += "?test=true"
	}

	// Build job properties
	jobReq := map[string]any{
		"title":  job.Title,
		"source": job.Source,
	}

	// Add optional properties
	if job.ContentType != "" {
		jobReq["contentType"] = job.ContentType
	}

	// Merge job properties
	for k, v := range job.JobProperties {
		jobReq[k] = v
	}

	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, jobReq)
	if err != nil {
		return nil, fmt.Errorf("submitting job: %w", err)
	}

	var submitResp SubmitResponse
	if err := parseResponse(resp, &submitResp); err != nil {
		return nil, fmt.Errorf("parsing submit response: %w", err)
	}

	if !submitResp.Success {
		return nil, fmt.Errorf("submit failed: %s (error ID: %s)", submitResp.ErrorDescription, submitResp.ErrorID)
	}

	return &submitResp, nil
}

// UploadDocument uploads a document to the cloud storage.
func (c *Client) UploadDocument(ctx context.Context, uploadLink string, headers map[string]string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadLink, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", "application/pdf")

	// Add any additional headers provided by Printix
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Use a separate HTTP client for cloud storage (no auth needed)
	storageClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := storageClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("upload failed with status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CompleteUpload notifies Printix that the document upload is complete.
func (c *Client) CompleteUpload(ctx context.Context, completeURL string) error {
	// CompleteUpload uses the HAL link provided in the submit response
	resp, err := c.doRequest(ctx, http.MethodPost, completeURL, nil)
	if err != nil {
		return fmt.Errorf("completing upload: %w", err)
	}

	var completeResp Response
	if err := parseResponse(resp, &completeResp); err != nil {
		return fmt.Errorf("parsing complete response: %w", err)
	}

	if !completeResp.Success {
		return fmt.Errorf("complete upload failed: %s (error ID: %s)", completeResp.ErrorDescription, completeResp.ErrorID)
	}

	return nil
}

// PrintFile prints a file using Printix.
func (c *Client) PrintFile(ctx context.Context, printerID, title, filePath string, options *PrintOptions) error {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Determine content type
	contentType := "application/pdf"
	if len(filePath) > 4 {
		switch filePath[len(filePath)-4:] {
		case ".zpl":
			contentType = "application/zpl"
		case ".pcl":
			contentType = "application/vnd.hp-PCL"
		case ".ps":
			contentType = "application/postscript"
		case ".xps":
			contentType = "application/vnd.ms-xpsdocument"
		case ".txt":
			contentType = "text/plain"
		}
	}

	// Create print job
	job := &PrintJob{
		PrinterID:   printerID,
		Title:       title,
		ContentType: contentType,
		Source:      "MTS API",
		TestMode:    c.testMode,
	}

	// Add options if provided
	if options != nil {
		job.JobProperties = make(map[string]any)
		if options.Copies > 0 {
			job.JobProperties["copies"] = options.Copies
		}
		if options.Duplex != "" {
			job.JobProperties["duplex"] = options.Duplex
		}
		if options.PageRange != "" {
			job.JobProperties["pageRange"] = options.PageRange
		}
		if options.Orientation != "" {
			job.JobProperties["orientation"] = options.Orientation
		}
		job.JobProperties["color"] = options.Color
	}

	// Submit the job
	submitResp, err := c.Submit(ctx, job)
	if err != nil {
		return fmt.Errorf("submitting print job: %w", err)
	}

	// Upload the document
	if len(submitResp.UploadLinks) == 0 {
		return fmt.Errorf("no upload links provided")
	}

	uploadLink := submitResp.UploadLinks[0]
	if err := c.UploadDocument(ctx, uploadLink.URL, uploadLink.Headers, data); err != nil {
		return fmt.Errorf("uploading document: %w", err)
	}

	// Complete the upload using the HAL link
	if err := c.CompleteUpload(ctx, submitResp.Links.UploadCompleted.Href); err != nil {
		return fmt.Errorf("completing upload: %w", err)
	}

	return nil
}

// PrintData prints raw data using Printix.
func (c *Client) PrintData(ctx context.Context, printerID, title string, data []byte, contentType string, options *PrintOptions) error {
	// Create print job
	job := &PrintJob{
		PrinterID:   printerID,
		Title:       title,
		ContentType: contentType,
		Source:      "MTS API",
		TestMode:    c.testMode,
	}

	// Add options if provided
	if options != nil {
		job.JobProperties = make(map[string]any)
		if options.Copies > 0 {
			job.JobProperties["copies"] = options.Copies
		}
		if options.Duplex != "" {
			job.JobProperties["duplex"] = options.Duplex
		}
		if options.PageRange != "" {
			job.JobProperties["pageRange"] = options.PageRange
		}
		if options.Orientation != "" {
			job.JobProperties["orientation"] = options.Orientation
		}
		job.JobProperties["color"] = options.Color
	}

	// Submit the job
	submitResp, err := c.Submit(ctx, job)
	if err != nil {
		return fmt.Errorf("submitting print job: %w", err)
	}

	// Upload the document
	if len(submitResp.UploadLinks) == 0 {
		return fmt.Errorf("no upload links provided")
	}

	uploadLink := submitResp.UploadLinks[0]
	if err := c.UploadDocument(ctx, uploadLink.URL, uploadLink.Headers, data); err != nil {
		return fmt.Errorf("uploading document: %w", err)
	}

	// Complete the upload using the HAL link
	if err := c.CompleteUpload(ctx, submitResp.Links.UploadCompleted.Href); err != nil {
		return fmt.Errorf("completing upload: %w", err)
	}

	return nil
}
