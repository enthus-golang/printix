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
		want        *SubmitResponse
		wantErr     bool
		errContains string
	}{
		{
			name: "successful submission",
			job: &PrintJob{
				PrinterID: "printer-123",
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
					case "/cloudprint/tenants/test-tenant/printers/printer-123/jobs":
						// Check query parameters instead of body for v1.0 API
						assert.Equal(t, "Test Document", r.URL.Query().Get("title"))
						assert.Equal(t, "Test", r.URL.Query().Get("user"))

						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
							"job": map[string]interface{}{
								"id":          "job-456",
								"createTime":  1600344674,
								"updateTime":  1600344674,
								"status":      "Created",
								"ownerId":     "owner-123",
								"contentType": "application/pdf",
								"title":       "Test Document",
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
			want: &SubmitResponse{
				Response: Response{Success: true},
				Job: struct {
					ID          string `json:"id"`
					CreateTime  int64  `json:"createTime"`
					UpdateTime  int64  `json:"updateTime"`
					Status      string `json:"status"`
					OwnerID     string `json:"ownerId"`
					ContentType string `json:"contentType"`
					Title       string `json:"title"`
				}{
					ID:     "job-456",
					Title:  "Test Document",
					Status: "Created",
				},
				UploadLinks: []struct {
					URL     string            `json:"url"`
					Headers map[string]string `json:"headers"`
					Type    string            `json:"type"`
				}{
					{
						URL:     "https://storage.example.com/upload",
						Headers: map[string]string{},
						Type:    "Azure",
					},
				},
				Links: struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					UploadCompleted struct {
						Href string `json:"href"`
					} `json:"uploadCompleted"`
				}{
					UploadCompleted: struct {
						Href string `json:"href"`
					}{
						Href: "", // Will be set dynamically
					},
				},
			},
			wantErr: false,
		},
		{
			name: "submission with test mode",
			job: &PrintJob{
				PrinterID: "printer-123",
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
					case "/cloudprint/tenants/test-tenant/printers/printer-123/jobs":
						assert.Equal(t, "true", r.URL.Query().Get("test"))

						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
							"job": map[string]interface{}{
								"id":          "test-job-789",
								"createTime":  1600344674,
								"updateTime":  1600344674,
								"status":      "Created",
								"ownerId":     "owner-123",
								"contentType": "application/pdf",
								"title":       "Test Document",
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
			want: &SubmitResponse{
				Response: Response{Success: true},
				Job: struct {
					ID          string `json:"id"`
					CreateTime  int64  `json:"createTime"`
					UpdateTime  int64  `json:"updateTime"`
					Status      string `json:"status"`
					OwnerID     string `json:"ownerId"`
					ContentType string `json:"contentType"`
					Title       string `json:"title"`
				}{
					ID:     "test-job-789",
					Title:  "Test Document",
					Status: "Created",
				},
				UploadLinks: []struct {
					URL     string            `json:"url"`
					Headers map[string]string `json:"headers"`
					Type    string            `json:"type"`
				}{
					{
						URL:     "https://test.storage.example.com/upload",
						Headers: map[string]string{},
						Type:    "Azure",
					},
				},
				Links: struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					UploadCompleted struct {
						Href string `json:"href"`
					} `json:"uploadCompleted"`
				}{
					UploadCompleted: struct {
						Href string `json:"href"`
					}{
						Href: "", // Will be set dynamically
					},
				},
			},
			wantErr: false,
		},
		{
			name: "submission failure",
			job: &PrintJob{
				PrinterID: "printer-123",
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
					case "/cloudprint/tenants/test-tenant/printers/printer-123/jobs":
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
				assert.Equal(t, tt.want.Job.ID, got.Job.ID)
				assert.Equal(t, len(tt.want.UploadLinks), len(got.UploadLinks))
				if len(got.UploadLinks) > 0 {
					assert.Equal(t, tt.want.UploadLinks[0].URL, got.UploadLinks[0].URL)
					assert.Equal(t, tt.want.UploadLinks[0].Type, got.UploadLinks[0].Type)
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
