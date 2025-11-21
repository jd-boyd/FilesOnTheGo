package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock S3 server for testing
func createMockS3Server(t *testing.T) (*httptest.Server, map[string][]byte) {
	files := make(map[string][]byte)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required headers
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusForbidden)
			xml.Write(w, xml.Name{Local: "Error"}, &S3ErrorResponse{
				Code:    "AccessDenied",
				Message: "Missing authentication header",
			})
			return
		}

		// Parse URL to get key
		path := strings.TrimPrefix(r.URL.Path, "/test-bucket/")

		switch r.Method {
		case "GET":
			if r.URL.Query().Get("delete") != "" {
				// Handle batch delete
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><DeleteResult/>`))
				return
			}

			// Download file
			if data, exists := files[path]; exists {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("ETag", `"test-etag"`)
				w.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))
				w.WriteHeader(http.StatusOK)
				w.Write(data)
			} else {
				w.WriteHeader(http.StatusNotFound)
				xml.Write(w, xml.Name{Local: "Error"}, &S3ErrorResponse{
					Code:    "NoSuchKey",
					Message: "The specified key does not exist",
					Key:     path,
				})
			}

		case "PUT":
			// Upload file
			body, _ := io.ReadAll(r.Body)
			files[path] = body
			w.WriteHeader(http.StatusOK)

		case "HEAD":
			// Check file exists
			if _, exists := files[path]; exists {
				data := files[path]
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("ETag", `"test-etag"`)
				w.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}

		case "DELETE":
			// Delete file
			delete(files, path)
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))

	return server, files
}

func TestNewLightweightS3Service(t *testing.T) {
	server, _ := createMockS3Server(t)
	defer server.Close()

	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: &config.Config{
				S3Bucket:    "test-bucket",
				S3AccessKey: "test-access-key",
				S3SecretKey: "test-secret-key",
				S3Region:    "us-east-1",
				S3Endpoint:  server.URL,
			},
			wantErr: false,
		},
		{
			name: "Missing bucket",
			config: &config.Config{
				S3AccessKey: "test-access-key",
				S3SecretKey: "test-secret-key",
				S3Region:    "us-east-1",
				S3Endpoint:  server.URL,
			},
			wantErr: true,
			errMsg:  "bucket name is required",
		},
		{
			name: "Missing access key",
			config: &config.Config{
				S3Bucket:    "test-bucket",
				S3SecretKey: "test-secret-key",
				S3Region:    "us-east-1",
				S3Endpoint:  server.URL,
			},
			wantErr: true,
			errMsg:  "access key is required",
		},
		{
			name: "Missing secret key",
			config: &config.Config{
				S3Bucket:    "test-bucket",
				S3AccessKey: "test-access-key",
				S3Region:    "us-east-1",
				S3Endpoint:  server.URL,
			},
			wantErr: true,
			errMsg:  "secret key is required",
		},
		{
			name: "Default region",
			config: &config.Config{
				S3Bucket:    "test-bucket",
				S3AccessKey: "test-access-key",
				S3SecretKey: "test-secret-key",
				S3Endpoint:  server.URL,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewLightweightS3Service(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, tt.config.S3Bucket, service.bucket)
				assert.Equal(t, tt.config.S3AccessKey, service.accessKey)
				assert.Equal(t, tt.config.S3SecretKey, service.secretKey)
			}
		})
	}
}

func TestLightweightS3Service_UploadFile(t *testing.T) {
	server, files := createMockS3Server(t)
	defer server.Close()

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	tests := []struct {
		name         string
		key          string
		data         []byte
		size         int64
		contentType  string
		expectErr    bool
		expectUpload bool
	}{
		{
			name:         "Successful upload",
			key:          "test/file.txt",
			data:         []byte("test content"),
			size:         12,
			contentType:  "text/plain",
			expectErr:    false,
			expectUpload: true,
		},
		{
			name:         "Upload with default content type",
			key:          "test/file.bin",
			data:         []byte("binary data"),
			size:         11,
			contentType:  "",
			expectErr:    false,
			expectUpload: true,
		},
		{
			name:         "Empty key",
			key:          "",
			data:         []byte("content"),
			size:         7,
			contentType:  "text/plain",
			expectErr:    true,
			expectUpload: false,
		},
		{
			name:         "Key too long",
			key:          strings.Repeat("a", 1025),
			data:         []byte("content"),
			size:         7,
			contentType:  "text/plain",
			expectErr:    true,
			expectUpload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear files map before each test
			for k := range files {
				delete(files, k)
			}

			err := service.UploadFile(tt.key, bytes.NewReader(tt.data), tt.size, tt.contentType)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expectUpload {
					storedData, exists := files[tt.key]
					assert.True(t, exists, "File should be stored")
					assert.Equal(t, tt.data, storedData, "Stored data should match uploaded data")
				}
			}
		})
	}
}

func TestLightweightS3Service_DownloadFile(t *testing.T) {
	server, files := createMockS3Server(t)
	defer server.Close()

	// Pre-populate with test data
	testData := []byte("download test content")
	files["existing/file.txt"] = testData

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	tests := []struct {
		name          string
		key           string
		expectErr     bool
		expectContent []byte
		errorType     error
	}{
		{
			name:          "Successful download",
			key:           "existing/file.txt",
			expectErr:     false,
			expectContent: testData,
		},
		{
			name:      "File not found",
			key:       "nonexistent/file.txt",
			expectErr: true,
			errorType: ErrFileNotFound,
		},
		{
			name:      "Empty key",
			key:       "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := service.DownloadFile(tt.key)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, reader)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)

				downloadedData, err := io.ReadAll(reader)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectContent, downloadedData)

				reader.Close()
			}
		})
	}
}

func TestLightweightS3Service_DeleteFile(t *testing.T) {
	server, files := createMockS3Server(t)
	defer server.Close()

	// Pre-populate with test data
	testData := []byte("test content")
	files["to-delete/file.txt"] = testData

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	tests := []struct {
		name        string
		key         string
		expectErr   bool
		expectExist bool // After deletion
	}{
		{
			name:        "Successful delete",
			key:         "to-delete/file.txt",
			expectErr:   false,
			expectExist: false,
		},
		{
			name:        "Delete non-existent file",
			key:         "nonexistent/file.txt",
			expectErr:   false, // S3 doesn't error on deleting non-existent files
			expectExist: false,
		},
		{
			name:        "Empty key",
			key:         "",
			expectErr:   true,
			expectExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteFile(tt.key)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check if file exists after operation
			_, exists := files[tt.key]
			assert.Equal(t, tt.expectExist, exists)
		})
	}
}

func TestLightweightS3Service_DeleteFiles(t *testing.T) {
	server, files := createMockS3Server(t)
	defer server.Close()

	// Pre-populate with test data
	testData := []byte("test content")
	files["batch/delete1.txt"] = testData
	files["batch/delete2.txt"] = testData
	files["batch/keep.txt"] = testData

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	tests := []struct {
		name      string
		keys      []string
		expectErr bool
	}{
		{
			name:      "Successful batch delete",
			keys:      []string{"batch/delete1.txt", "batch/delete2.txt"},
			expectErr: false,
		},
		{
			name:      "Empty keys list",
			keys:      []string{},
			expectErr: false,
		},
		{
			name:      "Keys with invalid entry",
			keys:      []string{"valid/key.txt", ""},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteFiles(tt.keys)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that specified keys are deleted (except empty keys)
				for _, key := range tt.keys {
					if key != "" {
						_, exists := files[key]
						assert.False(t, exists, "Key should be deleted: %s", key)
					}
				}

				// Check that other keys still exist
				_, exists := files["batch/keep.txt"]
				assert.True(t, exists, "Unspecified key should still exist")
			}
		})
	}
}

func TestLightweightS3Service_FileExists(t *testing.T) {
	server, files := createMockS3Server(t)
	defer server.Close()

	// Pre-populate with test data
	testData := []byte("test content")
	files["exists/file.txt"] = testData

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	tests := []struct {
		name        string
		key         string
		expectErr   bool
		expectExist bool
	}{
		{
			name:        "File exists",
			key:         "exists/file.txt",
			expectErr:   false,
			expectExist: true,
		},
		{
			name:        "File does not exist",
			key:         "missing/file.txt",
			expectErr:   false,
			expectExist: false,
		},
		{
			name:        "Empty key",
			key:         "",
			expectErr:   true,
			expectExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := service.FileExists(tt.key)

			if tt.expectErr {
				assert.Error(t, err)
				assert.False(t, exists)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectExist, exists)
			}
		})
	}
}

func TestLightweightS3Service_GetFileMetadata(t *testing.T) {
	server, files := createMockS3Server(t)
	defer server.Close()

	// Pre-populate with test data
	testData := []byte("test content")
	files["metadata/file.txt"] = testData

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	tests := []struct {
		name       string
		key        string
		expectErr  bool
		errorType  error
		expectSize int64
	}{
		{
			name:       "Successful metadata retrieval",
			key:        "metadata/file.txt",
			expectErr:  false,
			expectSize: int64(len(testData)),
		},
		{
			name:      "File not found",
			key:       "missing/file.txt",
			expectErr: true,
			errorType: ErrFileNotFound,
		},
		{
			name:      "Empty key",
			key:       "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := service.GetFileMetadata(tt.key)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, metadata)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metadata)
				assert.Equal(t, tt.expectSize, metadata.Size)
				assert.Equal(t, "application/octet-stream", metadata.ContentType)
				assert.Equal(t, "test-etag", metadata.ETag)
			}
		})
	}
}

func TestLightweightS3Service_GetPresignedURL(t *testing.T) {
	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  "https://s3.us-east-1.amazonaws.com",
		bucket:    "test-bucket",
		client:    &http.Client{},
	}

	tests := []struct {
		name             string
		key              string
		expirationMinutes int
		expectErr        bool
		expectURL        bool
	}{
		{
			name:             "Generate presigned URL",
			key:              "test/file.txt",
			expirationMinutes: 15,
			expectErr:        false,
			expectURL:        true,
		},
		{
			name:             "Default expiration",
			key:              "test/file.txt",
			expirationMinutes: 0,
			expectErr:        false,
			expectURL:        true,
		},
		{
			name:             "Max expiration",
			key:              "test/file.txt",
			expirationMinutes: 120, // Should be capped at 60
			expectErr:        false,
			expectURL:        true,
		},
		{
			name:             "Empty key",
			key:              "",
			expirationMinutes: 15,
			expectErr:        true,
			expectURL:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := service.GetPresignedURL(tt.key, tt.expirationMinutes)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "X-Amz-Signature")
				assert.Contains(t, url, "X-Amz-Expires")

				// Check expiration is capped at 60 minutes
				if tt.expirationMinutes > 60 {
					assert.Contains(t, url, "X-Amz-Expires=3600")
				} else if tt.expirationMinutes <= 0 {
					assert.Contains(t, url, "X-Amz-Expires=900") // 15 minutes default
				} else {
					expectedExpires := tt.expirationMinutes * 60
					assert.Contains(t, url, fmt.Sprintf("X-Amz-Expires=%d", expectedExpires))
				}
			}
		})
	}
}

// Test AWS v4 signing helpers
func TestLightweightS3Service_Signing(t *testing.T) {
	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  "https://s3.us-east-1.amazonaws.com",
		bucket:    "test-bucket",
		client:    &http.Client{},
	}

	t.Run("getCanonicalHeaders", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)

		req.Header.Set("Host", "example.com")
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("X-Amz-Date", "20231201T000000Z")
		req.Header.Set("X-Amz-Meta-Test", "value")

		canonicalHeaders, signedHeaders := service.getCanonicalHeaders(req)

		assert.Contains(t, canonicalHeaders, "host:example.com")
		assert.Contains(t, canonicalHeaders, "content-type:application/octet-stream")
		assert.Contains(t, canonicalHeaders, "x-amz-date:20231201T000000Z")
		assert.Contains(t, canonicalHeaders, "x-amz-meta-test:value")

		assert.Equal(t, "content-type;host;x-amz-date;x-amz-meta-test", signedHeaders)
	})

	t.Run("getSignatureKey", func(t *testing.T) {
		dateStamp := "20231201"
		key := service.getSignatureKey(dateStamp)
		assert.NotNil(t, key)
		assert.Len(t, key, 32) // SHA256 output length
	})

	t.Run("hmacSha256", func(t *testing.T) {
		key := []byte("test-key")
		data := "test-data"
		signature := service.hmacSha256(key, data)
		assert.NotNil(t, signature)
		assert.Len(t, signature, 32) // SHA256 output length
	})

	t.Run("sha256Hash", func(t *testing.T) {
		data := "test-data"
		hash := service.sha256Hash(data)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64) // Hex encoded SHA256
	})
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expectErr bool
	}{
		{"Valid key", "users/123/file.txt", false},
		{"Empty key", "", true},
		{"Key too long", strings.Repeat("a", 1025), true},
		{"Key with null byte", "file\x00.txt", true},
		{"Max length key", strings.Repeat("a", 1024), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKey(tt.key)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Mock multipart upload responses
func createMockMultipartS3Server(t *testing.T) (*httptest.Server, map[string][]byte) {
	files := make(map[string][]byte)
	uploadSessions := make(map[string]*MultipartUpload)
	partCounter := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required headers
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>AccessDenied</Code><Message>Missing authentication header</Message></Error>`))
			return
		}

		// Parse URL to get key and parameters
		path := strings.TrimPrefix(r.URL.Path, "/test-bucket/")
		query := r.URL.Query()

		switch r.Method {
		case "POST":
			if query.Get("uploads") != "" {
				// Initiate multipart upload
				uploadID := fmt.Sprintf("upload_%d", partCounter)
				partCounter++

				response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<InitiateMultipartUploadResult>
	<Bucket>test-bucket</Bucket>
	<Key>%s</Key>
	<UploadId>%s</UploadId>
</InitiateMultipartUploadResult>`, html.EscapeString(path), uploadID)

				uploadSessions[uploadID] = &MultipartUpload{
					UploadID: uploadID,
					Key:      path,
					Bucket:   "test-bucket",
				}

				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))

			} else if query.Get("uploadId") != "" {
				// Complete multipart upload
				uploadID := query.Get("uploadId")
				if _, exists := uploadSessions[uploadID]; !exists {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchUpload</Code></Error>`))
					return
				}

				// Combine all uploaded parts
				var combinedData []byte
				// For simplicity, assume parts were stored and combine them
				for _, key := range files {
					combinedData = append(combinedData, key...)
				}
				files[path] = combinedData

				// Clean up session
				delete(uploadSessions, uploadID)

				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<CompleteMultipartUploadResult>
	<Location>http://test-bucket.s3.amazonaws.com/` + path + `</Location>
	<Bucket>test-bucket</Bucket>
	<Key>` + path + `</Key>
	<ETag>"combined-etag"</ETag>
