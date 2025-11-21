package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jd-boyd/filesonthego/handlers"
	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApp holds the test application context
type TestApp struct {
	app          *pocketbase.PocketBase
	shareHandler *handlers.ShareHandler
	tmpDir       string
}

// setupIntegrationTest creates a test app for integration testing
func setupIntegrationTest(t *testing.T) *TestApp {
	tmpDir, err := os.MkdirTemp("", "pb_integration_*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DataDir:          tmpDir,
		DataMaxOpenConns: 10,
		DataMaxIdleConns: 2,
		EncryptionEnv:    "test",
	})

	shareService := services.NewShareService(app)
	logger := zerolog.New(os.Stderr)
	shareHandler := handlers.NewShareHandler(app, shareService, logger)

	return &TestApp{
		app:          app,
		shareHandler: shareHandler,
		tmpDir:       tmpDir,
	}
}

// createTestUser creates a test user and returns the record
func (ta *TestApp) createTestUser(t *testing.T) *core.Record {
	collection := ta.app.FindCollectionByNameOrId("users")
	if collection == nil {
		t.Skip("users collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("email", "integration@example.com")
	record.Set("username", "integrationuser")
	record.SetPassword("testpassword")

	err := ta.app.Save(record)
	require.NoError(t, err)

	return record
}

// createTestFile creates a test file
func (ta *TestApp) createTestFile(t *testing.T, userID string) *core.Record {
	collection := ta.app.FindCollectionByNameOrId("files")
	if collection == nil {
		t.Skip("files collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("user", userID)
	record.Set("name", "integration-test.txt")
	record.Set("s3_key", "integration-test-key")
	record.Set("size", 2048)
	record.Set("mime_type", "text/plain")

	err := ta.app.Save(record)
	require.NoError(t, err)

	return record
}

func TestIntegration_CreateShareFlow(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create share request
	reqBody := map[string]interface{}{
		"resource_type":   "file",
		"resource_id":     file.Id,
		"permission_type": "read",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/shares", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	// Create mock RequestEvent
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}

	// Set auth record
	e.Set(core.RequestEventAuthKey, user)

	// Handle request
	err := testApp.shareHandler.CreateShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Parse response
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.NotNil(t, response["share"])
	assert.NotNil(t, response["url"])

	share := response["share"].(map[string]interface{})
	assert.Equal(t, user.Id, share["user_id"])
	assert.NotEmpty(t, share["share_token"])
}

func TestIntegration_CreateShareWithPassword(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create password-protected share
	reqBody := map[string]interface{}{
		"resource_type":   "file",
		"resource_id":     file.Id,
		"permission_type": "read",
		"password":        "integration123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/shares", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}
	e.Set(core.RequestEventAuthKey, user)

	err := testApp.shareHandler.CreateShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	share := response["share"].(map[string]interface{})
	assert.True(t, share["is_password_protected"].(bool))
}

func TestIntegration_CreateShareWithExpiration(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	expiresAt := time.Now().Add(48 * time.Hour)

	reqBody := map[string]interface{}{
		"resource_type":   "file",
		"resource_id":     file.Id,
		"permission_type": "read",
		"expires_at":      expiresAt.Format(time.RFC3339),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/shares", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}
	e.Set(core.RequestEventAuthKey, user)

	err := testApp.shareHandler.CreateShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	share := response["share"].(map[string]interface{})
	assert.NotNil(t, share["expires_at"])
	assert.False(t, share["is_expired"].(bool))
}

func TestIntegration_ListUserShares(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file1 := testApp.createTestFile(t, user.Id)

	// Create file 2
	collection := testApp.app.FindCollectionByNameOrId("files")
	file2 := core.NewRecord(collection)
	file2.Set("user", user.Id)
	file2.Set("name", "integration-test2.txt")
	file2.Set("s3_key", "integration-test-key-2")
	file2.Set("size", 3072)
	file2.Set("mime_type", "text/plain")
	testApp.app.Save(file2)

	// Create shares
	shareService := services.NewShareService(testApp.app)

	shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file1.Id,
		PermissionType: "read",
	})

	shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file2.Id,
		PermissionType: "read_upload",
	})

	// List shares
	req := httptest.NewRequest("GET", "/api/shares", nil)
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}
	e.Set(core.RequestEventAuthKey, user)

	err := testApp.shareHandler.ListShares(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	shares := response["shares"].([]interface{})
	assert.Len(t, shares, 2)
}

func TestIntegration_RevokeShare(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create share
	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Revoke share
	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}
	e.Set(core.RequestEventAuthKey, user)

	// Mock path value
	req.SetPathValue("share_id", share.ID)

	err = testApp.shareHandler.RevokeShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify share no longer exists
	_, err = shareService.GetShareByID(share.ID)
	assert.Error(t, err)
}

func TestIntegration_AccessPublicShare_Valid(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create share
	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Access public share
	req := httptest.NewRequest("GET", "/api/public/share/"+share.ShareToken, nil)
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}

	req.SetPathValue("share_token", share.ShareToken)

	err = testApp.shareHandler.AccessPublicShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.True(t, response["valid"].(bool))
	assert.Equal(t, share.ID, response["share_id"])
}

func TestIntegration_AccessPublicShare_PasswordProtected(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	password := "integration456"

	// Create password-protected share
	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		Password:       password,
	})
	require.NoError(t, err)

	// Access without password
	req := httptest.NewRequest("GET", "/api/public/share/"+share.ShareToken, nil)
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}

	req.SetPathValue("share_token", share.ShareToken)

	err = testApp.shareHandler.AccessPublicShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.False(t, response["valid"].(bool))
	assert.True(t, response["requires_password"].(bool))
}

func TestIntegration_ValidateSharePassword_Correct(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	password := "integration789"

	// Create password-protected share
	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		Password:       password,
	})
	require.NoError(t, err)

	// Validate with correct password
	reqBody := map[string]string{
		"password": password,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/public/share/"+share.ShareToken+"/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}

	req.SetPathValue("share_token", share.ShareToken)

	err = testApp.shareHandler.ValidateSharePassword(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.True(t, response["valid"].(bool))
}

func TestIntegration_ValidateSharePassword_Wrong(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create password-protected share
	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		Password:       "correct_password",
	})
	require.NoError(t, err)

	// Validate with wrong password
	reqBody := map[string]string{
		"password": "wrong_password",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/public/share/"+share.ShareToken+"/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}

	req.SetPathValue("share_token", share.ShareToken)

	err = testApp.shareHandler.ValidateSharePassword(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.False(t, response["valid"].(bool))
}

func TestIntegration_UpdateShareExpiration(t *testing.T) {
	testApp := setupIntegrationTest(t)

	if testApp.app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user := testApp.createTestUser(t)
	file := testApp.createTestFile(t, user.Id)

	// Create share
	shareService := services.NewShareService(testApp.app)
	share, err := shareService.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Update expiration
	newExpiration := time.Now().Add(72 * time.Hour)
	reqBody := map[string]interface{}{
		"expires_at": newExpiration.Format(time.RFC3339),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/shares/"+share.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{
		Request:  req,
		Response: rec,
	}
	e.Set(core.RequestEventAuthKey, user)

	req.SetPathValue("share_id", share.ID)

	err = testApp.shareHandler.UpdateShare(e)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	updatedShare := response["share"].(map[string]interface{})
	assert.NotNil(t, updatedShare["expires_at"])
}
