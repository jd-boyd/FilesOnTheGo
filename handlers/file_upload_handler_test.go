package handlers

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
)

// Helper functions for tests

func createMultipartUpload(filename string, content []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)

	writer.Close()
	return body, writer.FormDataContentType()
}

func TestValidateUploadRequest_Success(t *testing.T) {
	content := []byte("test file content")
	body, contentType := createMultipartUpload("test.txt", content)

	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", contentType)

	_ = req.ParseMultipartForm(32 << 20)
	file, header, err := req.FormFile("file")
	assert.NoError(t, err)
	defer file.Close()

	err = ValidateUploadRequest(file, header, 100*1024*1024)
	assert.NoError(t, err)
}

func TestValidateUploadRequest_NoFile(t *testing.T) {
	err := ValidateUploadRequest(nil, nil, 100*1024*1024)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no file provided")
}

func TestValidateUploadRequest_FileTooLarge(t *testing.T) {
	content := make([]byte, 1000)
	body, contentType := createMultipartUpload("large.txt", content)

	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", contentType)

	_ = req.ParseMultipartForm(32 << 20)
	file, header, err := req.FormFile("file")
	assert.NoError(t, err)
	defer file.Close()

	err = ValidateUploadRequest(file, header, 500) // Max 500 bytes
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestValidateUploadRequest_InvalidFilename(t *testing.T) {
	content := []byte("test")
	// Use a filename with null byte
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test\x00.txt")
	part.Write(content)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	parseErr := req.ParseMultipartForm(32 << 20)
	file, header, err := req.FormFile("file")

	// Depending on Go version, null byte in filename may cause parsing error
	// or may return a file that fails validation
	if parseErr != nil || err != nil || file == nil || header == nil {
		// If parsing failed due to invalid filename, that's acceptable
		t.Log("Multipart parsing failed with null byte in filename - acceptable behavior")
		return
	}
	defer file.Close()

	err = ValidateUploadRequest(file, header, 100*1024*1024)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filename")
}

func TestGetFileReader_Success(t *testing.T) {
	content := []byte("test file content")
	body, contentType := createMultipartUpload("test.txt", content)

	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", contentType)

	_ = req.ParseMultipartForm(32 << 20)
	file, _, err := req.FormFile("file")
	assert.NoError(t, err)
	defer file.Close()

	reader, err := GetFileReader(file)
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	// Read content
	result, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, content, result)
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Bytes", 100, "100 B"},
		{"Kilobytes", 2048, "2.00 KB"},
		{"Megabytes", 5 * 1024 * 1024, "5.00 MB"},
		{"Gigabytes", 3 * 1024 * 1024 * 1024, "3.00 GB"},
		{"Large GB", 15 * 1024 * 1024 * 1024, "15.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileUploadHandler_ValidateFileSize(t *testing.T) {
	cfg := &config.Config{
		MaxUploadSize: 10 * 1024 * 1024, // 10MB
	}

	handler := &FileUploadHandler{
		config: cfg,
	}

	tests := []struct {
		name      string
		size      int64
		shouldErr bool
	}{
		{"Valid small file", 1024, false},
		{"Valid medium file", 5 * 1024 * 1024, false},
		{"Valid large file at limit", 10 * 1024 * 1024, false},
		{"Invalid too large file", 11 * 1024 * 1024, true},
		{"Invalid zero size", 0, true},
		{"Invalid negative size", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateFileSize(tt.size)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileUploadHandler_ErrorResponse_JSON(t *testing.T) {
	cfg := &config.Config{}
	handler := &FileUploadHandler{
		config: cfg,
	}

	// Create mock request
	req := httptest.NewRequest("POST", "/api/files/upload", nil)
	rec := httptest.NewRecorder()

	// Create RequestEvent (simplified, without full PocketBase setup)
	c := &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	details := map[string]interface{}{
		"max_size": 100,
	}

	err := handler.errorResponse(c, false, http.StatusBadRequest, "FILE_TOO_LARGE", "File is too large", details)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "FILE_TOO_LARGE")
	assert.Contains(t, rec.Body.String(), "File is too large")
}

func TestFileUploadHandler_ErrorResponse_HTMX(t *testing.T) {
	cfg := &config.Config{}
	handler := &FileUploadHandler{
		config: cfg,
	}

	// Create mock request with HTMX header
	req := httptest.NewRequest("POST", "/api/files/upload", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	c := &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	err := handler.errorResponse(c, true, http.StatusBadRequest, "FILE_TOO_LARGE", "File is too large", nil)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "File is too large")
	assert.Contains(t, rec.Body.String(), "FILE_TOO_LARGE")
	// Should contain HTML
	assert.Contains(t, rec.Body.String(), "<div")
}

func TestFileUploadHandler_SuccessResponse_JSON(t *testing.T) {
	cfg := &config.Config{}
	handler := &FileUploadHandler{
		config: cfg,
	}

	req := httptest.NewRequest("POST", "/api/files/upload", nil)
	rec := httptest.NewRecorder()

	c := &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	fileInfo := &FileInfo{
		ID:       "test-id",
		Name:     "test.txt",
		Size:     1024,
		MimeType: "text/plain",
		Path:     "/",
		Created:  time.Now(),
	}

	err := handler.successResponse(c, false, fileInfo)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "test-id")
	assert.Contains(t, rec.Body.String(), "test.txt")
	assert.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestFileUploadHandler_SuccessResponse_HTMX(t *testing.T) {
	cfg := &config.Config{}
	handler := &FileUploadHandler{
		config: cfg,
	}

	req := httptest.NewRequest("POST", "/api/files/upload", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	c := &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	fileInfo := &FileInfo{
		ID:       "test-id",
		Name:     "test.txt",
		Size:     1024,
		MimeType: "text/plain",
		Path:     "/",
		Created:  time.Now(),
	}

	err := handler.successResponse(c, true, fileInfo)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "test.txt")
	assert.Contains(t, rec.Body.String(), "file-item")
	// Should contain HTML
	assert.Contains(t, rec.Body.String(), "<div")
	// Should have HX-Trigger header
	assert.Equal(t, "fileUploaded", rec.Header().Get("HX-Trigger"))
}

func TestModels_SanitizeFilename(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		shouldErr bool
	}{
		{"Normal filename", "document.pdf", "document.pdf", false},
		{"Filename with spaces", "my document.txt", "my document.txt", false},
		{"Path traversal", "../../../etc/passwd", "passwd", false},
		{"Windows path", "C:\\Users\\file.txt", "file.txt", false},
		{"Null byte", "file\x00.txt", "", true},
		{"Control characters", "file\n.txt", "", true},
		{"Just dots", "..", "", true},
		{"Empty", "", "", true},
		{"Very long name", strings.Repeat("a", 300), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := models.SanitizeFilename(tt.input)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestModels_ValidateFileSize(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		maxSize   int64
		shouldErr bool
	}{
		{"Valid size", 1024, 2048, false},
		{"At max size", 2048, 2048, false},
		{"Exceeds max", 3000, 2048, true},
		{"Zero size", 0, 2048, true},
		{"Negative size", -1, 2048, true},
		{"No limit", 1000000, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateFileSize(tt.size, tt.maxSize)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMockS3Service_UploadFile(t *testing.T) {
	mockS3 := new(MockS3Service)

	content := []byte("test content")
	reader := bytes.NewReader(content)

	mockS3.On("UploadFile", "test-key", reader, int64(len(content)), "text/plain").Return(nil)

	err := mockS3.UploadFile("test-key", reader, int64(len(content)), "text/plain")
	assert.NoError(t, err)

	mockS3.AssertExpectations(t)
}

func TestMockS3Service_UploadFile_Error(t *testing.T) {
	mockS3 := new(MockS3Service)

	content := []byte("test content")
	reader := bytes.NewReader(content)

	mockS3.On("UploadFile", "test-key", reader, int64(len(content)), "text/plain").
		Return(errors.New("upload failed"))

	err := mockS3.UploadFile("test-key", reader, int64(len(content)), "text/plain")
	assert.Error(t, err)
	assert.Equal(t, "upload failed", err.Error())

	mockS3.AssertExpectations(t)
}

func TestMockPermissionService_CanUploadFile(t *testing.T) {
	mockPerms := new(MockPermissionService)

	mockPerms.On("CanUploadFile", "user123", "dir456", "").Return(true, nil)

	canUpload, err := mockPerms.CanUploadFile("user123", "dir456", "")
	assert.NoError(t, err)
	assert.True(t, canUpload)

	mockPerms.AssertExpectations(t)
}

func TestMockPermissionService_CanUploadFile_Denied(t *testing.T) {
	mockPerms := new(MockPermissionService)

	mockPerms.On("CanUploadFile", "user123", "dir456", "").Return(false, nil)

	canUpload, err := mockPerms.CanUploadFile("user123", "dir456", "")
	assert.NoError(t, err)
	assert.False(t, canUpload)

	mockPerms.AssertExpectations(t)
}

func TestMockPermissionService_CanUploadSize(t *testing.T) {
	mockPerms := new(MockPermissionService)

	mockPerms.On("CanUploadSize", "user123", int64(1024)).Return(true, nil)

	canUpload, err := mockPerms.CanUploadSize("user123", 1024)
	assert.NoError(t, err)
	assert.True(t, canUpload)

	mockPerms.AssertExpectations(t)
}

func TestMockPermissionService_CanUploadSize_QuotaExceeded(t *testing.T) {
	mockPerms := new(MockPermissionService)

	mockPerms.On("CanUploadSize", "user123", int64(1000000)).Return(false, nil)

	canUpload, err := mockPerms.CanUploadSize("user123", 1000000)
	assert.NoError(t, err)
	assert.False(t, canUpload)

	mockPerms.AssertExpectations(t)
}

func TestMockPermissionService_GetUserQuota(t *testing.T) {
	mockPerms := new(MockPermissionService)

	expectedQuota := &services.QuotaInfo{
		TotalQuota: 10 * 1024 * 1024 * 1024,
		UsedQuota:  5 * 1024 * 1024 * 1024,
		Available:  5 * 1024 * 1024 * 1024,
		Percentage: 50.0,
	}

	mockPerms.On("GetUserQuota", "user123").Return(expectedQuota, nil)

	quota, err := mockPerms.GetUserQuota("user123")
	assert.NoError(t, err)
	assert.NotNil(t, quota)
	assert.Equal(t, expectedQuota.TotalQuota, quota.TotalQuota)
	assert.Equal(t, expectedQuota.UsedQuota, quota.UsedQuota)
	assert.Equal(t, 50.0, quota.Percentage)

	mockPerms.AssertExpectations(t)
}