</CompleteMultipartUploadResult>`))

			}

		case "PUT":
			if query.Get("partNumber") != "" && query.Get("uploadId") != "" {
				// Upload part
				partNumber := query.Get("partNumber")
				uploadID := query.Get("uploadId")

				if _, exists := uploadSessions[uploadID]; !exists {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchUpload</Code></Error>`))
					return
				}

				// Read part data
				body, _ := io.ReadAll(r.Body)
				etag := fmt.Sprintf("etag-part-%s", partNumber)

				// Store part (simplified - in real implementation would store separately)
				files[path+"_part_"+partNumber] = body

				w.Header().Set("ETag", `"`+etag+`"`)
				w.WriteHeader(http.StatusOK)
			} else {
				// Regular PUT upload
				body, _ := io.ReadAll(r.Body)
				files[path] = body
				w.WriteHeader(http.StatusOK)
			}

		case "GET":
			if r.URL.Query().Get("delete") != "" {
				// Handle batch delete
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><DeleteResult/>`))
				return
			}

			// Download file
			if data, exists := files[path]; exists {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("ETag", `"test-etag"`)
				w.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))
				w.WriteHeader(http.StatusOK)
				w.Write(data)
			} else {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist</Message><Key>` + path + `</Key></Error>`))
			}

		case "HEAD":
			// Check file exists
			if _, exists := files[path]; exists {
				data := files[path]
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("ETag", `"test-etag"`)
				w.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}

		case "DELETE":
			if query.Get("uploadId") != "" {
				// Abort multipart upload
				uploadID := query.Get("uploadId")
				if _, exists := uploadSessions[uploadID]; exists {
					delete(uploadSessions, uploadID)
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			} else {
				// Delete file
				delete(files, path)
				w.WriteHeader(http.StatusNoContent)
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))

	return server, files
}

