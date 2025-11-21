package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/rs/zerolog/log"
)

// Custom error types for S3 operations
var (
	ErrFileNotFound  = errors.New("file not found in S3")
	ErrUploadFailed  = errors.New("failed to upload file to S3")
	ErrDeleteFailed  = errors.New("failed to delete file from S3")
	ErrInvalidKey    = errors.New("invalid S3 key")
	ErrInvalidConfig = errors.New("invalid S3 configuration")
	ErrConnectionFailed = errors.New("failed to connect to S3")
)

// FileMetadata represents metadata about a file stored in S3
type FileMetadata struct {
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// S3Service defines the interface for S3 operations
type S3Service interface {
	// UploadFile uploads a file with known size to S3
	UploadFile(key string, reader io.Reader, size int64, contentType string) error

	// UploadStream uploads a file using streaming (multipart upload for large files)
	UploadStream(key string, reader io.Reader) error

	// DownloadFile downloads a file from S3 as a stream
	DownloadFile(key string) (io.ReadCloser, error)

	// DeleteFile deletes a single file from S3
	DeleteFile(key string) error

	// DeleteFiles deletes multiple files from S3 (batch operation)
	DeleteFiles(keys []string) error

	// GetPresignedURL generates a time-limited presigned URL for file access
	GetPresignedURL(key string, expirationMinutes int) (string, error)

	// FileExists checks if a file exists in S3
	FileExists(key string) (bool, error)

	// GetFileMetadata retrieves metadata about a file
	GetFileMetadata(key string) (*FileMetadata, error)
}

// S3ServiceImpl implements the S3Service interface
type S3ServiceImpl struct {
	client   *s3.Client
	bucket   string
	region   string
	endpoint string
}

// NewS3Service creates a new S3 service instance and validates the connection
func NewS3Service(cfg *config.Config) (*S3ServiceImpl, error) {
	// Validate configuration
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("%w: bucket name is required", ErrInvalidConfig)
	}
	if cfg.S3AccessKey == "" {
		return nil, fmt.Errorf("%w: access key is required", ErrInvalidConfig)
	}
	if cfg.S3SecretKey == "" {
		return nil, fmt.Errorf("%w: secret key is required", ErrInvalidConfig)
	}
	if cfg.S3Region == "" {
		return nil, fmt.Errorf("%w: region is required", ErrInvalidConfig)
	}

	// Create AWS credentials
	creds := credentials.NewStaticCredentialsProvider(
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		"",
	)

	// Create custom endpoint resolver for S3-compatible services (like MinIO)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.S3Endpoint != "" {
			return aws.Endpoint{
				URL:           cfg.S3Endpoint,
				SigningRegion: cfg.S3Region,
			}, nil
		}
		// Return default AWS endpoint
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Configure AWS SDK
	awsCfg := aws.Config{
		Region:                      cfg.S3Region,
		Credentials:                 creds,
		EndpointResolverWithOptions: customResolver,
	}

	// Create S3 client with path-style addressing for S3-compatible services
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// Force path-style addressing for compatibility with MinIO and other S3-compatible services
		o.UsePathStyle = true
	})

	service := &S3ServiceImpl{
		client:   s3Client,
		bucket:   cfg.S3Bucket,
		region:   cfg.S3Region,
		endpoint: cfg.S3Endpoint,
	}

	// Test connection by checking if bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(cfg.S3Bucket),
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("bucket", cfg.S3Bucket).
			Str("endpoint", cfg.S3Endpoint).
			Msg("Failed to connect to S3 bucket")
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	log.Info().
		Str("bucket", cfg.S3Bucket).
		Str("region", cfg.S3Region).
		Str("endpoint", cfg.S3Endpoint).
		Msg("Successfully connected to S3")

	return service, nil
}

// UploadFile uploads a file with known size to S3
func (s *S3ServiceImpl) UploadFile(key string, reader io.Reader, size int64, contentType string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set default content type if not provided
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	log.Debug().
		Str("key", key).
		Int64("size", size).
		Str("content_type", contentType).
		Msg("Uploading file to S3")

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
		Metadata: map[string]string{
			"uploaded_at": time.Now().UTC().Format(time.RFC3339),
		},
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to upload file to S3")
		return fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	log.Info().
		Str("key", key).
		Int64("size", size).
		Msg("Successfully uploaded file to S3")

	return nil
}

// UploadStream uploads a file using streaming with multipart upload for large files
func (s *S3ServiceImpl) UploadStream(key string, reader io.Reader) error {
	if err := validateKey(key); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Debug().
		Str("key", key).
		Msg("Starting streaming upload to S3")

	// Use S3 Upload Manager for efficient multipart uploads
	uploader := manager.NewUploader(s.client, func(u *manager.Uploader) {
		// Set part size to 10MB for multipart uploads
		u.PartSize = 10 * 1024 * 1024
	})

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
		Metadata: map[string]string{
			"uploaded_at": time.Now().UTC().Format(time.RFC3339),
		},
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to stream upload file to S3")
		return fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	log.Info().
		Str("key", key).
		Msg("Successfully streamed file to S3")

	return nil
}

