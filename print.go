package printix

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// UserMapping represents user mapping for print job assignment.
type UserMapping struct {
	Key   string `json:"key"`   // AzureObjectId, AzureUPN, SAMAccountName, OnPremImmutableId, OnPremUpn, Email
	Value string `json:"value"` // Value to filter for
}

// PrintJob represents a print job submission.
type PrintJob struct {
	PrinterID     string         `json:"-"` // Not sent in body, used in URL
	QueueID       string         `json:"-"` // Not sent in body, used in URL
	Title         string         `json:"-"` // Not sent in body, used in URL query
	User          string         `json:"-"` // Not sent in body, used in URL query
	PDL           string         `json:"-"` // Not sent in body, used in URL query
	// v1.1 properties (sent in body)
	Color           *bool        `json:"color,omitempty"`
	Duplex          string       `json:"duplex,omitempty"`            // NONE, SHORT_EDGE, LONG_EDGE
	PageOrientation string       `json:"page_orientation,omitempty"`  // PORTRAIT, LANDSCAPE, AUTO
	Copies          *int         `json:"copies,omitempty"`
	MediaSize       string       `json:"media_size,omitempty"`
	Scaling         string       `json:"scaling,omitempty"`           // NOSCALE, SHRINK, FIT
	UserMapping     *UserMapping `json:"userMapping,omitempty"`
	// Control fields
	ReleaseImmediately *bool `json:"-"`  // Not sent in body, used in URL query
	TestMode           bool  `json:"-"`  // Not sent in body, used in URL query
	UseV11             bool  `json:"-"`  // Use v1.1 API
}

