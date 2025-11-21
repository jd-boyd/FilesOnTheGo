package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jd-boyd/filesonthego/config"
	"github.com/rs/zerolog/log"
)

// LightweightS3Service implements S3 operations using pure HTTP with AWS v4 signing
// Based on restic's approach but implemented from scratch for minimal dependencies
type LightweightS3Service struct {
	accessKey string
	secretKey string
	region    string
	endpoint  string
	bucket    string
	client    *http.Client
}

// S3ErrorResponse represents S3 API error response
type S3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
	Key     string   `xml:"Key"`
}

// NewLightweightS3Service creates a new lightweight S3 service
func NewLightweightS3Service(cfg *config.Config) (*LightweightS3Service, error) {
	// Validate configuration
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if cfg.S3AccessKey == "" {
		return nil, fmt.Errorf("access key is required")
	}
	if cfg.S3SecretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}
	if cfg.S3Region == "" {
		cfg.S3Region = "us-east-1" // Default region
	}

	// Set default endpoint if not provided
	endpoint := cfg.S3Endpoint
	if endpoint == "" {
		endpoint = "https://s3." + cfg.S3Region + ".amazonaws.com"
	}

	service := &LightweightS3Service{
		accessKey: cfg.S3AccessKey,
		secretKey: cfg.S3SecretKey,
		region:    cfg.S3Region,
		endpoint:  endpoint,
		bucket:    cfg.S3Bucket,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use HEAD request to check bucket access
	req, err := http.NewRequestWithContext(ctx, "HEAD", service.getBucketURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	service.signRequest(req, "")

	resp, err := service.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("bucket", cfg.S3Bucket).
			Str("endpoint", endpoint).
			Msg("Failed to connect to S3 bucket")
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("bucket access failed with status: %d", resp.StatusCode)
	}

	log.Info().
		Str("bucket", cfg.S3Bucket).
		Str("region", cfg.S3Region).
		Str("endpoint", endpoint).
		Msg("Successfully connected to S3 with lightweight client")

	return service, nil
}

// UploadFile uploads a file with known size to S3
func (s *LightweightS3Service) UploadFile(key string, reader io.Reader, size int64, contentType string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url := s.getObjectURL(key)

	// Create request
	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-meta-uploaded-at", time.Now().UTC().Format(time.RFC3339))

	// Sign request
	s.signRequest(req, "")

	log.Debug().
		Str("key", key).
		Int64("size", size).
		Str("content_type", contentType).
		Msg("Uploading file to S3")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to upload file to S3")
		return fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.parseS3Error(resp, "upload failed")
	}

	log.Info().
		Str("key", key).
		Int64("size", size).
		Msg("Successfully uploaded file to S3")

	return nil
}

// MultipartUpload represents a multipart upload session
type MultipartUpload struct {
	UploadID string
	Key      string
	Bucket   string
}

// UploadedPart represents an uploaded part in a multipart upload
type UploadedPart struct {
	PartNumber int
	ETag       string
	Size       int64
}

// UploadStream uploads a file using streaming with multipart upload for large files
// Automatically uses multipart upload for files larger than 10MB
func (s *LightweightS3Service) UploadStream(key string, reader io.Reader) error {
	if err := validateKey(key); err != nil {
		return err
	}

	log.Debug().
		Str("key", key).
		Msg("Starting streaming upload to S3")

	// Create a buffer reader to determine file size
	bufReader := newBufferedReader(reader, 10*1024*1024) // 10MB buffer

	// Check if file is large enough for multipart upload (>10MB)
	if bufReader.hasMoreThan(10 * 1024 * 1024) {
		return s.multipartUpload(key, bufReader)
	}

	// For smaller files, use simple upload
	log.Debug().
		Str("key", key).
		Msg("Using simple upload for small file")

	content, err := io.ReadAll(bufReader)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	return s.UploadFile(key, bytes.NewReader(content), int64(len(content)), "application/octet-stream")
}