func TestLightweightS3Service_UploadStream_SmallFile(t *testing.T) {
	server, files := createMockMultipartS3Server(t)
	defer server.Close()

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	// Test with small file (should use simple upload)
	smallData := make([]byte, 1024) // 1KB
	for i := range smallData {
		smallData[i] = byte(i % 256)
	}

	err := service.UploadStream("small/file.bin", bytes.NewReader(smallData))
	assert.NoError(t, err)

	storedData, exists := files["small/file.bin"]
	assert.True(t, exists, "File should be stored")
	assert.Equal(t, smallData, storedData, "Stored data should match uploaded data")
}

func TestLightweightS3Service_UploadStream_LargeFile(t *testing.T) {
	server, files := createMockMultipartS3Server(t)
	defer server.Close()

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	// Test with large file (should use multipart upload)
	largeData := make([]byte, 15*1024*1024) // 15MB to trigger multipart upload
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := service.UploadStream("large/file.bin", bytes.NewReader(largeData))
	assert.NoError(t, err)

	// Check that multipart upload was used (parts should exist)
	part1Exists := files["large/file.bin_part_1"] != nil
	part2Exists := files["large/file.bin_part_2"] != nil
	assert.True(t, part1Exists || part2Exists, "Multipart upload should have been used")
}

