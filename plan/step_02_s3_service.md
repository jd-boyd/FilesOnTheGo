# Step 02: S3 Service Implementation

## Overview
Implement the S3 service layer that handles all interactions with S3-compatible storage, including upload, download, deletion, and pre-signed URL generation.

## Dependencies
- Step 01: Project scaffolding (requires project structure and config)

## Duration Estimate
45 minutes

## Agent Prompt

You are implementing Step 02 of the FilesOnTheGo project. Your task is to create a comprehensive S3 service with full test coverage.

### Tasks

1. **Create services/s3_service.go**
   - Define `S3Service` interface with methods:
     - `UploadFile(key string, reader io.Reader, size int64, contentType string) error`
     - `UploadStream(key string, reader io.Reader) error`
     - `DownloadFile(key string) (io.ReadCloser, error)`
     - `DeleteFile(key string) error`
     - `DeleteFiles(keys []string) error` (batch delete)
     - `GetPresignedURL(key string, expirationMinutes int) (string, error)`
     - `FileExists(key string) (bool, error)`
     - `GetFileMetadata(key string) (*FileMetadata, error)`

   - Define `FileMetadata` struct:
     - Size int64
     - ContentType string
     - LastModified time.Time
     - ETag string

   - Implement `S3ServiceImpl` struct with:
     - S3 client
     - Bucket name
     - Region
     - Configuration

2. **Implement Constructor**
   - `NewS3Service(config *config.S3Config) (*S3ServiceImpl, error)`
   - Validate configuration
   - Initialize AWS SDK v2 S3 client
   - Test connection to S3
   - Return error if connection fails

3. **Implement Upload Methods**
   - `UploadFile`: Direct upload with known size
     - Use PutObject API
     - Set Content-Type
     - Add metadata (uploaded_at timestamp)
     - Handle errors appropriately

   - `UploadStream`: Streaming upload for large files
     - Use S3 Upload Manager for multipart uploads
     - Automatically chunk large files
     - Handle errors and cleanup on failure

4. **Implement Download Methods**
   - `DownloadFile`: Stream file download
     - Return io.ReadCloser for streaming
     - Handle 404 errors gracefully

   - `GetPresignedURL`: Generate time-limited URLs
     - Default to 15 minutes if not specified
     - Maximum 60 minutes
     - Use AWS presigner

5. **Implement Delete Methods**
   - `DeleteFile`: Single file deletion
     - Handle non-existent files gracefully

   - `DeleteFiles`: Batch deletion
     - Use DeleteObjects API for efficiency
     - Return partial errors if some deletions fail

6. **Implement Utility Methods**
   - `FileExists`: Check if file exists without downloading
     - Use HeadObject API

   - `GetFileMetadata`: Retrieve file metadata
     - Use HeadObject API
     - Return FileMetadata struct

7. **Generate S3 Keys**
   - Implement helper function `GenerateS3Key(userID, fileID, filename string) string`
   - Format: `users/{userID}/{fileID}/{filename}`
   - Sanitize filename to prevent path traversal
   - Document the key structure

8. **Error Handling**
   - Define custom error types:
     - `ErrFileNotFound`
     - `ErrUploadFailed`
     - `ErrDeleteFailed`
     - `ErrInvalidKey`
   - Wrap AWS SDK errors with context
   - Use structured logging for all operations

9. **Write Comprehensive Tests (services/s3_service_test.go)**

   **Unit Tests with Mocking:**
   - Mock S3 client using testify/mock
   - Test all methods with mocked responses
   - Test error conditions
   - Test timeout scenarios

   **Integration Tests (if MinIO available):**
   - Set up test bucket
   - Test actual upload/download cycle
   - Test pre-signed URLs
   - Test file deletion
   - Clean up after tests

   **Security Tests:**
   - Test path traversal prevention in key generation
   - Test that pre-signed URLs expire
   - Test invalid bucket access

   **Performance Benchmarks:**
   - Benchmark upload for 1MB, 10MB files
   - Benchmark concurrent uploads

   **Test Coverage Requirements:**
   - Minimum 80% overall
   - 100% coverage for key generation and sanitization

### Success Criteria

- [ ] All S3Service methods implemented
- [ ] AWS SDK v2 properly configured
- [ ] Streaming uploads work for large files
- [ ] Pre-signed URLs generated correctly
- [ ] Error handling is comprehensive
- [ ] All tests pass (`go test ./services/...`)
- [ ] Test coverage >= 80%
- [ ] Code follows CLAUDE.md guidelines
- [ ] Path traversal protection implemented
- [ ] Logging added to all operations

### Testing Commands

```bash
# Run S3 service tests
go test ./services/ -v

# Run with coverage
go test ./services/ -cover

# Run benchmarks
go test ./services/ -bench=. -benchmem

# Test with race detector
go test ./services/ -race
```

### Example Test Structure

```go
func TestS3Service_UploadFile_Success(t *testing.T) {
    mockS3 := new(MockS3Client)
    service := &S3ServiceImpl{client: mockS3, bucket: "test-bucket"}

    mockS3.On("PutObject", mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

    err := service.UploadFile("test-key", bytes.NewReader([]byte("data")), 4, "text/plain")

    assert.NoError(t, err)
    mockS3.AssertExpectations(t)
}

func TestGenerateS3Key_PreventPathTraversal(t *testing.T) {
    tests := []struct {
        filename string
        expected string
    }{
        {"normal.txt", "users/user123/file456/normal.txt"},
        {"../../etc/passwd", "users/user123/file456/passwd"},
        {"../../../etc/passwd", "users/user123/file456/passwd"},
    }

    for _, tt := range tests {
        result := GenerateS3Key("user123", "file456", tt.filename)
        assert.Equal(t, tt.expected, result)
    }
}
```

### References

- DESIGN.md: S3 Integration section
- CLAUDE.md: Testing Requirements and Security Guidelines
- AWS SDK Go v2 docs: https://aws.github.io/aws-sdk-go-v2/docs/

### Notes

- Use AWS SDK v2 (not v1)
- Implement connection pooling for performance
- Add retry logic with exponential backoff
- Log all S3 operations with request IDs
- Consider rate limiting for production
- Document S3 bucket permissions required