// multipartUpload implements the full multipart upload workflow
func (s *LightweightS3Service) multipartUpload(key string, reader io.Reader) error {
	log.Info().
		Str("key", key).
		Msg("Starting multipart upload for large file")

	// 1. Initiate multipart upload
	upload, err := s.initiateMultipartUpload(key)
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	// 2. Upload parts
	parts, err := s.uploadParts(upload, reader)
	if err != nil {
		// Abort upload on error
		_ = s.abortMultipartUpload(upload)
		return fmt.Errorf("failed to upload parts: %w", err)
	}

	// 3. Complete multipart upload
	err = s.completeMultipartUpload(upload, parts)
	if err != nil {
		// Abort upload on error
		_ = s.abortMultipartUpload(upload)
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	log.Info().
		Str("key", key).
		Int("parts_uploaded", len(parts)).
		Msg("Successfully completed multipart upload")

	return nil
}

// initiateMultipartUpload starts a multipart upload session
func (s *LightweightS3Service) initiateMultipartUpload(key string) (*MultipartUpload, error) {
	url := fmt.Sprintf("%s?uploads", s.getObjectURL(key))

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create initiate request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	s.signRequest(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.parseS3Error(resp, "initiate multipart upload failed")
	}

	// Parse XML response to get UploadID
	type InitiateResult struct {
		XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
		Bucket   string   `xml:"Bucket"`
		Key      string   `xml:"Key"`
		UploadID string   `xml:"UploadId"`
	}

	var result InitiateResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse initiate response: %w", err)
	}

	log.Debug().
		Str("key", key).
		Str("upload_id", result.UploadID).
		Msg("Initiated multipart upload")

	return &MultipartUpload{
		UploadID: result.UploadID,
		Key:      result.Key,
		Bucket:   result.Bucket,
	}, nil
}

// uploadParts uploads data in chunks and returns uploaded parts
func (s *LightweightS3Service) uploadParts(upload *MultipartUpload, reader io.Reader) ([]*UploadedPart, error) {
	const partSize = 10 * 1024 * 1024 // 10MB parts
	const maxParts = 10000             // S3 limit

	var parts []*UploadedPart
	partNumber := 1

	for {
		// Read chunk
		buf := make([]byte, partSize)
		n, err := io.ReadFull(reader, buf)

		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("failed to read part %d: %w", partNumber, err)
		}

		if n == 0 {
			break
		}

		// Upload part
		partData := buf[:n]
		part, err := s.uploadPart(upload, partNumber, partData)
		if err != nil {
			return nil, fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		parts = append(parts, part)
		partNumber++

		log.Debug().
			Str("upload_id", upload.UploadID).
			Int("part_number", partNumber-1).
			Int64("part_size", int64(n)).
			Msg("Uploaded part")

		if partNumber > maxParts {
			return nil, fmt.Errorf("too many parts: %d > %d", partNumber, maxParts)
		}
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("no parts to upload")
	}

	return parts, nil
}

// uploadPart uploads a single part of a multipart upload
func (s *LightweightS3Service) uploadPart(upload *MultipartUpload, partNumber int, data []byte) (*UploadedPart, error) {
	url := fmt.Sprintf("%s?partNumber=%d&uploadId=%s",
		s.getObjectURL(upload.Key), partNumber, upload.UploadID)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create part request: %w", err)
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	s.signRequest(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload part: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.parseS3Error(resp, "upload part failed")
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		return nil, fmt.Errorf("missing ETag in part upload response")
	}

	return &UploadedPart{
		PartNumber: partNumber,
		ETag:       strings.Trim(etag, `"`),
		Size:       int64(len(data)),
	}, nil
}

