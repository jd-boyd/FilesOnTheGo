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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockS3Service is a mock implementation of S3Service
type MockS3Service struct {
	mock.Mock
}

func (m *MockS3Service) UploadFile(key string, reader io.Reader, size int64, contentType string) error {
	args := m.Called(key, reader, size, contentType)
	return args.Error(0)
}

func (m *MockS3Service) UploadStream(key string, reader io.Reader) error {
	args := m.Called(key, reader)
	return args.Error(0)
}

func (m *MockS3Service) DownloadFile(key string) (io.ReadCloser, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockS3Service) DeleteFile(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockS3Service) DeleteFiles(keys []string) error {
	args := m.Called(keys)
	return args.Error(0)
}

func (m *MockS3Service) GetPresignedURL(key string, expirationMinutes int) (string, error) {
	args := m.Called(key, expirationMinutes)
	return args.String(0), args.Error(1)
}

func (m *MockS3Service) FileExists(key string) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *MockS3Service) GetFileMetadata(key string) (*services.FileMetadata, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.FileMetadata), args.Error(1)
}

// MockPermissionService is a mock implementation of PermissionService
type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) CanReadFile(userID, fileID, shareToken string) (bool, error) {
	args := m.Called(userID, fileID, shareToken)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanUploadFile(userID, directoryID, shareToken string) (bool, error) {
	args := m.Called(userID, directoryID, shareToken)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanDeleteFile(userID, fileID string) (bool, error) {
	args := m.Called(userID, fileID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanMoveFile(userID, fileID, targetDirID string) (bool, error) {
	args := m.Called(userID, fileID, targetDirID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanReadDirectory(userID, directoryID, shareToken string) (bool, error) {
	args := m.Called(userID, directoryID, shareToken)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanCreateDirectory(userID, parentDirID string) (bool, error) {
	args := m.Called(userID, parentDirID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanDeleteDirectory(userID, directoryID string) (bool, error) {
	args := m.Called(userID, directoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanCreateShare(userID, resourceID, resourceType string) (bool, error) {
	args := m.Called(userID, resourceID, resourceType)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanRevokeShare(userID, shareID string) (bool, error) {
	args := m.Called(userID, shareID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) ValidateShareToken(shareToken, password string) (*services.SharePermissions, error) {
	args := m.Called(shareToken, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.SharePermissions), args.Error(1)
}

func (m *MockPermissionService) CanUploadSize(userID string, fileSize int64) (bool, error) {
	args := m.Called(userID, fileSize)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) GetUserQuota(userID string) (*services.QuotaInfo, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.QuotaInfo), args.Error(1)
}

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

	_ = req.ParseMultipartForm(32 << 20)
	file, header, err := req.FormFile("file")
	assert.NoError(t, err)
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
		Request:  req,
		Response: rec,
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
		Request:  req,
		Response: rec,
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
		Request:  req,
		Response: rec,
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
		Request:  req,
		Response: rec,
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