// DownloadFile downloads a file from S3 as a stream
func (s *S3ServiceImpl) DownloadFile(key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Debug().
		Str("key", key).
		Msg("Downloading file from S3")

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a not found error
		var notFound *types.NoSuchKey
		if errors.As(err, &notFound) {
			log.Warn().
				Str("key", key).
				Msg("File not found in S3")
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, key)
		}

		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to download file from S3")
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	log.Info().
		Str("key", key).
		Msg("Successfully retrieved file from S3")

	return result.Body, nil
}

// DeleteFile deletes a single file from S3
func (s *S3ServiceImpl) DeleteFile(key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debug().
		Str("key", key).
		Msg("Deleting file from S3")

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to delete file from S3")
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	log.Info().
		Str("key", key).
		Msg("Successfully deleted file from S3")

	return nil
}

// DeleteFiles deletes multiple files from S3 in a batch operation
func (s *S3ServiceImpl) DeleteFiles(keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Validate all keys first
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Debug().
		Int("count", len(keys)).
		Msg("Batch deleting files from S3")

	// Convert keys to S3 object identifiers
	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	// Perform batch delete
	result, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false), // Return list of deleted items
		},
	})

	if err != nil {
		log.Error().
			Err(err).
			Int("count", len(keys)).
			Msg("Failed to batch delete files from S3")
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	// Check for partial failures
	if len(result.Errors) > 0 {
		log.Warn().
			Int("failed_count", len(result.Errors)).
			Int("total_count", len(keys)).
			Msg("Some files failed to delete")

		// Return error with details about failures
		var errorKeys []string
		for _, errItem := range result.Errors {
			errorKeys = append(errorKeys, *errItem.Key)
		}
		return fmt.Errorf("%w: failed to delete %d files: %v", ErrDeleteFailed, len(result.Errors), errorKeys)
	}

	log.Info().
		Int("count", len(keys)).
		Msg("Successfully batch deleted files from S3")

	return nil
}

// GetPresignedURL generates a time-limited presigned URL for file access
func (s *S3ServiceImpl) GetPresignedURL(key string, expirationMinutes int) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}

	// Validate expiration time
	if expirationMinutes <= 0 {
		expirationMinutes = 15 // Default to 15 minutes
	}
	if expirationMinutes > 60 {
		expirationMinutes = 60 // Maximum 60 minutes
	}

	log.Debug().
		Str("key", key).
		Int("expiration_minutes", expirationMinutes).
		Msg("Generating presigned URL")

	// Create presign client
	presignClient := s3.NewPresignClient(s.client)

	// Generate presigned URL for GetObject
	request, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expirationMinutes) * time.Minute
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to generate presigned URL")
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	log.Info().
		Str("key", key).
		Int("expiration_minutes", expirationMinutes).
		Msg("Successfully generated presigned URL")

	return request.URL, nil
}

// FileExists checks if a file exists in S3 without downloading it
func (s *S3ServiceImpl) FileExists(key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a not found error
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}

		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to check if file exists in S3")
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// GetFileMetadata retrieves metadata about a file without downloading it
func (s *S3ServiceImpl) GetFileMetadata(key string) (*FileMetadata, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debug().
		Str("key", key).
		Msg("Retrieving file metadata from S3")

	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a not found error
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			log.Warn().
				Str("key", key).
				Msg("File not found in S3")
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, key)
		}

		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to retrieve file metadata from S3")
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	metadata := &FileMetadata{
		Size:         aws.ToInt64(result.ContentLength),
		ContentType:  aws.ToString(result.ContentType),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         aws.ToString(result.ETag),
	}

	log.Info().
		Str("key", key).
		Int64("size", metadata.Size).
		Msg("Successfully retrieved file metadata")

	return metadata, nil
}

// GenerateS3Key creates a sanitized S3 key from user ID, file ID, and filename
// Format: users/{userID}/{fileID}/{filename}
// This function prevents path traversal attacks by sanitizing the filename
func GenerateS3Key(userID, fileID, filename string) string {
	// Sanitize filename to prevent path traversal
	sanitized := sanitizeFilename(filename)

	// Generate key with consistent structure
	key := fmt.Sprintf("users/%s/%s/%s", userID, fileID, sanitized)

	log.Debug().
		Str("user_id", userID).
		Str("file_id", fileID).
		Str("original_filename", filename).
		Str("sanitized_filename", sanitized).
		Str("key", key).
		Msg("Generated S3 key")

	return key
}

// sanitizeFilename removes path separators and dangerous characters from filename
func sanitizeFilename(filename string) string {
	// Extract just the base filename (removes any path components)
	filename = filepath.Base(filename)

	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Remove control characters
	var builder strings.Builder
	for _, r := range filename {
		if r >= 32 && r != 127 { // Skip control characters
			builder.WriteRune(r)
		}
	}

	sanitized := builder.String()

	// If filename is empty after sanitization, use a default
	if sanitized == "" || sanitized == "." || sanitized == ".." {
		sanitized = "unnamed_file"
	}

	// Trim to maximum length (255 characters)
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}

// validateKey checks if an S3 key is valid
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrInvalidKey)
	}
	if len(key) > 1024 {
		return fmt.Errorf("%w: key too long (max 1024 characters)", ErrInvalidKey)
	}
	// Check for null bytes
	if strings.Contains(key, "\x00") {
		return fmt.Errorf("%w: key contains null byte", ErrInvalidKey)
	}
	return nil
}