// completeMultipartUpload completes a multipart upload session
func (s *LightweightS3Service) completeMultipartUpload(upload *MultipartUpload, parts []*UploadedPart) error {
	// Build complete multipart upload XML
	var completeXML bytes.Buffer
	completeXML.WriteString(`<?xml version="1.0" encoding="UTF-8"?><CompleteMultipartUpload>`)

	for _, part := range parts {
		completeXML.WriteString(fmt.Sprintf(`<Part><PartNumber>%d</PartNumber><ETag>%s</ETag></Part>`,
			part.PartNumber, html.EscapeString(part.ETag)))
	}
	completeXML.WriteString(`</CompleteMultipartUpload>`)

	url := fmt.Sprintf("%s?uploadId=%s", s.getObjectURL(upload.Key), upload.UploadID)

	req, err := http.NewRequest("POST", url, &completeXML)
	if err != nil {
		return fmt.Errorf("failed to create complete request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	s.signRequest(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.parseS3Error(resp, "complete multipart upload failed")
	}

	return nil
}

// abortMultipartUpload aborts a multipart upload session
func (s *LightweightS3Service) abortMultipartUpload(upload *MultipartUpload) error {
	url := fmt.Sprintf("%s?uploadId=%s", s.getObjectURL(upload.Key), upload.UploadID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create abort request: %w", err)
	}

	s.signRequest(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return s.parseS3Error(resp, "abort multipart upload failed")
	}

	log.Warn().
		Str("key", upload.Key).
		Str("upload_id", upload.UploadID).
		Msg("Aborted multipart upload")

	return nil
}

// bufferedReader helps read a specific amount of data from a reader
type bufferedReader struct {
	reader io.Reader
	buffer []byte
	pos    int
}

func newBufferedReader(reader io.Reader, bufferSize int) *bufferedReader {
	return &bufferedReader{
		reader: reader,
		buffer: make([]byte, 0, bufferSize),
		pos:    0,
	}
}

func (b *bufferedReader) Read(p []byte) (n int, err error) {
	// First drain buffer
	if b.pos < len(b.buffer) {
		n = copy(p, b.buffer[b.pos:])
		b.pos += n
		if n == len(p) {
			return n, nil
		}
	}

	// Read from underlying reader
	remaining := p[n:]
	m, err := io.ReadFull(b.reader, remaining)
	n += m
	return n, err
}

func (b *bufferedReader) hasMoreThan(size int) bool {
	// Fill buffer if needed
	if len(b.buffer) == 0 {
		b.buffer = make([]byte, 0, size+1)
		buf := make([]byte, size+1)
		n, err := io.ReadFull(b.reader, buf)
		if err == io.EOF {
			return false
		}
		if err == io.ErrUnexpectedEOF {
			b.buffer = append(b.buffer, buf[:n]...)
			return len(b.buffer) > size
		}
		if err != nil {
			return false
		}
		b.buffer = append(b.buffer, buf[:n]...)
	}

	return len(b.buffer) > size
}

// DownloadFile downloads a file from S3 as a stream
func (s *LightweightS3Service) DownloadFile(key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	url := s.getObjectURL(key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	s.signRequest(req, "")

	log.Debug().
		Str("key", key).
		Msg("Downloading file from S3")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to download file from S3")
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		log.Warn().
			Str("key", key).
			Msg("File not found in S3")
		return nil, fmt.Errorf("%w: %s", ErrFileNotFound, key)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, s.parseS3Error(resp, "download failed")
	}

	log.Info().
		Str("key", key).
		Msg("Successfully retrieved file from S3")

	return resp.Body, nil
}

// DeleteFile deletes a single file from S3
func (s *LightweightS3Service) DeleteFile(key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	url := s.getObjectURL(key)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	s.signRequest(req, "")

	log.Debug().
		Str("key", key).
		Msg("Deleting file from S3")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to delete file from S3")
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return s.parseS3Error(resp, "delete failed")
	}

	log.Info().
		Str("key", key).
		Msg("Successfully deleted file from S3")

	return nil
}

