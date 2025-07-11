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
	"time"
)

// WebhookEvent represents a Printix webhook event.
type WebhookEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
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

// ParseWebhookEvent parses a webhook event from the request body.
func ParseWebhookEvent(r *http.Request) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		return nil, fmt.Errorf("decoding webhook event: %w", err)
	}
	return &event, nil
}

// ParseJobStatusChange parses job status change data from webhook event.
func ParseJobStatusChange(event *WebhookEvent) (*WebhookJobStatusChange, error) {
	if event.Type != "job.status.changed" {
		return nil, fmt.Errorf("incorrect event type: %s", event.Type)
	}

	var data WebhookJobStatusChange
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return nil, fmt.Errorf("parsing job status change: %w", err)
	}

	return &data, nil
}
