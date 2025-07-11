# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go client library for the Printix Cloud Print API. It provides a comprehensive interface to interact with Printix services including print job submission, printer management, user/group management, and webhook handling.

## Common Development Commands

### Build and Test
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a specific test
go test -run TestClient_Submit

# Run tests with verbose output
go test -v ./...

# Build the module
go build ./...

# Update dependencies
go mod tidy
```

### Linting (as configured in CI)
```bash
# Install golangci-lint if not present
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Architecture and Code Structure

### Core Client Architecture

The library follows a centralized client pattern where all API operations go through a single `Client` struct that manages:

1. **Authentication State**: OAuth 2.0 tokens with automatic renewal
   - Token expiry tracking with renewal buffer (10 minutes before expiry)
   - Separate auth URLs for production and test environments
   
2. **Request Handling**: All requests flow through `doRequest()` method which:
   - Ensures authentication is current
   - Adds OAuth bearer token headers
   - Handles both relative API paths and absolute HAL links
   - Extracts rate limit information from response headers

3. **Multi-tenant Support**: 
   - Client can list available tenants via root endpoint
   - Tenant ID can be set during initialization or changed dynamically
   - All tenant-scoped operations require tenant ID

### API Integration Patterns

1. **HAL+JSON Support**: The API returns HAL (Hypertext Application Language) responses with discoverable links. The implementation handles this by:
   - Storing HAL links in response structs
   - Using absolute URLs from HAL links for operations like `CompleteUpload`

2. **Two-Phase Print Job Submission**:
   - Submit job metadata → receive upload links
   - Upload document to cloud storage (Azure/GCP)
   - Complete upload via HAL link to trigger printing

3. **Webhook Security**: HMAC-SHA512 signature validation with:
   - Timestamp validation to prevent replay attacks
   - Support for key rotation (old/new secret)
   - Payload format: `{timestamp}.{body}`

### Error Handling Strategy

All API responses include a `success` boolean and potential error information. The pattern used throughout:
- Parse response into typed structs
- Check `success` flag
- Return formatted error with description and error ID

### Testing Approach

Tests use HTTP test servers to mock API responses, validating:
- Request formation (headers, body, query parameters)
- Response parsing
- Error conditions
- OAuth token refresh logic

### Key Implementation Details

1. **Rate Limiting**: Client tracks `X-Rate-Limit-Remaining` and `X-Rate-Limit-Reset` headers, exposed via `GetRateLimitInfo()`

2. **File Type Detection**: `PrintFile()` automatically detects content type based on file extension

3. **Pagination**: List endpoints support pagination with consistent options pattern

4. **Context Support**: All API methods accept context for cancellation/timeout control

## API Endpoints Reference

Production base: `https://api.printix.net`
Test auth: `https://auth.testenv.printix.net/oauth/token`

Key endpoints implemented:
- `/cloudprint` - Root endpoint for tenant discovery
- `/cloudprint/tenants/{tenantId}/printers` - Printer management
- `/cloudprint/tenants/{tenantId}/printers/{printerId}/jobs` - Job submission
- `/cloudprint/tenants/{tenantId}/jobs` - Job listing/management
- `/cloudprint/tenants/{tenantId}/users` - User management
- `/cloudprint/tenants/{tenantId}/groups` - Group management
- `/cloudprint/completeUpload` - Complete job upload (via HAL link)