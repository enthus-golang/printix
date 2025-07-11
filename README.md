# Printix Go Client

A Go client library for the [Printix Cloud Print API](https://printix.github.io/).

## Installation

```bash
go get github.com/enthus-golang/printix
```

## Usage

### Authentication

The client uses OAuth 2.0 Client Credentials flow for authentication. You'll need to obtain a client ID and client secret from your Printix Administrator dashboard.

```go
import "github.com/enthus-golang/printix"

// Create a new client
client := printix.New(clientID, clientSecret)

// Use test environment
client := printix.New(clientID, clientSecret, printix.WithTestMode())

// Set tenant ID if known
client := printix.New(clientID, clientSecret, printix.WithTenantID("your-tenant-id"))
```

### Basic Print Job

```go
ctx := context.Background()

// Print a PDF file
err := client.PrintFile(ctx, printerID, "My Document", "/path/to/document.pdf", nil)
if err != nil {
    log.Fatal(err)
}

// Print with options (automatically uses v1.1 API)
options := &printix.PrintOptions{
    Copies:  2,
    Color:   true,
    Duplex:  "long-edge",
}
err = client.PrintFile(ctx, printerID, "My Document", "/path/to/document.pdf", options)
```

### Advanced Print Job Submission

For more control over the print job submission, you can use the Submit method directly:

```go
// v1.0 API (query parameters)
job := &printix.PrintJob{
    PrinterID: printerID,
    Title:     "My Document",
    User:      "john.doe",
    PDL:       "PCL5", // For non-PDF documents
}

// v1.1 API (JSON body with print settings)
copies := 2
job := &printix.PrintJob{
    PrinterID:       printerID,
    Title:          "My Document", 
    User:           "john.doe",
    UseV11:         true,
    Color:          &[]bool{true}[0],
    Duplex:         "LONG_EDGE", // NONE, SHORT_EDGE, LONG_EDGE
    PageOrientation: "PORTRAIT",  // PORTRAIT, LANDSCAPE, AUTO
    Copies:         &copies,
    MediaSize:      "A4",
    Scaling:        "FIT", // NOSCALE, SHRINK, FIT
}

submitResp, err := client.Submit(ctx, job)
if err != nil {
    log.Fatal(err)
}

// Upload document and complete
// ... (upload process)
```

### Managing Printers

```go
// Get all printers
printers, err := client.GetAllPrinters(ctx, "")
if err != nil {
    log.Fatal(err)
}

for _, printer := range printers {
    fmt.Printf("Printer: %s (ID: %s)\n", printer.Name, printer.ID)
}

// Find printer by name
printer, err := client.FindPrinterByName(ctx, "Office Printer")
if err != nil {
    log.Fatal(err)
}

// Get printer details
printer, err := client.GetPrinter(ctx, printerID)
if err != nil {
    log.Fatal(err)
}

// Check if printer supports a content type
if printer.SupportsContentType("application/pdf") {
    fmt.Println("Printer supports PDF")
}
```

### Managing Jobs

```go
// Get all jobs
jobs, err := client.GetJobs(ctx, nil)
if err != nil {
    log.Fatal(err)
}

// Get jobs with filters
opts := &printix.GetJobsOptions{
    PrinterID: "printer-123",
    Status:    printix.JobStatusPending,
    Limit:     10,
}
jobs, err := client.GetJobs(ctx, opts)

// Get specific job
job, err := client.GetJob(ctx, jobID)
if err != nil {
    log.Fatal(err)
}

// Cancel a job
err = client.CancelJob(ctx, jobID)

// Delete a job
err = client.DeleteJob(ctx, jobID)
```

### Managing Users

```go
// Get all users
usersResp, err := client.GetUsers(ctx, nil)
if err != nil {
    log.Fatal(err)
}

// Find user by email
opts := &printix.GetUsersOptions{
    Email: "user@example.com",
}
usersResp, err := client.GetUsers(ctx, opts)

// Create a new user (regular user)
user := &printix.User{
    Email:       "newuser@example.com",
    Name:        "New User",
    DisplayName: "New User",
    Active:      true,
}
createdUser, err := client.CreateUser(ctx, user)

// Create a guest user
guestUser := &printix.User{
    Email:    "guest@example.com",
    FullName: "Guest User",
    Role:     "GUEST_USER",
    PIN:      "1234",     // Optional 4-digit PIN
    Password: "password", // Optional password
}
createdGuest, err := client.CreateUser(ctx, guestUser)

// Update user
user.DisplayName = "Updated Name"
updatedUser, err := client.UpdateUser(ctx, user.ID, user)

// Delete user
err = client.DeleteUser(ctx, userID)
```

### Managing Groups

```go
// Get all groups
groupsResp, err := client.GetGroups(ctx, nil)
if err != nil {
    log.Fatal(err)
}

// Create a new group
group := &printix.Group{
    Name:        "Engineering",
    Description: "Engineering team",
}
createdGroup, err := client.CreateGroup(ctx, group)

// Add user to group
err = client.AddGroupMember(ctx, groupID, userID)

// Remove user from group
err = client.RemoveGroupMember(ctx, groupID, userID)

// Delete group
err = client.DeleteGroup(ctx, groupID)
```

### Webhook Validation

```go
// Create a webhook validator
validator := printix.NewWebhookValidator(sharedSecret)

// In your webhook handler
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    // Validate the request
    if err := validator.ValidateRequest(r); err != nil {
        http.Error(w, "Invalid webhook", http.StatusUnauthorized)
        return
    }

    // Parse the webhook payload
    payload, err := printix.ParseWebhookPayload(r)
    if err != nil {
        http.Error(w, "Invalid payload", http.StatusBadRequest)
        return
    }

    // Process events
    for _, event := range payload.Events {
        if event.IsUserCreateEvent() {
            fmt.Printf("User created: %s\n", event.Href)
        }
        
        if event.IsJobStatusChangeEvent() {
            fmt.Printf("Job status change: %s\n", event.Href)
        }
        
        // Get event timestamp
        timestamp := event.GetTimestamp()
        fmt.Printf("Event time: %s\n", timestamp)
    }

    w.WriteHeader(http.StatusOK)
}
```

### Advanced Usage

#### Custom HTTP Client

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
}
client := printix.New(clientID, clientSecret, printix.WithHTTPClient(httpClient))
```

#### Rate Limiting

The API has a rate limit of 100 requests per minute per user. The client exposes rate limit information:

```go
remaining, reset := client.GetRateLimitInfo()
fmt.Printf("Remaining requests: %d, Reset at: %s\n", remaining, reset)
```

#### Multiple Tenants

If your client has access to multiple tenants:

```go
// Get available tenants
tenantsResp, err := client.GetTenants(ctx)
if err != nil {
    log.Fatal(err)
}

// Set active tenant
client.SetTenant(tenantsResp.Tenants[0].ID)
```

## Supported File Types

- **PDF** (application/pdf) - Default, no PDL parameter needed
- **PCL** (PCL5) - For PCL files, detected automatically by .pcl extension
- **PostScript** (POSTSCRIPT) - For .ps files
- **XPS** (XPS) - For .xps files
- **ZPL** (ZPL) - For .zpl label printer files
- **Plain Text** (text/plain) - For .txt files

The client automatically detects file types and sets the appropriate PDL parameter for non-PDF files.

## Error Handling

All API errors include a success flag and error description:

```go
jobs, err := client.GetJobs(ctx, nil)
if err != nil {
    // API errors will include description and error ID
    fmt.Printf("Error: %v\n", err)
}
```

## Testing

The client supports test mode which uses the test environment:

```go
client := printix.New(clientID, clientSecret, printix.WithTestMode())
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.