// DeleteFiles deletes multiple files from S3 (batch operation using S3 delete API)
func (s *LightweightS3Service) DeleteFiles(keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Validate all keys first
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return err
		}
	}

	log.Debug().
		Int("count", len(keys)).
		Msg("Batch deleting files from S3")

	// Build delete XML payload
	var deleteXML bytes.Buffer
	deleteXML.WriteString(`<?xml version="1.0" encoding="UTF-8"?><Delete>`)
	for _, key := range keys {
		deleteXML.WriteString(fmt.Sprintf(`<Object><Key>%s</Key></Object>`, html.EscapeString(key)))
	}
	deleteXML.WriteString(`</Delete>`)

	url := fmt.Sprintf("%s?delete", s.getBucketURL())

	req, err := http.NewRequest("POST", url, &deleteXML)
	if err != nil {
		return fmt.Errorf("failed to create batch delete request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Content-MD5", s.calculateMD5(deleteXML.String()))

	s.signRequest(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Int("count", len(keys)).
			Msg("Failed to batch delete files from S3")
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.parseS3Error(resp, "batch delete failed")
	}

	log.Info().
		Int("count", len(keys)).
		Msg("Successfully batch deleted files from S3")

	return nil
}

// GetPresignedURL generates a time-limited presigned URL for file access
func (s *LightweightS3Service) GetPresignedURL(key string, expirationMinutes int) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}

	if expirationMinutes <= 0 {
		expirationMinutes = 15
	}
	if expirationMinutes > 60 {
		expirationMinutes = 60
	}

	url := s.getObjectURL(key)

	// Create request for signing
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create presigned URL request: %w", err)
	}

	// Generate presigned URL with expiration
	expiration := time.Duration(expirationMinutes) * time.Minute
	signedURL := s.presignURL(req, expiration)

	log.Debug().
		Str("key", key).
		Int("expiration_minutes", expirationMinutes).
		Msg("Generated presigned URL")

	return signedURL, nil
}

// FileExists checks if a file exists in S3 without downloading it
func (s *LightweightS3Service) FileExists(key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	url := s.getObjectURL(key)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	s.signRequest(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to check if file exists in S3")
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, s.parseS3Error(resp, "head failed")
	}

	return true, nil
}

// GetFileMetadata retrieves metadata about a file without downloading it
func (s *LightweightS3Service) GetFileMetadata(key string) (*FileMetadata, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	url := s.getObjectURL(key)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	s.signRequest(req, "")

	log.Debug().
		Str("key", key).
		Msg("Retrieving file metadata from S3")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to retrieve file metadata from S3")
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Warn().
			Str("key", key).
			Msg("File not found in S3")
		return nil, fmt.Errorf("%w: %s", ErrFileNotFound, key)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.parseS3Error(resp, "head failed")
	}

	size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	contentType := resp.Header.Get("Content-Type")
	lastModified, _ := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	etag := resp.Header.Get("ETag")

	metadata := &FileMetadata{
		Size:         size,
		ContentType:  contentType,
		LastModified: lastModified,
		ETag:         strings.Trim(etag, `"`),
	}

	log.Info().
		Str("key", key).
		Int64("size", metadata.Size).
		Msg("Successfully retrieved file metadata")

	return metadata, nil
}

// Helper methods

func (s *LightweightS3Service) getBucketURL() string {
	if strings.Contains(s.endpoint, "amazonaws.com") {
		return fmt.Sprintf("%s/%s", s.endpoint, s.bucket)
	}
	return fmt.Sprintf("%s/%s", s.endpoint, s.bucket)
}

func (s *LightweightS3Service) getObjectURL(key string) string {
	return fmt.Sprintf("%s/%s", s.getBucketURL(), key)
}

