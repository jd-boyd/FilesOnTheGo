package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockS3Client is a mock implementation of the S3 client for testing
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadBucketOutput), args.Error(1)
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectsOutput), args.Error(1)
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

// mockReadCloser is a simple ReadCloser for testing
type mockReadCloser struct {
	*bytes.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(data []byte) io.ReadCloser {
	return &mockReadCloser{bytes.NewReader(data)}
}

// Test NewS3Service
func TestNewS3Service_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "us-east-1",
		S3Bucket:    "test-bucket",
		S3AccessKey: "test-access-key",
		S3SecretKey: "test-secret-key",
		S3UseSSL:    false,
	}

	// Note: This test will fail without a real S3 connection
	// In a real scenario, you would use a test container or mock the HeadBucket call
	service, err := NewS3Service(cfg)

	// We expect an error because there's no real S3 server
	// But we're testing that the service initialization logic works
	if service != nil {
		assert.NotNil(t, service.client)
		assert.Equal(t, "test-bucket", service.bucket)
		assert.Equal(t, "us-east-1", service.region)
	} else {
		// If connection failed, that's expected in test environment
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrConnectionFailed)
	}
}

func TestNewS3Service_MissingBucket(t *testing.T) {
	cfg := &config.Config{
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "us-east-1",
		S3Bucket:    "", // Missing bucket
		S3AccessKey: "test-access-key",
		S3SecretKey: "test-secret-key",
	}

	service, err := NewS3Service(cfg)

	assert.Nil(t, service)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
	assert.Contains(t, err.Error(), "bucket name is required")
}

func TestNewS3Service_MissingAccessKey(t *testing.T) {
	cfg := &config.Config{
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "us-east-1",
		S3Bucket:    "test-bucket",
		S3AccessKey: "", // Missing access key
		S3SecretKey: "test-secret-key",
	}

	service, err := NewS3Service(cfg)

	assert.Nil(t, service)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
	assert.Contains(t, err.Error(), "access key is required")
}

func TestNewS3Service_MissingSecretKey(t *testing.T) {
	cfg := &config.Config{
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "us-east-1",
		S3Bucket:    "test-bucket",
		S3AccessKey: "test-access-key",
		S3SecretKey: "", // Missing secret key
	}

	service, err := NewS3Service(cfg)

	assert.Nil(t, service)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
	assert.Contains(t, err.Error(), "secret key is required")
}

func TestNewS3Service_MissingRegion(t *testing.T) {
	cfg := &config.Config{
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "", // Missing region
		S3Bucket:    "test-bucket",
		S3AccessKey: "test-access-key",
		S3SecretKey: "test-secret-key",
	}

	service, err := NewS3Service(cfg)

	assert.Nil(t, service)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
	assert.Contains(t, err.Error(), "region is required")
}

// Test UploadFile
func TestUploadFile_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	testData := []byte("test file content")
	reader := bytes.NewReader(testData)

	mockClient.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == "test-bucket" &&
			*input.Key == "test-key" &&
			*input.ContentType == "text/plain" &&
			*input.ContentLength == int64(len(testData))
	})).Return(&s3.PutObjectOutput{}, nil)

	err := service.UploadFile("test-key", reader, int64(len(testData)), "text/plain")

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestUploadFile_InvalidKey(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	reader := bytes.NewReader([]byte("test"))

	err := service.UploadFile("", reader, 4, "text/plain")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestUploadFile_DefaultContentType(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	testData := []byte("test")
	reader := bytes.NewReader(testData)

	mockClient.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.ContentType == "application/octet-stream"
	})).Return(&s3.PutObjectOutput{}, nil)

	err := service.UploadFile("test-key", reader, int64(len(testData)), "")

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestUploadFile_S3Error(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	reader := bytes.NewReader([]byte("test"))

	mockClient.On("PutObject", mock.Anything, mock.Anything).
		Return(nil, errors.New("S3 error"))

	err := service.UploadFile("test-key", reader, 4, "text/plain")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUploadFailed)
	mockClient.AssertExpectations(t)
}

// Test DownloadFile
func TestDownloadFile_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	testData := []byte("test file content")
	mockBody := newMockReadCloser(testData)

	mockClient.On("GetObject", mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Bucket == "test-bucket" && *input.Key == "test-key"
	})).Return(&s3.GetObjectOutput{
		Body: mockBody,
	}, nil)

	result, err := service.DownloadFile("test-key")

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Read the content
	content, err := io.ReadAll(result)
	assert.NoError(t, err)
	assert.Equal(t, testData, content)

	mockClient.AssertExpectations(t)
}

