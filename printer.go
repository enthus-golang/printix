package printix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Printer represents a Printix printer.
type Printer struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	ConnectionStatus string                 `json:"connectionStatus,omitempty"`
	PrinterSignID    string                 `json:"printerSignId,omitempty"`
	Location         string                 `json:"location,omitempty"`
	Model            string                 `json:"model,omitempty"`
	Vendor           string                 `json:"vendor,omitempty"`
	SerialNo         string                 `json:"serialNo,omitempty"`
	Capabilities     PrinterCapabilities    `json:"capabilities,omitempty"`
	Links            map[string]interface{} `json:"_links,omitempty"`
}

// PrinterCapabilities represents printer capabilities.
type PrinterCapabilities struct {
	Printer struct {
		MediaSize struct {
			Option []MediaSizeOption `json:"option,omitempty"`
		} `json:"media_size,omitempty"`
		SupportedContentType []ContentType `json:"supported_content_type,omitempty"`
		Copies               struct {
			Default int `json:"default,omitempty"`
			Max     int `json:"max,omitempty"`
		} `json:"copies,omitempty"`
		Color struct {
			Option []ColorOption `json:"option,omitempty"`
		} `json:"color,omitempty"`
		VendorCapability []VendorCapability `json:"vendor_capability,omitempty"`
	} `json:"printer,omitempty"`
}

// MediaSizeOption represents a media size option.
type MediaSizeOption struct {
	HeightMicrons    int    `json:"heightMicrons"`
	WidthMicrons     int    `json:"widthMicrons"`
	Name             string `json:"name"`
	IsContinuousFeed bool   `json:"isContinuousFeed"`
	IsDefault        bool   `json:"isDefault"`
}

// ContentType represents a supported content type.
type ContentType struct {
	ContentType string `json:"content_type"`
	MinVersion  string `json:"min_version,omitempty"`
}

// ColorOption represents a color option.
type ColorOption struct {
	Type    string `json:"type"`
	Default bool   `json:"default"`
}

// VendorCapability represents a vendor-specific capability.
type VendorCapability struct {
	ID                   string                 `json:"id"`
	DisplayName          string                 `json:"display_name"`
	Type                 string                 `json:"type"`
	DisplayNameLocalized []LocalizedString      `json:"display_name_localized,omitempty"`
	TypedValueCap        map[string]interface{} `json:"typed_value_cap,omitempty"`
}

// LocalizedString represents a localized string.
type LocalizedString struct {
	Locale string `json:"locale"`
	Value  string `json:"value"`
}

// PrintersResponse represents the HAL+JSON response from listing printers.
type PrintersResponse struct {
	Links    map[string]interface{} `json:"_links"`
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Printers []Printer              `json:"printers"`
	Page     struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// GetPrintersOptions represents options for listing printers.
type GetPrintersOptions struct {
	Query    string // Search query for printer names
	Page     int    // Page number (0-based)
	PageSize int    // Number of printers per page
}

// GetPrinters retrieves the list of available printers with pagination.
func (c *Client) GetPrinters(ctx context.Context, opts *GetPrintersOptions) (*PrintersResponse, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting printers")
	}

	endpoint := fmt.Sprintf(printersEndpoint, c.tenantID)

	// Add query parameters if options are provided
	if opts != nil {
		params := make([]string, 0, 3)
		if opts.Query != "" {
			params = append(params, fmt.Sprintf("query=%s", url.QueryEscape(opts.Query)))
		}
		if opts.Page > 0 {
			params = append(params, fmt.Sprintf("page=%d", opts.Page))
		}
		if opts.PageSize > 0 {
			params = append(params, fmt.Sprintf("pageSize=%d", opts.PageSize))
		}
		if len(params) > 0 {
			endpoint += "?" + strings.Join(params, "&")
		}
	}

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting printers: %w", err)
	}

	var printersResp PrintersResponse
	if err := parseResponse(resp, &printersResp); err != nil {
		return nil, fmt.Errorf("parsing printers response: %w", err)
	}

	if !printersResp.Success {
		return nil, fmt.Errorf("get printers failed: %s", printersResp.Message)
	}

	return &printersResp, nil
}

// GetAllPrinters retrieves all available printers by automatically handling pagination.
func (c *Client) GetAllPrinters(ctx context.Context, query string) ([]Printer, error) {
	var allPrinters []Printer
	page := 0
	pageSize := 100 // Use a larger page size for efficiency

	for {
		opts := &GetPrintersOptions{
			Query:    query,
			Page:     page,
			PageSize: pageSize,
		}

		resp, err := c.GetPrinters(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("getting printers page %d: %w", page, err)
		}

		allPrinters = append(allPrinters, resp.Printers...)

		// Check if we've reached the last page
		if page >= resp.Page.TotalPages-1 || len(resp.Printers) == 0 {
			break
		}

		page++
	}

	return allPrinters, nil
}

// GetPrinter retrieves details for a specific printer.
func (c *Client) GetPrinter(ctx context.Context, printerID string) (*Printer, error) {
	if c.tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for getting printer")
	}

	endpoint := fmt.Sprintf("%s/%s", fmt.Sprintf(printersEndpoint, c.tenantID), printerID)
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting printer: %w", err)
	}

	var printerResp struct {
		Links   map[string]interface{} `json:"_links"`
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Printer
	}

	if err := parseResponse(resp, &printerResp); err != nil {
		return nil, fmt.Errorf("parsing printer response: %w", err)
	}

	if !printerResp.Success {
		return nil, fmt.Errorf("get printer failed: %s", printerResp.Message)
	}

	// Create a printer instance from the embedded fields
	printer := Printer{
		ID:               printerResp.ID,
		Name:             printerResp.Name,
		ConnectionStatus: printerResp.ConnectionStatus,
		PrinterSignID:    printerResp.PrinterSignID,
		Location:         printerResp.Location,
		Model:            printerResp.Model,
		Vendor:           printerResp.Vendor,
		SerialNo:         printerResp.SerialNo,
		Capabilities:     printerResp.Capabilities,
		Links:            printerResp.Links,
	}

	return &printer, nil
}

// FindPrinterByName finds a printer by its name.
func (c *Client) FindPrinterByName(ctx context.Context, name string) (*Printer, error) {
	// Use the query parameter to search for the printer by name
	printers, err := c.GetAllPrinters(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting printers: %w", err)
	}

	// Look for exact match
	for i := range printers {
		printer := &printers[i]
		if printer.Name == name {
			return printer, nil
		}
	}

	return nil, fmt.Errorf("printer with name %s not found", name)
}

// SupportsContentType checks if a printer supports a specific content type.
func (p *Printer) SupportsContentType(contentType string) bool {
	for _, ct := range p.Capabilities.Printer.SupportedContentType {
		if ct.ContentType == contentType {
			return true
		}
	}
	return false
}
