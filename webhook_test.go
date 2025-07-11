package printix

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookValidator_ValidateRequest(t *testing.T) {
	secret := "test-secret"
	validator := NewWebhookValidator(secret)

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request",
			setupReq: func() *http.Request {
				body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
				timestamp := time.Now().Unix()

				req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
				req.Header.Set("X-Printix-Timestamp", strconv.FormatInt(timestamp, 10))

				// Calculate signature
				payload := fmt.Sprintf("%d.%s", timestamp, body)
				// We need to manually calculate the signature here
				h := hmac.New(sha512.New, []byte(secret))
				h.Write([]byte(payload))
				signature := hex.EncodeToString(h.Sum(nil))
				req.Header.Set("X-Printix-Signature", signature)

				return req
			},
			wantErr: false,
		},
		{
			name: "missing timestamp",
			setupReq: func() *http.Request {
				body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
				req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
				req.Header.Set("X-Printix-Signature", "invalid")
				return req
			},
			wantErr:     true,
			errContains: "missing timestamp header",
		},
		{
			name: "invalid timestamp",
			setupReq: func() *http.Request {
				body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
				req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
				req.Header.Set("X-Printix-Timestamp", "not-a-number")
				req.Header.Set("X-Printix-Signature", "invalid")
				return req
			},
			wantErr:     true,
			errContains: "invalid timestamp",
		},
		{
			name: "timestamp too old",
			setupReq: func() *http.Request {
				body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
				timestamp := time.Now().Add(-30 * time.Minute).Unix()

				req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
				req.Header.Set("X-Printix-Timestamp", strconv.FormatInt(timestamp, 10))
				req.Header.Set("X-Printix-Signature", "invalid")
				return req
			},
			wantErr:     true,
			errContains: "timestamp outside acceptable window",
		},
		{
			name: "missing signature",
			setupReq: func() *http.Request {
				body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
				timestamp := time.Now().Unix()

				req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
				req.Header.Set("X-Printix-Timestamp", strconv.FormatInt(timestamp, 10))
				return req
			},
			wantErr:     true,
			errContains: "missing signature header",
		},
		{
			name: "invalid signature",
			setupReq: func() *http.Request {
				body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
				timestamp := time.Now().Unix()

				req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
				req.Header.Set("X-Printix-Timestamp", strconv.FormatInt(timestamp, 10))
				req.Header.Set("X-Printix-Signature", "wrong-signature")
				return req
			},
			wantErr:     true,
			errContains: "invalid signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			err := validator.ValidateRequest(req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWebhookValidator_KeyRotation(t *testing.T) {
	oldSecret := "old-secret"
	newSecret := "new-secret"

	validator := NewWebhookValidator(newSecret)
	validator.SetOldSecret(oldSecret)

	// Test that old secret still works
	body := `{"event":"job.status.changed","data":{"jobId":"123"}}`
	timestamp := time.Now().Unix()

	req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	req.Header.Set("X-Printix-Timestamp", strconv.FormatInt(timestamp, 10))

	// Calculate signature with old secret
	payload := fmt.Sprintf("%d.%s", timestamp, body)
	h := hmac.New(sha512.New, []byte(oldSecret))
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))
	req.Header.Set("X-Printix-Signature", signature)

	err := validator.ValidateRequest(req)
	require.NoError(t, err)
}

func TestParseWebhookEvent(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    *WebhookEvent
		wantErr bool
	}{
		{
			name: "valid event",
			body: `{
				"id": "evt-123",
				"type": "job.status.changed",
				"timestamp": "2023-01-01T00:00:00Z",
				"data": {"jobId": "job-456"}
			}`,
			want: &WebhookEvent{
				ID:   "evt-123",
				Type: "job.status.changed",
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			body:    `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(tt.body))
			got, err := ParseWebhookEvent(req)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.Type, got.Type)
			}
		})
	}
}

func TestParseJobStatusChange(t *testing.T) {
	tests := []struct {
		name    string
		event   *WebhookEvent
		want    *WebhookJobStatusChange
		wantErr bool
	}{
		{
			name: "valid job status change",
			event: &WebhookEvent{
				Type: "job.status.changed",
				Data: json.RawMessage(`{
					"jobId": "job-123",
					"printerId": "printer-456",
					"status": "completed",
					"message": "Print completed successfully"
				}`),
			},
			want: &WebhookJobStatusChange{
				JobID:     "job-123",
				PrinterID: "printer-456",
				Status:    "completed",
				Message:   "Print completed successfully",
			},
			wantErr: false,
		},
		{
			name: "wrong event type",
			event: &WebhookEvent{
				Type: "printer.online",
				Data: json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "invalid data",
			event: &WebhookEvent{
				Type: "job.status.changed",
				Data: json.RawMessage(`{invalid json}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJobStatusChange(tt.event)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.JobID, got.JobID)
				assert.Equal(t, tt.want.PrinterID, got.PrinterID)
				assert.Equal(t, tt.want.Status, got.Status)
				assert.Equal(t, tt.want.Message, got.Message)
			}
		})
	}
}