// SubmitResponse represents the response from submitting a print job.
type SubmitResponse struct {
	Response
	Job struct {
		ID          string `json:"id"`
		CreateTime  string `json:"createTime"`  // ISO format timestamp
		UpdateTime  string `json:"updateTime"`  // ISO format timestamp
		Status      string `json:"status"`
		OwnerID     string `json:"ownerId"`
		ContentType string `json:"contentType"`
		Title       string `json:"title"`
		Links       struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			Printer struct {
				Href string `json:"href"`
			} `json:"printer"`
			ChangeOwner struct {
				Href      string `json:"href"`
				Templated bool   `json:"templated"`
			} `json:"changeOwner"`
		} `json:"_links"`
	} `json:"job"`
	UploadLinks []struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Type    string            `json:"type,omitempty"` // "Azure" or "GCP" - not always present
	} `json:"uploadLinks"`
	Links struct {
		UploadCompleted struct {
			Href string `json:"href"`
		} `json:"uploadCompleted"`
		ChangeOwner struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"changeOwner"`
	} `json:"_links"`
}

// CompleteUploadRequest represents the request to complete an upload.
type CompleteUploadRequest struct {
	JobID string `json:"jobId"`
}

// PrintOptions represents print job options.
type PrintOptions struct {
	Copies          int    `json:"copies,omitempty"`      // Number of copies (positive integer)
	Color           bool   `json:"color,omitempty"`       // true for color, false for monochrome
	Duplex          string `json:"duplex,omitempty"`      // "none", "long-edge", "short-edge"
	Orientation     string `json:"orientation,omitempty"` // "portrait", "landscape"
	MediaSize       string `json:"mediaSize,omitempty"`   // Paper size: A0-A5, B4-B5, LETTER, LEGAL, etc.
	Scaling         string `json:"scaling,omitempty"`     // "NOSCALE", "SHRINK", "FIT"
	PageRange       string `json:"pageRange,omitempty"`   // Page range (not used in v1.1 API)
}

// Submit creates a new print job.
func (c *Client) Submit(ctx context.Context, job *PrintJob) (*SubmitResponse, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for job submission")
	}

	if job.QueueID == "" {
		return nil, fmt.Errorf("queue ID is required for job submission")
	}
	endpoint := fmt.Sprintf(submitEndpoint, c.tenantID, job.PrinterID, job.QueueID)
	
	// Add query parameters
	params := url.Values{}
	if job.Title != "" {
		params.Set("title", job.Title)
	}
	if job.User != "" {
		params.Set("user", job.User)
	}
	if job.PDL != "" {
		params.Set("PDL", job.PDL)
	}
	if c.testMode || job.TestMode {
		params.Set("test", "true")
	}
	// Handle releaseImmediately parameter (default is true)
	if job.ReleaseImmediately != nil {
		params.Set("releaseImmediately", strconv.FormatBool(*job.ReleaseImmediately))
	} else {
		params.Set("releaseImmediately", "true")
	}
	
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var requestBody any
	headers := make(map[string]string)
	
	// Use v1.1 if specified or if any v1.1 properties are set
	if job.UseV11 || job.Color != nil || job.Duplex != "" || job.PageOrientation != "" || 
	   job.Copies != nil || job.MediaSize != "" || job.Scaling != "" {
		headers["version"] = "1.1"
		
		// Build v1.1 request body
		v11Body := make(map[string]any)
		if job.Color != nil {
			v11Body["color"] = *job.Color
		}
		if job.Duplex != "" {
			v11Body["duplex"] = job.Duplex
		}
		if job.PageOrientation != "" {
			v11Body["page_orientation"] = job.PageOrientation
		}
		if job.Copies != nil {
			v11Body["copies"] = *job.Copies
		}
		if job.MediaSize != "" {
			v11Body["media_size"] = job.MediaSize
		}
		if job.Scaling != "" {
			v11Body["scaling"] = job.Scaling
		}
		v11Body["userMapping"] = job.UserMapping
		
		// Always send body for v1.1, even if empty
		requestBody = v11Body
	}

	resp, err := c.doRequestWithHeaders(ctx, http.MethodPost, endpoint, requestBody, headers)
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

	// Use the configured HTTP client for cloud storage uploads
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading document: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

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
func (c *Client) PrintFile(ctx context.Context, printerID, queueID, title, filePath string, options *PrintOptions) error {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Determine PDL based on file extension
	var pdl string
	if len(filePath) > 4 {
		switch filePath[len(filePath)-4:] {
		case ".zpl":
			pdl = "ZPL"
		case ".pcl":
			pdl = "PCL5"
		case ".ps":
			pdl = "POSTSCRIPT"
		case ".xps":
			pdl = "XPS"
		}
	}

	// Create print job
	job := &PrintJob{
		PrinterID: printerID,
		QueueID:   queueID,
		Title:     title,
		User:      c.userIdentifier,
		PDL:       pdl,
		TestMode:  c.testMode,
	}

	// Add options if provided  
	if options != nil {
		job.UseV11 = true
		if options.Copies > 0 {
			job.Copies = &options.Copies
		}
		if options.Color {
			job.Color = &options.Color
		}
		// Map old duplex values to new format
		switch options.Duplex {
		case "none":
			job.Duplex = "NONE"
		case "long-edge":
			job.Duplex = "LONG_EDGE"
		case "short-edge":
			job.Duplex = "SHORT_EDGE"
		}
		// Map old orientation to new format
		switch options.Orientation {
		case "portrait":
			job.PageOrientation = "PORTRAIT"
		case "landscape":
			job.PageOrientation = "LANDSCAPE"
		}
		// Add new v1.1 options
		if options.MediaSize != "" {
			job.MediaSize = options.MediaSize
		}
		if options.Scaling != "" {
			job.Scaling = options.Scaling
		}
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
func (c *Client) PrintData(ctx context.Context, printerID, queueID, title string, data []byte, pdl string, options *PrintOptions) error {
	// Create print job
	job := &PrintJob{
		PrinterID: printerID,
		QueueID:   queueID,
		Title:     title,
		User:      c.userIdentifier,
		PDL:       pdl,
		TestMode:  c.testMode,
	}

	// Add options if provided  
	if options != nil {
		job.UseV11 = true
		if options.Copies > 0 {
			job.Copies = &options.Copies
		}
		if options.Color {
			job.Color = &options.Color
		}
		// Map old duplex values to new format
		switch options.Duplex {
		case "none":
			job.Duplex = "NONE"
		case "long-edge":
			job.Duplex = "LONG_EDGE"
		case "short-edge":
			job.Duplex = "SHORT_EDGE"
		}
		// Map old orientation to new format
		switch options.Orientation {
		case "portrait":
			job.PageOrientation = "PORTRAIT"
		case "landscape":
			job.PageOrientation = "LANDSCAPE"
		}
		// Add new v1.1 options
		if options.MediaSize != "" {
			job.MediaSize = options.MediaSize
		}
		if options.Scaling != "" {
			job.Scaling = options.Scaling
		}
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