func TestDownloadFile_NotFound(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	mockClient.On("GetObject", mock.Anything, mock.Anything).
		Return(nil, &types.NoSuchKey{})

	result, err := service.DownloadFile("nonexistent-key")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrFileNotFound)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}

func TestDownloadFile_InvalidKey(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	result, err := service.DownloadFile("")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
	assert.Nil(t, result)
}

// Test DeleteFile
func TestDeleteFile_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	mockClient.On("DeleteObject", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectInput) bool {
		return *input.Bucket == "test-bucket" && *input.Key == "test-key"
	})).Return(&s3.DeleteObjectOutput{}, nil)

	err := service.DeleteFile("test-key")

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeleteFile_InvalidKey(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	err := service.DeleteFile("")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestDeleteFile_S3Error(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	mockClient.On("DeleteObject", mock.Anything, mock.Anything).
		Return(nil, errors.New("S3 error"))

	err := service.DeleteFile("test-key")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeleteFailed)
	mockClient.AssertExpectations(t)
}

// Test DeleteFiles
func TestDeleteFiles_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	keys := []string{"key1", "key2", "key3"}

	mockClient.On("DeleteObjects", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectsInput) bool {
		return *input.Bucket == "test-bucket" && len(input.Delete.Objects) == 3
	})).Return(&s3.DeleteObjectsOutput{
		Deleted: []types.DeletedObject{
			{Key: aws.String("key1")},
			{Key: aws.String("key2")},
			{Key: aws.String("key3")},
		},
		Errors: []types.Error{},
	}, nil)

	err := service.DeleteFiles(keys)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeleteFiles_EmptyList(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	err := service.DeleteFiles([]string{})

	assert.NoError(t, err)
}

func TestDeleteFiles_PartialFailure(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	keys := []string{"key1", "key2", "key3"}

	mockClient.On("DeleteObjects", mock.Anything, mock.Anything).Return(&s3.DeleteObjectsOutput{
		Deleted: []types.DeletedObject{
			{Key: aws.String("key1")},
		},
		Errors: []types.Error{
			{Key: aws.String("key2"), Code: aws.String("AccessDenied")},
			{Key: aws.String("key3"), Code: aws.String("NoSuchKey")},
		},
	}, nil)

	err := service.DeleteFiles(keys)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeleteFailed)
	assert.Contains(t, err.Error(), "failed to delete 2 files")
	mockClient.AssertExpectations(t)
}

func TestDeleteFiles_InvalidKey(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	err := service.DeleteFiles([]string{"valid-key", ""})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

// Test FileExists
func TestFileExists_True(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	mockClient.On("HeadObject", mock.Anything, mock.MatchedBy(func(input *s3.HeadObjectInput) bool {
		return *input.Bucket == "test-bucket" && *input.Key == "test-key"
	})).Return(&s3.HeadObjectOutput{}, nil)

	exists, err := service.FileExists("test-key")

	assert.NoError(t, err)
	assert.True(t, exists)
	mockClient.AssertExpectations(t)
}

func TestFileExists_False(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	mockClient.On("HeadObject", mock.Anything, mock.Anything).
		Return(nil, &types.NotFound{})

	exists, err := service.FileExists("nonexistent-key")

	assert.NoError(t, err)
	assert.False(t, exists)
	mockClient.AssertExpectations(t)
}

func TestFileExists_InvalidKey(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	exists, err := service.FileExists("")

	assert.Error(t, err)
	assert.False(t, exists)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

// Test GetFileMetadata
func TestGetFileMetadata_Success(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	now := time.Now()
	mockClient.On("HeadObject", mock.Anything, mock.MatchedBy(func(input *s3.HeadObjectInput) bool {
		return *input.Bucket == "test-bucket" && *input.Key == "test-key"
	})).Return(&s3.HeadObjectOutput{
		ContentLength: aws.Int64(1024),
		ContentType:   aws.String("text/plain"),
		LastModified:  &now,
		ETag:          aws.String("\"abc123\""),
	}, nil)

	metadata, err := service.GetFileMetadata("test-key")

	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, int64(1024), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
	assert.Equal(t, now, metadata.LastModified)
	assert.Equal(t, "\"abc123\"", metadata.ETag)
	mockClient.AssertExpectations(t)
}

func TestGetFileMetadata_NotFound(t *testing.T) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	mockClient.On("HeadObject", mock.Anything, mock.Anything).
		Return(nil, &types.NotFound{})

	metadata, err := service.GetFileMetadata("nonexistent-key")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrFileNotFound)
	assert.Nil(t, metadata)
	mockClient.AssertExpectations(t)
}