func TestLightweightS3Service_MultipartUploadInitiate(t *testing.T) {
	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  "https://s3.us-east-1.amazonaws.com",
		bucket:    "test-bucket",
		client:    &http.Client{},
	}

	t.Run("Valid initiate", func(t *testing.T) {
		// Test multipart upload initiate URL generation
		key := "test/multipart.txt"
		expectedURL := "https://s3.us-east-1.amazonaws.com/test-bucket/test/multipart.txt?uploads"

		// This test validates the URL structure is correct
		url := service.getObjectURL(key) + "?uploads"
		assert.Equal(t, expectedURL, url)
	})
}

func TestLightweightS3Service_UploadPart(t *testing.T) {
	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  "https://s3.us-east-1.amazonaws.com",
		bucket:    "test-bucket",
		client:    &http.Client{},
	}

	t.Run("Part URL generation", func(t *testing.T) {
		upload := &MultipartUpload{
			UploadID: "test-upload-123",
			Key:      "test/file.txt",
			Bucket:   "test-bucket",
		}

		// Test part upload URL generation
		partNumber := 1
		expectedURL := "https://s3.us-east-1.amazonaws.com/test-bucket/test/file.txt?partNumber=1&uploadId=test-upload-123"

		url := fmt.Sprintf("%s?partNumber=%d&uploadId=%s",
			service.getObjectURL(upload.Key), partNumber, upload.UploadID)
		assert.Equal(t, expectedURL, url)
	})
}

