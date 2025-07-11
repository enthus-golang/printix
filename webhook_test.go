package printix

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
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

func TestParseWebhookPayload(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    *WebhookPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			body: `{
				"emitted": 1718093846.488,
				"events": [
					{
						"name": "RESOURCE.TENANT_USER.CREATE",
						"href": "https://api.printix.net/cloudprint/tenants/123/users/456",
						"time": 1718093846.488
					}
				]
			}`,
			want: &WebhookPayload{
				Emitted: 1718093846.488,
				Events: []WebhookEvent{
					{
						Name: "RESOURCE.TENANT_USER.CREATE",
						Href: "https://api.printix.net/cloudprint/tenants/123/users/456",
						Time: 1718093846.488,
					},
				},
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
			got, err := ParseWebhookPayload(req)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.Emitted, got.Emitted)
				assert.Equal(t, len(tt.want.Events), len(got.Events))
				if len(got.Events) > 0 {
					assert.Equal(t, tt.want.Events[0].Name, got.Events[0].Name)
					assert.Equal(t, tt.want.Events[0].Href, got.Events[0].Href)
				}
			}
		})
	}
}

func TestWebhookEventMethods(t *testing.T) {
	tests := []struct {
		name  string
		event WebhookEvent
		want  bool
	}{
		{
			name: "user create event",
			event: WebhookEvent{
				Name: "RESOURCE.TENANT_USER.CREATE",
				Href: "https://api.printix.net/cloudprint/tenants/123/users/456",
				Time: 1718093846.488,
			},
			want: true,
		},
		{
			name: "other event",
			event: WebhookEvent{
				Name: "RESOURCE.PRINTER.UPDATE",
				Href: "https://api.printix.net/cloudprint/tenants/123/printers/456",
				Time: 1718093846.488,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.IsUserCreateEvent()
			assert.Equal(t, tt.want, got)
			
			// Test timestamp conversion
			timestamp := tt.event.GetTimestamp()
			assert.True(t, timestamp.Unix() > 0)
		})
	}
}
