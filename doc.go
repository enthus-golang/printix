// Package printix provides a client for the Printix Cloud Print API.
//
// The Printix API allows applications to submit print jobs to cloud-connected printers.
// It supports various document formats including PDF, PCL, PostScript, and plain text.
//
// Basic usage:
//
//	client := printix.New(clientID, clientSecret, printix.WithTestMode(true))
//
//	// Print a PDF file
//	err := client.PrintFile(ctx, printerID, "My Document", "/path/to/document.pdf", nil)
//
//	// Get available printers
//	printers, err := client.GetPrinters(ctx)
//
// The package handles OAuth authentication automatically and provides methods for:
//   - Submitting print jobs
//   - Uploading documents
//   - Managing printers
//   - Tracking job status
//
// For ZPL label printing, the content type should be set appropriately,
// though full ZPL support depends on the printer capabilities.
package printix
