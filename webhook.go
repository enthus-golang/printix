package printix

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhookEvent represents a Printix webhook event.
type WebhookEvent struct {
	Name string `json:"name"` // e.g., "RESOURCE.TENANT_USER.CREATE"
	Href string `json:"href"` // Link to the resource
	Time float64 `json:"time"` // Unix timestamp with milliseconds
}

// WebhookPayload represents the full webhook payload.
type WebhookPayload struct {
	Emitted float64        `json:"emitted"` // Unix timestamp when webhook was emitted
	Events  []WebhookEvent `json:"events"`  // Array of events
}

// WebhookJobStatusChange represents a job status change event.
type WebhookJobStatusChange struct {
	JobID     string `json:"jobId"`
	PrinterID string `json:"printerId"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

// WebhookValidator validates incoming webhook requests.
type WebhookValidator struct {
	sharedSecret    string
	oldSharedSecret string // For zero-downtime key rotation
	timestampWindow time.Duration
}

// NewWebhookValidator creates a new webhook validator.
func NewWebhookValidator(sharedSecret string) *WebhookValidator {
	return &WebhookValidator{
		sharedSecret:    sharedSecret,
		timestampWindow: 15 * time.Minute,
	}
}

// SetOldSecret sets the old shared secret for key rotation.
func (v *WebhookValidator) SetOldSecret(oldSecret string) {
	v.oldSharedSecret = oldSecret
}

// ValidateRequest validates an incoming webhook request.
func (v *WebhookValidator) ValidateRequest(r *http.Request) error {
	// Check timestamp to prevent replay attacks
	timestampStr := r.Header.Get("X-Printix-Timestamp")
	if timestampStr == "" {
		return fmt.Errorf("missing timestamp header")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	requestTime := time.Unix(timestamp, 0)
	if time.Since(requestTime).Abs() > v.timestampWindow {
		return fmt.Errorf("timestamp outside acceptable window")
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Validate signature
	signature := r.Header.Get("X-Printix-Signature")
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// Create payload for signature
	payload := fmt.Sprintf("%s.%s", timestampStr, string(body))

	// Check with current secret
	if v.verifySignature(payload, signature, v.sharedSecret) {
		return nil
	}

	// Check with old secret if set (for key rotation)
	if v.oldSharedSecret != "" && v.verifySignature(payload, signature, v.oldSharedSecret) {
		return nil
	}

	return fmt.Errorf("invalid signature")
}

// verifySignature verifies the HMAC-SHA512 signature.
func (v *WebhookValidator) verifySignature(payload, signature, secret string) bool {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ParseWebhookPayload parses a webhook payload from the request body.
func ParseWebhookPayload(r *http.Request) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding webhook payload: %w", err)
	}
	return &payload, nil
}

// IsUserCreateEvent checks if the event is a user creation event.
func (e *WebhookEvent) IsUserCreateEvent() bool {
	return e.Name == "RESOURCE.TENANT_USER.CREATE"
}

// IsJobStatusChangeEvent checks if the event is a job status change event.
func (e *WebhookEvent) IsJobStatusChangeEvent() bool {
	return strings.Contains(e.Name, "JOB") && strings.Contains(e.Name, "STATUS")
}

// GetTimestamp returns the event timestamp as a time.Time.
func (e *WebhookEvent) GetTimestamp() time.Time {
	return time.Unix(int64(e.Time), int64((e.Time-float64(int64(e.Time)))*1e9))
}
