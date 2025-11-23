// Package services provides business logic for the FilesOnTheGo application.
package services

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsService provides Prometheus-style metrics collection and exposure.
type MetricsService struct {
	mu sync.RWMutex

	// HTTP request metrics
	httpRequestsTotal   map[string]*uint64 // key: method:path:status
	httpRequestDuration map[string]*histogramData

	// Application-specific metrics
	fileUploadsTotal   uint64
	fileDownloadsTotal uint64
	uploadBytesTotal   uint64
	downloadBytesTotal uint64
	activeConnections  int64
	shareAccessTotal   uint64
	authAttemptsTotal  map[string]*uint64 // key: success/failure

	// System metrics
	startTime time.Time
}

// histogramData holds histogram bucket data for latency metrics.
type histogramData struct {
	count   uint64
	sum     float64
	buckets map[float64]*uint64 // bucket upper bound -> count
}

// DefaultLatencyBuckets defines the default histogram buckets for latency (in seconds).
var DefaultLatencyBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// NewMetricsService creates a new MetricsService instance.
func NewMetricsService() *MetricsService {
	return &MetricsService{
		httpRequestsTotal:   make(map[string]*uint64),
		httpRequestDuration: make(map[string]*histogramData),
		authAttemptsTotal: map[string]*uint64{
			"success": new(uint64),
			"failure": new(uint64),
		},
		startTime: time.Now(),
	}
}

// RecordHTTPRequest records an HTTP request with its duration.
func (m *MetricsService) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	// Normalize path to avoid high cardinality
	normalizedPath := normalizePath(path)
	key := fmt.Sprintf("%s:%s:%d", method, normalizedPath, statusCode)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment request counter
	if m.httpRequestsTotal[key] == nil {
		m.httpRequestsTotal[key] = new(uint64)
	}
	atomic.AddUint64(m.httpRequestsTotal[key], 1)

	// Record duration in histogram
	histKey := fmt.Sprintf("%s:%s", method, normalizedPath)
	if m.httpRequestDuration[histKey] == nil {
		m.httpRequestDuration[histKey] = newHistogramData()
	}
	m.httpRequestDuration[histKey].observe(duration.Seconds())
}

// RecordFileUpload records a file upload event.
func (m *MetricsService) RecordFileUpload(sizeBytes int64) {
	atomic.AddUint64(&m.fileUploadsTotal, 1)
	if sizeBytes > 0 {
		atomic.AddUint64(&m.uploadBytesTotal, uint64(sizeBytes))
	}
}

// RecordFileDownload records a file download event.
func (m *MetricsService) RecordFileDownload(sizeBytes int64) {
	atomic.AddUint64(&m.fileDownloadsTotal, 1)
	if sizeBytes > 0 {
		atomic.AddUint64(&m.downloadBytesTotal, uint64(sizeBytes))
	}
}

// RecordShareAccess records a share link access.
func (m *MetricsService) RecordShareAccess() {
	atomic.AddUint64(&m.shareAccessTotal, 1)
}

// RecordAuthAttempt records an authentication attempt.
func (m *MetricsService) RecordAuthAttempt(success bool) {
	if success {
		atomic.AddUint64(m.authAttemptsTotal["success"], 1)
	} else {
		atomic.AddUint64(m.authAttemptsTotal["failure"], 1)
	}
}

// IncrementActiveConnections increments the active connections gauge.
func (m *MetricsService) IncrementActiveConnections() {
	atomic.AddInt64(&m.activeConnections, 1)
}

// DecrementActiveConnections decrements the active connections gauge.
func (m *MetricsService) DecrementActiveConnections() {
	atomic.AddInt64(&m.activeConnections, -1)
}

