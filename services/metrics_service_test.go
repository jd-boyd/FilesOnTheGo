package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsService(t *testing.T) {
	m := NewMetricsService()

	assert.NotNil(t, m)
	assert.NotNil(t, m.httpRequestsTotal)
	assert.NotNil(t, m.httpRequestDuration)
	assert.NotNil(t, m.authAttemptsTotal)
	assert.False(t, m.startTime.IsZero())
}

func TestMetricsService_RecordHTTPRequest(t *testing.T) {
	m := NewMetricsService()

	// Record some requests
	m.RecordHTTPRequest("GET", "/api/health", 200, 10*time.Millisecond)
	m.RecordHTTPRequest("GET", "/api/health", 200, 20*time.Millisecond)
	m.RecordHTTPRequest("POST", "/api/auth/login", 200, 50*time.Millisecond)
	m.RecordHTTPRequest("POST", "/api/auth/login", 401, 30*time.Millisecond)

	metrics := m.GetMetrics()

	// Check request counter metrics
	assert.Contains(t, metrics, `filesonthego_http_requests_total{method="GET",path="/api/health",status="200"} 2`)
	assert.Contains(t, metrics, `filesonthego_http_requests_total{method="POST",path="/api/auth/login",status="200"} 1`)
	assert.Contains(t, metrics, `filesonthego_http_requests_total{method="POST",path="/api/auth/login",status="401"} 1`)

	// Check histogram metrics exist
	assert.Contains(t, metrics, "filesonthego_http_request_duration_seconds_bucket")
	assert.Contains(t, metrics, "filesonthego_http_request_duration_seconds_sum")
	assert.Contains(t, metrics, "filesonthego_http_request_duration_seconds_count")
}

func TestMetricsService_RecordHTTPRequest_NormalizesUUIDs(t *testing.T) {
	m := NewMetricsService()

	// Record requests with UUID paths
	m.RecordHTTPRequest("GET", "/api/files/550e8400-e29b-41d4-a716-446655440000/download", 200, 10*time.Millisecond)
	m.RecordHTTPRequest("GET", "/api/files/123e4567-e89b-12d3-a456-426614174000/download", 200, 10*time.Millisecond)

	metrics := m.GetMetrics()

	// UUIDs should be normalized to {id}
	assert.Contains(t, metrics, `path="/api/files/{id}/download"`)
	assert.Contains(t, metrics, `} 2`) // Both requests should be counted together
}

func TestMetricsService_RecordHTTPRequest_NormalizesNumericIDs(t *testing.T) {
	m := NewMetricsService()

	// Record requests with numeric IDs
	m.RecordHTTPRequest("GET", "/api/files/12345/download", 200, 10*time.Millisecond)
	m.RecordHTTPRequest("GET", "/api/files/67890/download", 200, 10*time.Millisecond)

	metrics := m.GetMetrics()

	// Numeric IDs should be normalized to {id}
	assert.Contains(t, metrics, `path="/api/files/{id}/download"`)
}

func TestMetricsService_RecordFileUpload(t *testing.T) {
	m := NewMetricsService()

	m.RecordFileUpload(1024)
	m.RecordFileUpload(2048)

	metrics := m.GetMetrics()

	assert.Contains(t, metrics, "filesonthego_file_uploads_total 2")
	assert.Contains(t, metrics, "filesonthego_upload_bytes_total 3072")
}

func TestMetricsService_RecordFileDownload(t *testing.T) {
	m := NewMetricsService()

	m.RecordFileDownload(5000)
	m.RecordFileDownload(3000)

	metrics := m.GetMetrics()

	assert.Contains(t, metrics, "filesonthego_file_downloads_total 2")
	assert.Contains(t, metrics, "filesonthego_download_bytes_total 8000")
}

func TestMetricsService_RecordShareAccess(t *testing.T) {
	m := NewMetricsService()

	m.RecordShareAccess()
	m.RecordShareAccess()
	m.RecordShareAccess()

	metrics := m.GetMetrics()

	assert.Contains(t, metrics, "filesonthego_share_access_total 3")
}

func TestMetricsService_RecordAuthAttempt(t *testing.T) {
	m := NewMetricsService()

	m.RecordAuthAttempt(true)
	m.RecordAuthAttempt(true)
	m.RecordAuthAttempt(false)

	metrics := m.GetMetrics()

	assert.Contains(t, metrics, `filesonthego_auth_attempts_total{result="success"} 2`)
	assert.Contains(t, metrics, `filesonthego_auth_attempts_total{result="failure"} 1`)
}

func TestMetricsService_ActiveConnections(t *testing.T) {
	m := NewMetricsService()

	m.IncrementActiveConnections()
	m.IncrementActiveConnections()

	metrics := m.GetMetrics()
	assert.Contains(t, metrics, "filesonthego_active_connections 2")

	m.DecrementActiveConnections()
	metrics = m.GetMetrics()
	assert.Contains(t, metrics, "filesonthego_active_connections 1")
}

