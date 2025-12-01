package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jd-boyd/filesonthego/auth"
	"github.com/jd-boyd/filesonthego/config"
	"github.com/jd-boyd/filesonthego/models"
	handlers "github.com/jd-boyd/filesonthego/handlers_gin"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestApp holds all the dependencies for testing
type TestApp struct {
	DB                *gorm.DB
	Config            *config.Config
	JWTManager        *auth.JWTManager
	SessionManager    *auth.SessionManager
	Router            *gin.Engine
	UserService       *services.UserService
	ShareService      *services.ShareService
	PermissionService *services.PermissionService
	S3Service         services.S3Service
	TempDir           string
	Cleanup           func()
}

// SetupTestApp creates a complete test application with temporary database
func SetupTestApp(t *testing.T) *TestApp {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "filesonthego-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "test.db")

	// Create test configuration
	cfg := &config.Config{
		AppEnvironment:   "test",
		DBPath:           dbPath,
		MaxUploadSize:    100 * 1024 * 1024, // 100MB
		S3Bucket:         "test-bucket",
		S3Region:         "us-east-1",
		S3Endpoint:       "http://localhost:9000",
		S3AccessKey:      "test-access-key",
		S3SecretKey:      "test-secret-key",
		DefaultUserQuota: 10 * 1024 * 1024 * 1024, // 10GB
		PublicRegistration: true,
		TLSEnabled:      false,
	}

	// Initialize in-memory database
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate all models
	err = db.AutoMigrate(
		&models.User{},
		&models.File{},
		&models.Directory{},
		&models.Share{},
		&models.ShareAccessLog{},
	)
	require.NoError(t, err)

	// Initialize JWT manager
	jwtConfig := auth.JWTConfig{
		SecretKey:        []byte("test-secret-key"),
		AccessExpiration: 24 * time.Hour,
		Issuer:           "filesonthego-test",
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

	// Initialize session manager
	sessionConfig := auth.SessionConfig{
		CookieName:     "filesonthego_test_session",
		CookieDomain:   "",
		CookiePath:     "/",
		CookieSecure:   false, // Not needed for tests
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		MaxAge:         24 * time.Hour,
	}
	sessionManager := auth.NewSessionManager(jwtManager, sessionConfig)

	// Initialize services - use NoOp logger for tests
	noOpLogger := zerolog.New(io.Discard)
	userService := services.NewUserService(db, noOpLogger)
	shareService := services.NewShareService(db, noOpLogger)
	permissionService := services.NewPermissionService(db, noOpLogger)

	// Mock S3 service for tests
	s3Service := NewMockS3Service()

	// Initialize template renderer (minimal for tests)
	templateRenderer := handlers.NewTemplateRenderer("./assets/templates")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, templateRenderer, noOpLogger, cfg, jwtManager, sessionManager)
	fileUploadHandler := handlers.NewFileUploadHandler(db, s3Service, permissionService, userService, noOpLogger, cfg)
	fileDownloadHandler := handlers.NewFileDownloadHandler(db, s3Service, permissionService, noOpLogger)
	directoryHandler := handlers.NewDirectoryHandler(db, permissionService, noOpLogger, templateRenderer)
	shareHandler := handlers.NewShareHandler(db, shareService, permissionService, noOpLogger, templateRenderer)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test router
	router := gin.New()
	router.Use(gin.Recovery())

	// Health check
	router.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"environment": cfg.AppEnvironment,
			"version":     "0.2.0-test",
		})
	})

	// Auth routes
	router.POST("/api/auth/login", authHandler.HandleLogin)
	router.POST("/api/auth/register", authHandler.HandleRegister)

	// Protected routes
	protected := router.Group("/")
	protected.Use(sessionManager.RequireAuth())
	{
		protected.POST("/api/files/upload", fileUploadHandler.HandleUpload)
		protected.GET("/api/files/:id/download", fileDownloadHandler.HandleDownload)
		protected.DELETE("/api/files/:id", fileDownloadHandler.HandleDelete)
		protected.GET("/api/directories", directoryHandler.ListDirectory)
		protected.POST("/api/directories", directoryHandler.CreateDirectory)
		protected.DELETE("/api/directories/:id", directoryHandler.DeleteDirectory)
		protected.POST("/api/shares", shareHandler.CreateShare)
		protected.GET("/api/shares", shareHandler.ListShares)
		protected.GET("/api/shares/:id", shareHandler.GetShare)
		protected.DELETE("/api/shares/:id", shareHandler.RevokeShare)
	}

	// Public share access
	router.GET("/share", shareHandler.AccessShare)
	router.POST("/share", shareHandler.AccessShare)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return &TestApp{
		DB:                db,
		Config:            cfg,
		JWTManager:        jwtManager,
		SessionManager:    sessionManager,
		Router:            router,
		UserService:       userService,
		ShareService:      shareService,
		PermissionService: permissionService,
		S3Service:         s3Service,
		TempDir:           tempDir,
		Cleanup:           cleanup,
	}
}

