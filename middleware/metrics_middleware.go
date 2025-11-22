// Package middleware provides HTTP middleware for the FilesOnTheGo application.
package middleware

import (
	"time"

	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase/core"
)

// MetricsMiddleware creates middleware that records HTTP request metrics.
func MetricsMiddleware(metricsService *services.MetricsService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Track active connections
		metricsService.IncrementActiveConnections()
		defer metricsService.DecrementActiveConnections()

		// Record start time
		start := time.Now()

		// Get request details before processing
		method := e.Request.Method
		path := e.Request.URL.Path

		// Process the request by calling Next()
		err := e.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get status code from the response
		// Note: PocketBase uses a custom response writer, we need to get status from it
		statusCode := getStatusCode(e)

		// Record the metrics
		metricsService.RecordHTTPRequest(method, path, statusCode, duration)

		return err
	}
}

// getStatusCode attempts to get the HTTP status code from the response.
// Since PocketBase wraps the response writer, we check for common patterns.
func getStatusCode(e *core.RequestEvent) int {
	// PocketBase's response writer may have a Status method
	// Default to 200 if we can't determine the status
	// The actual status is recorded by PocketBase internally
	if e.Response != nil {
		// Try to get status from the response recorder if available
		type statusRecorder interface {
			Status() int
		}
		if sr, ok := e.Response.(statusRecorder); ok {
			status := sr.Status()
			if status > 0 {
				return status
			}
		}
	}
	return 200 // Default assumption
}