// GetMetrics returns the current metrics in Prometheus text format.
func (m *MetricsService) GetMetrics() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	// Write application info metric
	sb.WriteString("# HELP filesonthego_info Application information\n")
	sb.WriteString("# TYPE filesonthego_info gauge\n")
	sb.WriteString("filesonthego_info{version=\"0.1.0\"} 1\n\n")

	// Write uptime metric
	uptime := time.Since(m.startTime).Seconds()
	sb.WriteString("# HELP filesonthego_uptime_seconds Time since application start\n")
	sb.WriteString("# TYPE filesonthego_uptime_seconds gauge\n")
	sb.WriteString(fmt.Sprintf("filesonthego_uptime_seconds %.3f\n\n", uptime))

	// Write HTTP request total metrics
	sb.WriteString("# HELP filesonthego_http_requests_total Total number of HTTP requests\n")
	sb.WriteString("# TYPE filesonthego_http_requests_total counter\n")
	keys := make([]string, 0, len(m.httpRequestsTotal))
	for k := range m.httpRequestsTotal {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) == 3 {
			method, path, status := parts[0], parts[1], parts[2]
			count := atomic.LoadUint64(m.httpRequestsTotal[key])
			sb.WriteString(fmt.Sprintf("filesonthego_http_requests_total{method=\"%s\",path=\"%s\",status=\"%s\"} %d\n",
				method, path, status, count))
		}
	}
	sb.WriteString("\n")

	// Write HTTP request duration histogram
	sb.WriteString("# HELP filesonthego_http_request_duration_seconds HTTP request duration in seconds\n")
	sb.WriteString("# TYPE filesonthego_http_request_duration_seconds histogram\n")
	histKeys := make([]string, 0, len(m.httpRequestDuration))
	for k := range m.httpRequestDuration {
		histKeys = append(histKeys, k)
	}
	sort.Strings(histKeys)
	for _, key := range histKeys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			method, path := parts[0], parts[1]
			hist := m.httpRequestDuration[key]
			hist.writePrometheus(&sb, "filesonthego_http_request_duration_seconds", map[string]string{
				"method": method,
				"path":   path,
			})
		}
	}
	sb.WriteString("\n")

	// Write active connections gauge
	sb.WriteString("# HELP filesonthego_active_connections Current number of active connections\n")
	sb.WriteString("# TYPE filesonthego_active_connections gauge\n")
	sb.WriteString(fmt.Sprintf("filesonthego_active_connections %d\n\n", atomic.LoadInt64(&m.activeConnections)))

	// Write file upload metrics
	sb.WriteString("# HELP filesonthego_file_uploads_total Total number of file uploads\n")
	sb.WriteString("# TYPE filesonthego_file_uploads_total counter\n")
	sb.WriteString(fmt.Sprintf("filesonthego_file_uploads_total %d\n\n", atomic.LoadUint64(&m.fileUploadsTotal)))

	sb.WriteString("# HELP filesonthego_upload_bytes_total Total bytes uploaded\n")
	sb.WriteString("# TYPE filesonthego_upload_bytes_total counter\n")
	sb.WriteString(fmt.Sprintf("filesonthego_upload_bytes_total %d\n\n", atomic.LoadUint64(&m.uploadBytesTotal)))

	// Write file download metrics
	sb.WriteString("# HELP filesonthego_file_downloads_total Total number of file downloads\n")
	sb.WriteString("# TYPE filesonthego_file_downloads_total counter\n")
	sb.WriteString(fmt.Sprintf("filesonthego_file_downloads_total %d\n\n", atomic.LoadUint64(&m.fileDownloadsTotal)))

	sb.WriteString("# HELP filesonthego_download_bytes_total Total bytes downloaded\n")
	sb.WriteString("# TYPE filesonthego_download_bytes_total counter\n")
	sb.WriteString(fmt.Sprintf("filesonthego_download_bytes_total %d\n\n", atomic.LoadUint64(&m.downloadBytesTotal)))

	// Write share access metrics
	sb.WriteString("# HELP filesonthego_share_access_total Total number of share link accesses\n")
	sb.WriteString("# TYPE filesonthego_share_access_total counter\n")
	sb.WriteString(fmt.Sprintf("filesonthego_share_access_total %d\n\n", atomic.LoadUint64(&m.shareAccessTotal)))

	// Write auth metrics
	sb.WriteString("# HELP filesonthego_auth_attempts_total Total number of authentication attempts\n")
	sb.WriteString("# TYPE filesonthego_auth_attempts_total counter\n")
	sb.WriteString(fmt.Sprintf("filesonthego_auth_attempts_total{result=\"success\"} %d\n",
		atomic.LoadUint64(m.authAttemptsTotal["success"])))
	sb.WriteString(fmt.Sprintf("filesonthego_auth_attempts_total{result=\"failure\"} %d\n\n",
		atomic.LoadUint64(m.authAttemptsTotal["failure"])))

	return sb.String()
}

// Handler returns an http.HandlerFunc that serves metrics in Prometheus format.
func (m *MetricsService) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(m.GetMetrics()))
	}
}

// normalizePath normalizes URL paths to reduce cardinality.
// It replaces UUIDs, numeric IDs, and other dynamic segments with placeholders.
func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Skip empty parts
		if part == "" {
			continue
		}
		// Replace UUIDs
		if isUUID(part) {
			parts[i] = "{id}"
			continue
		}
		// Replace numeric IDs
		if isNumeric(part) {
			parts[i] = "{id}"
			continue
		}
		// Replace long alphanumeric strings (likely tokens or hashes)
		if len(part) > 20 && isAlphanumeric(part) {
			parts[i] = "{token}"
			continue
		}
	}
	return strings.Join(parts, "/")
}

// isUUID checks if a string looks like a UUID.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// isNumeric checks if a string is all digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isAlphanumeric checks if a string is all alphanumeric.
func isAlphanumeric(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

// newHistogramData creates a new histogram with default buckets.
func newHistogramData() *histogramData {
	buckets := make(map[float64]*uint64)
	for _, b := range DefaultLatencyBuckets {
		buckets[b] = new(uint64)
	}
	return &histogramData{
		buckets: buckets,
	}
}

// observe records a value in the histogram.
func (h *histogramData) observe(value float64) {
	h.count++
	h.sum += value
	for _, bound := range DefaultLatencyBuckets {
		if value <= bound {
			atomic.AddUint64(h.buckets[bound], 1)
		}
	}
}

// writePrometheus writes the histogram in Prometheus format.
func (h *histogramData) writePrometheus(sb *strings.Builder, name string, labels map[string]string) {
	labelStr := formatLabels(labels)

	// Write bucket values
	var cumulativeCount uint64
	for _, bound := range DefaultLatencyBuckets {
		cumulativeCount += atomic.LoadUint64(h.buckets[bound])
		sb.WriteString(fmt.Sprintf("%s_bucket{%sle=\"%s\"} %d\n",
			name, labelStr, formatFloat(bound), cumulativeCount))
	}
	// +Inf bucket
	sb.WriteString(fmt.Sprintf("%s_bucket{%sle=\"+Inf\"} %d\n", name, labelStr, h.count))

	// Write sum and count
	sb.WriteString(fmt.Sprintf("%s_sum{%s} %s\n", name, strings.TrimSuffix(labelStr, ","), formatFloat(h.sum)))
	sb.WriteString(fmt.Sprintf("%s_count{%s} %d\n", name, strings.TrimSuffix(labelStr, ","), h.count))
}

// formatLabels formats a map of labels as a Prometheus label string.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, labels[k]))
	}
	return strings.Join(parts, ",") + ","
}

// formatFloat formats a float64 for Prometheus output.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