// CreateTestUser creates a test user and returns the user model
func (app *TestApp) CreateTestUser(t *testing.T, email, username, password string, isAdmin bool) *models.User {
	user, err := app.UserService.CreateUser(email, username, password, isAdmin)
	require.NoError(t, err)

	return user
}

// CreateTestFile creates a test file record
func (app *TestApp) CreateTestFile(t *testing.T, userID, filename, s3Key, mimeType string, size int64) *models.File {
	file := &models.File{
		Name:            filename,
		User:            userID,
		ParentDirectory: "", // Root directory
		S3Key:           s3Key,
		MimeType:        mimeType,
		Size:            size,
		S3Bucket:        "test-bucket",
	}

	err := app.DB.Create(file).Error
	require.NoError(t, err)

	return file
}

// CreateTestDirectory creates a test directory
func (app *TestApp) CreateTestDirectory(t *testing.T, userID, name string, parentID *string) *models.Directory {
	var parentDir string
	if parentID != nil {
		parentDir = *parentID
	}

	directory := &models.Directory{
		Name:            name,
		User:            userID,
		ParentDirectory: parentDir,
	}

	err := app.DB.Create(directory).Error
	require.NoError(t, err)

	return directory
}

// CreateTestShare creates a test share
func (app *TestApp) CreateTestShare(t *testing.T, userID, resourceID, resourceType, permissionType string, password string, expiresAt *time.Time) *models.Share {
	share := &models.Share{
		User:           userID,
		PermissionType: models.PermissionType(permissionType),
		ResourceType:   models.ResourceType(resourceType),
	}

	if resourceType == "file" {
		share.File = resourceID
	} else {
		share.Directory = resourceID
	}

	if password != "" {
		err := share.SetPassword(password)
		require.NoError(t, err)
	}

	if expiresAt != nil {
		share.ExpiresAt = expiresAt
	}

	err := app.DB.Create(share).Error
	require.NoError(t, err)

	return share
}

// AuthenticateUser performs login and returns auth token
func (app *TestApp) AuthenticateUser(t *testing.T, email, password string) string {
	// The new Gin handlers expect form data, not JSON
	body := strings.NewReader("email=" + url.QueryEscape(email) + "&password=" + url.QueryEscape(password))
	req := httptest.NewRequest("POST", "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := app.ExecuteRequest(t, req)

	// Handle both direct JSON response (200) and redirect (302)
	var response map[string]interface{}
	var token string

	if w.Code == http.StatusOK {
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		var ok bool
		token, ok = response["token"].(string)
		require.True(t, ok, "Login response should contain token")
	} else if w.Code == http.StatusFound {
		// Check if token is in a cookie
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == "filesonthego_test_session" || cookie.Name == "filesonthego_session" {
				token = cookie.Value
				break
			}
		}
		require.NotEmpty(t, token, "Login should set session cookie")
	} else {
		require.Fail(t, "Expected login to succeed with status 200 or 302, got %d", w.Code)
	}

	return token
}

