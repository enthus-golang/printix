package printix

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		opts         []Option
		wantBaseURL  string
		wantTestMode bool
	}{
		{
			name:         "default configuration",
			clientID:     "test-id",
			clientSecret: "test-secret",
			opts:         nil,
			wantBaseURL:  defaultBaseURL,
			wantTestMode: false,
		},
		{
			name:         "with custom base URL",
			clientID:     "test-id",
			clientSecret: "test-secret",
			opts:         []Option{WithBaseURL("https://custom.api.com")},
			wantBaseURL:  "https://custom.api.com",
			wantTestMode: false,
		},
		{
			name:         "with test mode",
			clientID:     "test-id",
			clientSecret: "test-secret",
			opts:         []Option{WithTestMode()},
			wantBaseURL:  defaultBaseURL,
			wantTestMode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.clientID, tt.clientSecret, tt.opts...)

			assert.Equal(t, tt.clientID, client.clientID)
			assert.Equal(t, tt.clientSecret, client.clientSecret)
			assert.Equal(t, tt.wantBaseURL, client.baseURL)
			assert.Equal(t, tt.wantTestMode, client.testMode)
			assert.NotNil(t, client.httpClient)
		})
	}
}

func TestClient_authenticate(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		wantErr     bool
		errContains string
	}{
		{
			name: "successful authentication",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/oauth/token", r.URL.Path)
					assert.Equal(t, "POST", r.Method)

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token": "test-token",
						"expires_in":   3600,
						"token_type":   "Bearer",
					})
				}))
			},
			wantErr: false,
		},
		{
			name: "authentication failure",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Invalid credentials"))
				}))
			},
			wantErr:     true,
			errContains: "authentication failed with status 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := New("test-id", "test-secret", WithAuthURL(server.URL+"/oauth/token"))
			err := client.authenticate(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "test-token", client.accessToken)
				assert.True(t, time.Now().Before(client.tokenExpiry))
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		target      interface{}
		wantErr     bool
		errContains string
	}{
		{
			name: "successful response",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body: makeBody(map[string]interface{}{
					"success": true,
					"data":    "test",
				}),
			},
			target:  &Response{},
			wantErr: false,
		},
		{
			name: "error response",
			response: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       makeBody("Bad request"),
			},
			target:      &Response{},
			wantErr:     true,
			errContains: "request failed with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseResponse(tt.response, tt.target)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Helper function to create a response body
func makeBody(v interface{}) io.ReadCloser {
	var buf bytes.Buffer
	switch val := v.(type) {
	case string:
		buf.WriteString(val)
	case map[string]interface{}:
		json.NewEncoder(&buf).Encode(val)
	default:
		json.NewEncoder(&buf).Encode(v)
	}
	return io.NopCloser(&buf)
}