func TestMetricsService_GetMetrics_HasAllSections(t *testing.T) {
	m := NewMetricsService()

	metrics := m.GetMetrics()

	// Check that all metric sections are present
	assert.Contains(t, metrics, "# HELP filesonthego_info")
	assert.Contains(t, metrics, "# TYPE filesonthego_info gauge")
	assert.Contains(t, metrics, "# HELP filesonthego_uptime_seconds")
	assert.Contains(t, metrics, "# TYPE filesonthego_uptime_seconds gauge")
	assert.Contains(t, metrics, "# HELP filesonthego_http_requests_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_http_requests_total counter")
	assert.Contains(t, metrics, "# HELP filesonthego_http_request_duration_seconds")
	assert.Contains(t, metrics, "# TYPE filesonthego_http_request_duration_seconds histogram")
	assert.Contains(t, metrics, "# HELP filesonthego_active_connections")
	assert.Contains(t, metrics, "# TYPE filesonthego_active_connections gauge")
	assert.Contains(t, metrics, "# HELP filesonthego_file_uploads_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_file_uploads_total counter")
	assert.Contains(t, metrics, "# HELP filesonthego_upload_bytes_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_upload_bytes_total counter")
	assert.Contains(t, metrics, "# HELP filesonthego_file_downloads_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_file_downloads_total counter")
	assert.Contains(t, metrics, "# HELP filesonthego_download_bytes_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_download_bytes_total counter")
	assert.Contains(t, metrics, "# HELP filesonthego_share_access_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_share_access_total counter")
	assert.Contains(t, metrics, "# HELP filesonthego_auth_attempts_total")
	assert.Contains(t, metrics, "# TYPE filesonthego_auth_attempts_total counter")
}

func TestMetricsService_Uptime(t *testing.T) {
	m := NewMetricsService()

	// Wait a short time
	time.Sleep(10 * time.Millisecond)

	metrics := m.GetMetrics()

	// Extract uptime value
	assert.Contains(t, metrics, "filesonthego_uptime_seconds")
	// Uptime should be greater than 0
	lines := strings.Split(metrics, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "filesonthego_uptime_seconds ") {
			parts := strings.Fields(line)
			assert.Len(t, parts, 2)
			// The value should be a positive number
			assert.NotEqual(t, "0", parts[1])
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "static path",
			input:    "/api/health",
			expected: "/api/health",
		},
		{
			name:     "uuid path segment",
			input:    "/api/files/550e8400-e29b-41d4-a716-446655440000/download",
			expected: "/api/files/{id}/download",
		},
		{
			name:     "numeric id",
			input:    "/api/users/12345",
			expected: "/api/users/{id}",
		},
		{
			name:     "long token",
			input:    "/share/abcdef1234567890abcdef1234567890",
			expected: "/share/{token}",
		},
		{
			name:     "multiple segments",
			input:    "/api/users/123/files/456",
			expected: "/api/users/{id}/files/{id}",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"not-a-uuid", false},
		{"550e8400e29b41d4a716446655440000", false}, // No dashes
		{"", false},
		{"12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"12345", true},
		{"0", true},
		{"123abc", false},
		{"", false},
		{"abc", false},
		{"-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isNumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"ABC", true},
		{"123", true},
		{"abc-def", false},
		{"abc_def", false},
		{"", true}, // Empty string has no non-alphanumeric chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isAlphanumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHistogramData_Observe(t *testing.T) {
	h := newHistogramData()

	// Observe some values
	h.observe(0.005) // Should go in 0.005, 0.01, 0.025, etc buckets
	h.observe(0.1)   // Should go in 0.1, 0.25, etc buckets
	h.observe(1.5)   // Should go in 2.5, 5, 10 buckets

	assert.Equal(t, uint64(3), h.count)
	assert.InDelta(t, 1.605, h.sum, 0.001)
}

func TestMetricsService_ConcurrentAccess(t *testing.T) {
	m := NewMetricsService()

	// Run concurrent operations
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.RecordHTTPRequest("GET", "/api/test", 200, time.Millisecond)
				m.RecordFileUpload(100)
				m.RecordFileDownload(100)
				m.RecordShareAccess()
				m.RecordAuthAttempt(true)
				m.IncrementActiveConnections()
				m.DecrementActiveConnections()
				_ = m.GetMetrics()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final counts
	metrics := m.GetMetrics()
	assert.Contains(t, metrics, "filesonthego_file_uploads_total 1000")
	assert.Contains(t, metrics, "filesonthego_file_downloads_total 1000")
	assert.Contains(t, metrics, "filesonthego_share_access_total 1000")
}