func TestGetFileMetadata_InvalidKey(t *testing.T) {
	service := &S3ServiceImpl{
		bucket: "test-bucket",
		region: "us-east-1",
	}

	metadata, err := service.GetFileMetadata("")

	assert.Error(t, err)
	assert.Nil(t, metadata)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

// Test GenerateS3Key and path traversal protection
func TestGenerateS3Key_NormalFilename(t *testing.T) {
	key := GenerateS3Key("user123", "file456", "document.txt")
	expected := "users/user123/file456/document.txt"
	assert.Equal(t, expected, key)
}

func TestGenerateS3Key_PathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		fileID   string
		filename string
		expected string
	}{
		{
			name:     "Simple path traversal",
			userID:   "user123",
			fileID:   "file456",
			filename: "../../etc/passwd",
			expected: "users/user123/file456/passwd",
		},
		{
			name:     "Multiple path traversal",
			userID:   "user123",
			fileID:   "file456",
			filename: "../../../etc/passwd",
			expected: "users/user123/file456/passwd",
		},
		{
			name:     "Windows path traversal",
			userID:   "user123",
			fileID:   "file456",
			filename: "..\\..\\windows\\system32\\config",
			expected: "users/user123/file456/config",
		},
		{
			name:     "Absolute path Unix",
			userID:   "user123",
			fileID:   "file456",
			filename: "/etc/passwd",
			expected: "users/user123/file456/passwd",
		},
		{
			name:     "Absolute path Windows",
			userID:   "user123",
			fileID:   "file456",
			filename: "C:\\Windows\\System32\\cmd.exe",
			expected: "users/user123/file456/cmd.exe",
		},
		{
			name:     "Normal subdirectory path",
			userID:   "user123",
			fileID:   "file456",
			filename: "documents/report.pdf",
			expected: "users/user123/file456/report.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateS3Key(tt.userID, tt.fileID, tt.filename)
			assert.Equal(t, tt.expected, result)
			// Ensure no traversal characters remain
			assert.NotContains(t, result, "..")
			assert.True(t, strings.HasPrefix(result, "users/"))
		})
	}
}

func TestSanitizeFilename_NullBytes(t *testing.T) {
	result := sanitizeFilename("file\x00.txt")
	assert.NotContains(t, result, "\x00")
	assert.Equal(t, "file.txt", result)
}

func TestSanitizeFilename_ControlCharacters(t *testing.T) {
	// Control characters should be removed
	result := sanitizeFilename("file\x01\x02\x03.txt")
	assert.Equal(t, "file.txt", result)
}

func TestSanitizeFilename_EmptyOrDots(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "unnamed_file"},
		{".", "unnamed_file"},
		{"..", "unnamed_file"},
		{"...", "..."},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSanitizeFilename_MaxLength(t *testing.T) {
	longName := strings.Repeat("a", 300)
	result := sanitizeFilename(longName)
	assert.LessOrEqual(t, len(result), 255)
}

func TestSanitizeFilename_UnicodeCharacters(t *testing.T) {
	result := sanitizeFilename("文档.txt")
	assert.Equal(t, "文档.txt", result)
}

// Test validateKey
func TestValidateKey_Valid(t *testing.T) {
	err := validateKey("valid/key/path.txt")
	assert.NoError(t, err)
}

func TestValidateKey_Empty(t *testing.T) {
	err := validateKey("")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestValidateKey_TooLong(t *testing.T) {
	longKey := strings.Repeat("a", 1025)
	err := validateKey(longKey)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestValidateKey_NullByte(t *testing.T) {
	err := validateKey("key\x00with\x00null")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

// Benchmark tests
func BenchmarkUploadFile_1MB(b *testing.B) {
	mockClient := new(MockS3Client)
	service := &S3ServiceImpl{
		client: mockClient,
		bucket: "test-bucket",
		region: "us-east-1",
	}

	// Create 1MB of data
	data := make([]byte, 1024*1024)

	mockClient.On("PutObject", mock.Anything, mock.Anything).
		Return(&s3.PutObjectOutput{}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_ = service.UploadFile("benchmark-key", reader, int64(len(data)), "application/octet-stream")
	}
}

func BenchmarkGenerateS3Key(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateS3Key("user123", "file456", "document.txt")
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeFilename("../../etc/passwd")
	}
}

func BenchmarkSanitizeFilename_LongName(b *testing.B) {
	longName := strings.Repeat("a", 300)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeFilename(longName)
	}
}