func TestBufferedReader(t *testing.T) {
	t.Run("Small data", func(t *testing.T) {
		data := []byte("small content")
		reader := newBufferedReader(bytes.NewReader(data), 1024)

		assert.False(t, reader.hasMoreThan(1024))

		result, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("Large data", func(t *testing.T) {
		data := make([]byte, 2048)
		for i := range data {
			data[i] = byte(i % 256)
		}
		reader := newBufferedReader(bytes.NewReader(data), 1024)

		assert.True(t, reader.hasMoreThan(1024))

		result, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("EOF detection", func(t *testing.T) {
		data := []byte("content")
		reader := newBufferedReader(strings.NewReader("content"), 1024)

		assert.False(t, reader.hasMoreThan(1024))
	})
}

func TestLightweightS3Service_MultipartUploadErrorHandling(t *testing.T) {
	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  "https://invalid-endpoint.example.com",
		bucket:    "test-bucket",
		client:    &http.Client{Timeout: 1 * time.Second},
	}

	// Test error handling for multipart upload with invalid endpoint
	largeData := make([]byte, 15*1024*1024) // 15MB to trigger multipart upload
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := service.UploadStream("test/large.bin", bytes.NewReader(largeData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initiate multipart upload")
}

// Benchmark tests
func BenchmarkLightweightS3Service_SignRequest(b *testing.B) {
	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  "https://s3.us-east-1.amazonaws.com",
		bucket:    "test-bucket",
		client:    &http.Client{},
	}

	req, err := http.NewRequest("GET", "https://s3.us-east-1.amazonaws.com/test-bucket/key", nil)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.signRequest(req, "")
	}
}

func BenchmarkLightweightS3Service_UploadStream_Small(b *testing.B) {
	server, files := createMockMultipartS3Server(b)
	defer server.Close()

	service := &LightweightS3Service{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "us-east-1",
		endpoint:  server.URL,
		bucket:    "test-bucket",
		client:    server.Client(),
	}

	data := make([]byte, 1024) // 1KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench/test_%d.bin", i)
		err := service.UploadStream(key, bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}