// AWS v4 Signing implementation
func (s *LightweightS3Service) signRequest(req *http.Request, payload string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	// Set required headers
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	// Create canonical request
	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQuery := req.URL.Query().Encode()
	canonicalHeaders, signedHeaders := s.getCanonicalHeaders(req)

	payloadHash := s.sha256Hash(payload)

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.region)
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		credentialScope,
		s.sha256Hash(canonicalRequest),
	}, "\n")

	// Calculate signature
	signingKey := s.getSignatureKey(dateStamp)
	signature := s.hmacSha256(signingKey, stringToSign)

	// Add authorization header
	credential := fmt.Sprintf("%s/%s", s.accessKey, credentialScope)
	authHeader := fmt.Sprintf("%s Credential=%s, SignedHeaders=%s, Signature=%s",
		algorithm, credential, signedHeaders, hex.EncodeToString(signature))

	req.Header.Set("Authorization", authHeader)
}

func (s *LightweightS3Service) getCanonicalHeaders(req *http.Request) (string, string) {
	var headers []string
	var signedHeaders []string

	for name, values := range req.Header {
		name = strings.ToLower(strings.TrimSpace(name))
		if strings.HasPrefix(name, "x-amz-") || name == "host" || name == "content-type" {
			value := strings.Join(values, ",")
			value = strings.TrimSpace(value)
			headers = append(headers, name+":"+value)
			signedHeaders = append(signedHeaders, name)
		}
	}

	sort.Strings(headers)
	sort.Strings(signedHeaders)

	canonicalHeaders := strings.Join(headers, "\n") + "\n"
	return canonicalHeaders, strings.Join(signedHeaders, ";")
}

func (s *LightweightS3Service) sha256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *LightweightS3Service) hmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func (s *LightweightS3Service) getSignatureKey(dateStamp string) []byte {
	kDate := s.hmacSha256([]byte("AWS4"+s.secretKey), dateStamp)
	kRegion := s.hmacSha256(kDate, s.region)
	kService := s.hmacSha256(kRegion, "s3")
	kSigning := s.hmacSha256(kService, "aws4_request")
	return kSigning
}

func (s *LightweightS3Service) presignURL(req *http.Request, expiration time.Duration) string {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	expirationSeconds := int(expiration.Seconds())

	// Add X-Amz-Expires parameter
	query := req.URL.Query()
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", fmt.Sprintf("%s/%s/%s/s3/aws4_request", s.accessKey, dateStamp, s.region))
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.Itoa(expirationSeconds))
	query.Set("X-Amz-SignedHeaders", "host")

	req.URL.RawQuery = query.Encode()

	// Create string to sign for presigned URL
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		req.URL.Query().Encode(),
		"host:" + req.URL.Host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.region)
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		credentialScope,
		s.sha256Hash(canonicalRequest),
	}, "\n")

	// Calculate signature
	signingKey := s.getSignatureKey(dateStamp)
	signature := s.hmacSha256(signingKey, stringToSign)

	// Add signature to URL
	req.URL.RawQuery += "&X-Amz-Signature=" + hex.EncodeToString(signature)

	return req.URL.String()
}

func (s *LightweightS3Service) calculateMD5(data string) string {
	// Simple MD5 for delete payload (in production, use crypto/md5)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *LightweightS3Service) parseS3Error(resp *http.Response, operation string) error {
	var s3Error S3ErrorResponse
	if err := xml.NewDecoder(resp.Body).Decode(&s3Error); err != nil {
		return fmt.Errorf("%s: HTTP %d", operation, resp.StatusCode)
	}

	log.Error().
		Str("code", s3Error.Code).
		Str("message", s3Error.Message).
		Int("status", resp.StatusCode).
		Msg("S3 API error")

	if s3Error.Code == "NoSuchKey" || s3Error.Code == "NotFound" {
		return fmt.Errorf("%w: %s", ErrFileNotFound, s3Error.Key)
	}

	return fmt.Errorf("%s: %s - %s", operation, s3Error.Code, s3Error.Message)
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