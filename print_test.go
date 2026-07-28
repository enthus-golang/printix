package printix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Submit(t *testing.T) {
	tests := []struct {
		name        string
		job         *PrintJob
		setupServer func() *httptest.Server
		wantJobID   string
		wantSuccess bool
		wantErr     bool
		errContains string
	}{
		{
			name: "successful submission",
			job: &PrintJob{
				PrinterID: "printer-123",
				QueueID:   "printer-123",
				Title:     "Test Document",
				User:      "Test",
			},
			setupServer: func() *httptest.Server {
				var server *httptest.Server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oauth/token":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "test-token",
							"expires_in":   3600,
						})
					case "/cloudprint/tenants/test-tenant/printers/printer-123/queues/printer-123/submit":
						// Check query parameters instead of body for v1.0 API
						assert.Equal(t, "Test Document", r.URL.Query().Get("title"))
						assert.Equal(t, "Test", r.URL.Query().Get("user"))

						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
							"job": map[string]interface{}{
								"id":          "job-456",
								"createTime":  "2025-07-15T15:02:13.141525320Z",
								"updateTime":  "2025-07-15T15:02:13.248Z",
								"status":      "Created",
								"ownerId":     "owner-123",
								"contentType": "application/pdf",
								"title":       "Test Document",
								"_links": map[string]interface{}{
									"self": map[string]interface{}{
										"href": server.URL + "/cloudprint/tenants/test-tenant/jobs/job-456",
									},
								},
							},
							"uploadLinks": []map[string]interface{}{
								{
									"url":     "https://storage.example.com/upload",
									"headers": map[string]string{},
									"type":    "Azure",
								},
							},
							"_links": map[string]interface{}{
								"uploadCompleted": map[string]interface{}{
									"href": server.URL + "/cloudprint/jobs/job-456/uploadCompleted",
								},
							},
						})
					}
				}))
				return server
			},
			wantJobID:   "job-456",
			wantSuccess: true,
			wantErr: false,
		},
		{
			name: "submission with test mode",
			job: &PrintJob{
				PrinterID: "printer-123",
				QueueID:   "printer-123",
				Title:     "Test Document",
				TestMode:  true,
			},
			setupServer: func() *httptest.Server {
				var server *httptest.Server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oauth/token":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "test-token",
							"expires_in":   3600,
						})
					case "/cloudprint/tenants/test-tenant/printers/printer-123/queues/printer-123/submit":
						assert.Equal(t, "true", r.URL.Query().Get("test"))

						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
							"job": map[string]interface{}{
								"id":          "test-job-789",
								"createTime":  "2025-07-15T15:02:13.141525320Z",
								"updateTime":  "2025-07-15T15:02:13.248Z",
								"status":      "Created",
								"ownerId":     "owner-123",
								"contentType": "application/pdf",
								"title":       "Test Document",
								"_links": map[string]interface{}{
									"self": map[string]interface{}{
										"href": server.URL + "/cloudprint/tenants/test-tenant/jobs/test-job-789",
									},
								},
							},
							"uploadLinks": []map[string]interface{}{
								{
									"url":     "https://test.storage.example.com/upload",
									"headers": map[string]string{},
									"type":    "Azure",
								},
							},
							"_links": map[string]interface{}{
								"uploadCompleted": map[string]interface{}{
									"href": server.URL + "/cloudprint/jobs/test-job-789/uploadCompleted",
								},
							},
						})
					}
				}))
				return server
			},
			wantJobID:   "test-job-789",
			wantSuccess: true,
			wantErr: false,
		},
		{
			name: "submission with v1.1 API",
			job: &PrintJob{
				PrinterID: "printer-123",
				QueueID:   "printer-123",
				Title:     "Test Document",
				User:      "Test",
				UseV11:    true,
				Color:     &[]bool{false}[0],
				Duplex:    "NONE",
				Copies:    &[]int{2}[0],
			},
			setupServer: func() *httptest.Server {
				var server *httptest.Server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oauth/token":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "test-token",
							"expires_in":   3600,
						})
					case "/cloudprint/tenants/test-tenant/printers/printer-123/queues/printer-123/submit":
						// Check v1.1 specific requirements
						assert.Equal(t, "1.1", r.Header.Get("version"))
						assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
						
						// Check query parameters
						assert.Equal(t, "Test Document", r.URL.Query().Get("title"))
						assert.Equal(t, "Test", r.URL.Query().Get("user"))
						assert.Equal(t, "true", r.URL.Query().Get("releaseImmediately"))
						
						// Check request body
						var body map[string]interface{}
						_ = json.NewDecoder(r.Body).Decode(&body)
						assert.Equal(t, false, body["color"])
						assert.Equal(t, "NONE", body["duplex"])
						assert.Equal(t, float64(2), body["copies"])
						assert.Equal(t, nil, body["userMapping"])

						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
							"job": map[string]interface{}{
								"id":          "v11-job-123",
								"createTime":  "2025-07-15T15:02:13.141525320Z",
								"updateTime":  "2025-07-15T15:02:13.248Z",
								"status":      "Created",
								"ownerId":     "owner-123",
								"contentType": "application/pdf",
								"title":       "Test Document",
								"_links": map[string]interface{}{
									"self": map[string]interface{}{
										"href": server.URL + "/cloudprint/tenants/test-tenant/jobs/v11-job-123",
									},
								},
							},
							"uploadLinks": []map[string]interface{}{
								{
									"url":     "https://storage.example.com/upload",
									"headers": map[string]string{},
									"type":    "Azure",
								},
							},
							"_links": map[string]interface{}{
								"uploadCompleted": map[string]interface{}{
									"href": server.URL + "/cloudprint/jobs/v11-job-123/uploadCompleted",
								},
							},
						})
					}
				}))
				return server
			},
			wantJobID:   "v11-job-123",
			wantSuccess: true,
			wantErr: false,
		},
		{
			name: "submission failure",
			job: &PrintJob{
				PrinterID: "printer-123",
				QueueID:   "printer-123",
				Title:     "Test Document",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oauth/token":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "test-token",
							"expires_in":   3600,
						})
					case "/cloudprint/tenants/test-tenant/printers/printer-123/queues/printer-123/submit":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success":          false,
							"errorDescription": "Printer not found",
							"errorId":          "ERR001",
						})
					}
				}))
			},
			wantErr:     true,
			errContains: "submit failed: Printer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := New("test-id", "test-secret", WithBaseURL(server.URL), WithAuthURL(server.URL+"/oauth/token"), WithTenantID("test-tenant"))
			got, err := client.Submit(context.Background(), tt.job)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantSuccess, got.Success)
				assert.Equal(t, tt.wantJobID, got.Job.ID)
				assert.NotEmpty(t, got.Job.Title)
				assert.NotEmpty(t, got.Job.Status)
				assert.Greater(t, len(got.UploadLinks), 0)
				if len(got.UploadLinks) > 0 {
					assert.NotEmpty(t, got.UploadLinks[0].URL)
				}
				// Just check that upload completed href is not empty
				assert.NotEmpty(t, got.Links.UploadCompleted.Href)
			}
		})
	}
}

func TestClient_CompleteUpload(t *testing.T) {
	tests := []struct {
		name        string
		jobID       string
		setupServer func() *httptest.Server
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful completion",
			jobID: "job-123",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oauth/token":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "test-token",
							"expires_in":   3600,
						})
					case "/cloudprint/jobs/job-123/uploadCompleted":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
						})
					}
				}))
			},
			wantErr: false,
		},
		{
			name:  "completion failure",
			jobID: "job-123",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oauth/token":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "test-token",
							"expires_in":   3600,
						})
					case "/cloudprint/jobs/job-123/uploadCompleted":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success":          false,
							"errorDescription": "Job not found",
							"errorId":          "ERR002",
						})
					}
				}))
			},
			wantErr:     true,
			errContains: "complete upload failed: Job not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := New("test-id", "test-secret", WithBaseURL(server.URL), WithAuthURL(server.URL+"/oauth/token"), WithTenantID("test-tenant"))
			err := client.CompleteUpload(context.Background(), server.URL+"/cloudprint/jobs/job-123/uploadCompleted")

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