// AuthenticateUserWithCookie performs login and sets session cookie in request context
func (app *TestApp) AuthenticateUserWithCookie(t *testing.T, email, password string) *http.Cookie {
	// Create form data for login (handler expects form data, not JSON)
	formData := url.Values{}
	formData.Set("email", email)
	formData.Set("password", password)

	body := strings.NewReader(formData.Encode())
	req := httptest.NewRequest("POST", "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)

	// Accept both 200 (JSON response) and 302 (redirect) as successful login
	if w.Code != http.StatusOK && w.Code != http.StatusFound {
		require.Fail(t, "Expected login to succeed with status 200 or 302, got %d", w.Code)
	}

	// Extract session cookie
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "filesonthego_test_session" || cookie.Name == "filesonthego_session" {
			return cookie
		}
	}

	t.Fatal("No session cookie found in login response")
	return nil
}

// CreateMultipartUpload creates a multipart form data request body for file upload
func CreateMultipartUpload(filename string, content []byte, extraFields map[string]string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)

	// Add extra fields
	for key, value := range extraFields {
		writer.WriteField(key, value)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

// MakeAuthenticatedRequest creates an HTTP request with authentication token
func (app *TestApp) MakeAuthenticatedRequest(t *testing.T, method, url string, body io.Reader, token string) *http.Request {
	req := httptest.NewRequest(method, url, body)

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

// MakeAuthenticatedRequestWithCookie creates an HTTP request with session cookie
func (app *TestApp) MakeAuthenticatedRequestWithCookie(t *testing.T, method, url string, body io.Reader, cookie *http.Cookie) *http.Request {
	req := httptest.NewRequest(method, url, body)

	if cookie != nil {
		req.AddCookie(cookie)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

// ExecuteRequest executes a request and returns the response recorder
func (app *TestApp) ExecuteRequest(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}

// AssertJSONResponse asserts that response is valid JSON and unmarshals it
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	err := json.Unmarshal(w.Body.Bytes(), target)
	require.NoError(t, err)
}

// AssertErrorResponse asserts that response contains expected error
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	require.Equal(t, expectedStatus, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	message, ok := response["message"].(string)
	if expectedMessage != "" {
		require.True(t, ok, "Response should contain message field")
		require.Contains(t, message, expectedMessage)
	}
}

// MockS3Service provides a mock implementation of S3Service for testing
type MockS3Service struct {
	Files map[string][]byte
}

// NewMockS3Service creates a new mock S3 service
func NewMockS3Service() *MockS3Service {
	return &MockS3Service{
		Files: make(map[string][]byte),
	}
}

// UploadFile implements S3Service interface
func (m *MockS3Service) UploadFile(key string, data io.Reader, size int64, contentType string) error {
	content, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.Files[key] = content
	return nil
}

// UploadStream implements S3Service interface
func (m *MockS3Service) UploadStream(key string, data io.Reader) error {
	content, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.Files[key] = content
	return nil
}

// DownloadFile implements S3Service interface
func (m *MockS3Service) DownloadFile(key string) (io.ReadCloser, error) {
	content, exists := m.Files[key]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}


// GetPresignedURL implements S3Service interface
func (m *MockS3Service) GetPresignedURL(key string, expirationMinutes int) (string, error) {
	return fmt.Sprintf("http://localhost:9000/test-bucket/%s?presigned=true", key), nil
}

// FileExists implements S3Service interface
func (m *MockS3Service) FileExists(key string) (bool, error) {
	_, exists := m.Files[key]
	return exists, nil
}

// GetFileMetadata implements S3Service interface
func (m *MockS3Service) GetFileMetadata(key string) (*services.FileMetadata, error) {
	content, exists := m.Files[key]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", key)
	}
	return &services.FileMetadata{
		Size:         int64(len(content)),
		LastModified: time.Now(),
		ContentType:  "application/octet-stream",
	}, nil
}

// DeleteFile implements S3Service interface
func (m *MockS3Service) DeleteFile(key string) error {
	delete(m.Files, key)
	return nil
}

// DeleteFiles implements S3Service interface (batch delete)
func (m *MockS3Service) DeleteFiles(keys []string) error {
	for _, key := range keys {
		delete(m.Files, key)
	}
	return nil
}

