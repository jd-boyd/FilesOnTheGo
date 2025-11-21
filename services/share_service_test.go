package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jd-boyd/filesonthego/models"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestApp creates a test PocketBase instance
func setupTestApp(t *testing.T) *pocketbase.PocketBase {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "pb_test_*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	// Create PocketBase instance
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DataDir:          tmpDir,
		DataMaxOpenConns: 10,
		DataMaxIdleConns: 2,
		EncryptionEnv:    "test",
	})

	require.NotNil(t, app)
	return app
}

// createTestUser creates a test user
func createTestUser(t *testing.T, app *pocketbase.PocketBase) string {
	collection := app.FindCollectionByNameOrId("users")
	if collection == nil {
		t.Skip("users collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("email", "test@example.com")
	record.Set("username", "testuser")
	record.SetPassword("testpassword")

	err := app.Save(record)
	require.NoError(t, err)

	return record.Id
}

// createTestFile creates a test file for testing
func createTestFile(t *testing.T, app *pocketbase.PocketBase, userID string) string {
	collection := app.FindCollectionByNameOrId("files")
	if collection == nil {
		t.Skip("files collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("user", userID)
	record.Set("name", "test.txt")
	record.Set("s3_key", "test-key")
	record.Set("size", 1024)
	record.Set("mime_type", "text/plain")

	err := app.Save(record)
	require.NoError(t, err)

	return record.Id
}

// createTestDirectory creates a test directory for testing
func createTestDirectory(t *testing.T, app *pocketbase.PocketBase, userID string) string {
	collection := app.FindCollectionByNameOrId("directories")
	if collection == nil {
		t.Skip("directories collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("user", userID)
	record.Set("name", "test-dir")
	record.Set("path", "/test-dir")

	err := app.Save(record)
	require.NoError(t, err)

	return record.Id
}

func TestNewShareService(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	assert.NotNil(t, service)
	assert.IsType(t, &ShareServiceImpl{}, service)
}

func TestShareService_CreateShare_File(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	// Skip if collections don't exist
	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)

	require.NoError(t, err)
	assert.NotNil(t, share)
	assert.Equal(t, userID, share.UserID)
	assert.Equal(t, models.ResourceTypeFile, share.ResourceType)
	assert.Equal(t, fileID, share.ResourceID)
	assert.Equal(t, models.PermissionRead, share.PermissionType)
	assert.NotEmpty(t, share.ShareToken)
	assert.Equal(t, int64(0), share.AccessCount)
	assert.False(t, share.IsPasswordProtected)
}

func TestShareService_CreateShare_Directory(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	dirID := createTestDirectory(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "directory",
		ResourceID:     dirID,
		PermissionType: "read_upload",
	}

	share, err := service.CreateShare(params)

	require.NoError(t, err)
	assert.NotNil(t, share)
	assert.Equal(t, models.ResourceTypeDirectory, share.ResourceType)
	assert.Equal(t, dirID, share.ResourceID)
	assert.Equal(t, models.PermissionReadUpload, share.PermissionType)
}

func TestShareService_CreateShare_WithPassword(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		Password:       "secret123",
	}

	share, err := service.CreateShare(params)

	require.NoError(t, err)
	assert.NotNil(t, share)
	assert.True(t, share.IsPasswordProtected)
}

func TestShareService_CreateShare_WithExpiration(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		ExpiresAt:      &expiresAt,
	}

	share, err := service.CreateShare(params)

	require.NoError(t, err)
	assert.NotNil(t, share)
	assert.NotNil(t, share.ExpiresAt)
	assert.False(t, share.IsExpired)
}

func TestShareService_CreateShare_GeneratesUniqueTokens(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	// Create multiple shares
	share1, err := service.CreateShare(params)
	require.NoError(t, err)

	share2, err := service.CreateShare(params)
	require.NoError(t, err)

	// Tokens should be unique
	assert.NotEqual(t, share1.ShareToken, share2.ShareToken)

	// Tokens should be valid UUIDs
	_, err = uuid.Parse(share1.ShareToken)
	assert.NoError(t, err)

	_, err = uuid.Parse(share2.ShareToken)
	assert.NoError(t, err)
}

func TestShareService_CreateShare_ValidationErrors(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	tests := []struct {
		name   string
		params CreateShareParams
		errMsg string
	}{
		{
			name:   "missing user ID",
			params: CreateShareParams{ResourceType: "file", ResourceID: "123", PermissionType: "read"},
			errMsg: "user ID is required",
		},
		{
			name:   "missing resource type",
			params: CreateShareParams{UserID: "123", ResourceID: "456", PermissionType: "read"},
			errMsg: "resource type is required",
		},
		{
			name:   "missing resource ID",
			params: CreateShareParams{UserID: "123", ResourceType: "file", PermissionType: "read"},
			errMsg: "resource ID is required",
		},
		{
			name:   "missing permission type",
			params: CreateShareParams{UserID: "123", ResourceType: "file", ResourceID: "456"},
			errMsg: "permission type is required",
		},
		{
			name:   "invalid resource type",
			params: CreateShareParams{UserID: "123", ResourceType: "invalid", ResourceID: "456", PermissionType: "read"},
			errMsg: "resource type must be 'file' or 'directory'",
		},
		{
			name:   "invalid permission type",
			params: CreateShareParams{UserID: "123", ResourceType: "file", ResourceID: "456", PermissionType: "invalid"},
			errMsg: "permission type must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share, err := service.CreateShare(tt.params)

			assert.Error(t, err)
			assert.Nil(t, share)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestShareService_CreateShare_ExpiredExpiration(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	pastTime := time.Now().Add(-1 * time.Hour)
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		ExpiresAt:      &pastTime,
	}

	share, err := service.CreateShare(params)

	assert.Error(t, err)
	assert.Nil(t, share)
	assert.Contains(t, err.Error(), "expiration date must be in the future")
}

func TestShareService_CreateShare_NonExistentResource(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	userID := createTestUser(t, app)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     "nonexistent",
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)

	assert.Error(t, err)
	assert.Nil(t, share)
	assert.Contains(t, err.Error(), "not found")
}

func TestShareService_CreateShare_NotOwner(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	// Create two users
	user1ID := createTestUser(t, app)

	// Create second user
	collection := app.FindCollectionByNameOrId("users")
	user2 := core.NewRecord(collection)
	user2.Set("email", "test2@example.com")
	user2.Set("username", "testuser2")
	user2.SetPassword("testpassword")
	app.Save(user2)
	user2ID := user2.Id

	// User1 creates a file
	fileID := createTestFile(t, app, user1ID)

	// User2 tries to share user1's file
	params := CreateShareParams{
		UserID:         user2ID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)

	assert.Error(t, err)
	assert.Nil(t, share)
	assert.Contains(t, err.Error(), "permission")
}

func TestShareService_GetShareByToken(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	// Create share
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	created, err := service.CreateShare(params)
	require.NoError(t, err)

	// Retrieve by token
	retrieved, err := service.GetShareByToken(created.ShareToken)

	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.ShareToken, retrieved.ShareToken)
}

func TestShareService_GetShareByToken_NotFound(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	share, err := service.GetShareByToken("nonexistent-token")

	assert.Error(t, err)
	assert.Nil(t, share)
	assert.Contains(t, err.Error(), "not found")
}

func TestShareService_GetShareByToken_EmptyToken(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	share, err := service.GetShareByToken("")

	assert.Error(t, err)
	assert.Nil(t, share)
	assert.Contains(t, err.Error(), "required")
}

func TestShareService_GetShareByID(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	// Create share
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	created, err := service.CreateShare(params)
	require.NoError(t, err)

	// Retrieve by ID
	retrieved, err := service.GetShareByID(created.ID)

	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
}

func TestShareService_ValidateShareAccess_Valid(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Validate access
	info, err := service.ValidateShareAccess(share.ShareToken, "")

	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.True(t, info.IsValid)
	assert.Empty(t, info.ErrorMessage)
	assert.Equal(t, share.ID, info.ShareID)
}

func TestShareService_ValidateShareAccess_WithCorrectPassword(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	password := "secret123"
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		Password:       password,
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Validate with correct password
	info, err := service.ValidateShareAccess(share.ShareToken, password)

	require.NoError(t, err)
	assert.True(t, info.IsValid)
}

func TestShareService_ValidateShareAccess_WithWrongPassword(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		Password:       "secret123",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Validate with wrong password
	info, err := service.ValidateShareAccess(share.ShareToken, "wrongpassword")

	require.NoError(t, err)
	assert.False(t, info.IsValid)
	assert.Contains(t, info.ErrorMessage, "password")
}

func TestShareService_ValidateShareAccess_MissingPassword(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		Password:       "secret123",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Validate without password
	info, err := service.ValidateShareAccess(share.ShareToken, "")

	require.NoError(t, err)
	assert.False(t, info.IsValid)
	assert.Contains(t, info.ErrorMessage, "Password required")
}

func TestShareService_ValidateShareAccess_Expired(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	// Create share with very short expiration
	expiresAt := time.Now().Add(1 * time.Millisecond)
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		ExpiresAt:      &expiresAt,
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Validate access
	info, err := service.ValidateShareAccess(share.ShareToken, "")

	require.NoError(t, err)
	assert.False(t, info.IsValid)
	assert.Contains(t, info.ErrorMessage, "expired")
}

func TestShareService_ValidateShareAccess_InvalidToken(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	info, err := service.ValidateShareAccess("invalid-token", "")

	require.NoError(t, err)
	assert.False(t, info.IsValid)
	assert.Contains(t, info.ErrorMessage, "Invalid")
}

func TestShareService_RevokeShare(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Revoke share
	err = service.RevokeShare(share.ID, userID)
	assert.NoError(t, err)

	// Verify share no longer exists
	_, err = service.GetShareByID(share.ID)
	assert.Error(t, err)
}

func TestShareService_RevokeShare_NotOwner(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	user1ID := createTestUser(t, app)
	fileID := createTestFile(t, app, user1ID)

	// Create second user
	collection := app.FindCollectionByNameOrId("users")
	user2 := core.NewRecord(collection)
	user2.Set("email", "test2@example.com")
	user2.Set("username", "testuser2")
	user2.SetPassword("testpassword")
	app.Save(user2)
	user2ID := user2.Id

	params := CreateShareParams{
		UserID:         user1ID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// User2 tries to revoke user1's share
	err = service.RevokeShare(share.ID, user2ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}

func TestShareService_ListUserShares(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	file1ID := createTestFile(t, app, userID)

	// Create file 2
	collection := app.FindCollectionByNameOrId("files")
	file2 := core.NewRecord(collection)
	file2.Set("user", userID)
	file2.Set("name", "test2.txt")
	file2.Set("s3_key", "test-key-2")
	file2.Set("size", 2048)
	file2.Set("mime_type", "text/plain")
	app.Save(file2)
	file2ID := file2.Id

	// Create two file shares
	service.CreateShare(CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     file1ID,
		PermissionType: "read",
	})

	service.CreateShare(CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     file2ID,
		PermissionType: "read_upload",
	})

	// List shares
	shares, err := service.ListUserShares(userID, "")

	require.NoError(t, err)
	assert.Len(t, shares, 2)
}

func TestShareService_ListUserShares_FilterByType(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)
	dirID := createTestDirectory(t, app, userID)

	// Create file and directory shares
	service.CreateShare(CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	})

	service.CreateShare(CreateShareParams{
		UserID:         userID,
		ResourceType:   "directory",
		ResourceID:     dirID,
		PermissionType: "read_upload",
	})

	// List only file shares
	fileShares, err := service.ListUserShares(userID, "file")
	require.NoError(t, err)
	assert.Len(t, fileShares, 1)
	assert.Equal(t, models.ResourceTypeFile, fileShares[0].ResourceType)

	// List only directory shares
	dirShares, err := service.ListUserShares(userID, "directory")
	require.NoError(t, err)
	assert.Len(t, dirShares, 1)
	assert.Equal(t, models.ResourceTypeDirectory, dirShares[0].ResourceType)
}

func TestShareService_UpdateShareExpiration(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Update expiration
	newExpiration := time.Now().Add(48 * time.Hour)
	err = service.UpdateShareExpiration(share.ID, userID, &newExpiration)
	assert.NoError(t, err)

	// Verify expiration updated
	updated, err := service.GetShareByID(share.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated.ExpiresAt)
}

func TestShareService_UpdateShareExpiration_RemoveExpiration(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
		ExpiresAt:      &expiresAt,
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)
	assert.NotNil(t, share.ExpiresAt)

	// Remove expiration
	err = service.UpdateShareExpiration(share.ID, userID, nil)
	assert.NoError(t, err)

	// Verify expiration removed
	updated, err := service.GetShareByID(share.ID)
	require.NoError(t, err)
	assert.Nil(t, updated.ExpiresAt)
}

func TestShareService_IncrementAccessCount(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)
	assert.Equal(t, int64(0), share.AccessCount)

	// Increment access count
	err = service.IncrementAccessCount(share.ID)
	assert.NoError(t, err)

	// Verify count incremented
	updated, err := service.GetShareByID(share.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.AccessCount)

	// Increment again
	service.IncrementAccessCount(share.ID)
	updated, _ = service.GetShareByID(share.ID)
	assert.Equal(t, int64(2), updated.AccessCount)
}

func TestShareService_LogShareAccess(t *testing.T) {
	app := setupTestApp(t)
	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		t.Skip("shares collection not found")
	}

	userID := createTestUser(t, app)
	fileID := createTestFile(t, app, userID)

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	share, err := service.CreateShare(params)
	require.NoError(t, err)

	// Log access (may not work if collection doesn't exist, but should not error)
	err = service.LogShareAccess(share.ID, "download", "test.txt", "192.168.1.1", "Mozilla/5.0")
	assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkShareService_CreateShare(b *testing.B) {
	// Create temp dir for benchmark
	tmpDir, _ := os.MkdirTemp("", "pb_bench_*")
	defer os.RemoveAll(tmpDir)

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DataDir: tmpDir,
	})

	service := NewShareService(app)

	// Skip if collections don't exist
	if app.FindCollectionByNameOrId("shares") == nil {
		b.Skip("shares collection not found")
	}

	// Create test user and file
	userCollection := app.FindCollectionByNameOrId("users")
	if userCollection == nil {
		b.Skip("users collection not found")
	}
	userRecord := core.NewRecord(userCollection)
	userRecord.Set("email", "bench@example.com")
	userRecord.Set("username", "benchuser")
	userRecord.SetPassword("password")
	app.Save(userRecord)
	userID := userRecord.Id

	fileCollection := app.FindCollectionByNameOrId("files")
	if fileCollection == nil {
		b.Skip("files collection not found")
	}
	fileRecord := core.NewRecord(fileCollection)
	fileRecord.Set("user", userID)
	fileRecord.Set("name", "bench.txt")
	fileRecord.Set("s3_key", "bench-key")
	fileRecord.Set("size", 1024)
	fileRecord.Set("mime_type", "text/plain")
	app.Save(fileRecord)
	fileID := fileRecord.Id

	params := CreateShareParams{
		UserID:         userID,
		ResourceType:   "file",
		ResourceID:     fileID,
		PermissionType: "read",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CreateShare(params)
	}
}

func BenchmarkShareService_ValidateShareAccess(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "pb_bench_*")
	defer os.RemoveAll(tmpDir)

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DataDir: tmpDir,
	})

	service := NewShareService(app)

	if app.FindCollectionByNameOrId("shares") == nil {
		b.Skip("shares collection not found")
	}

	// Create test data
	userCollection := app.FindCollectionByNameOrId("users")
	if userCollection == nil {
		b.Skip("users collection not found")
	}
	userRecord := core.NewRecord(userCollection)
	userRecord.Set("email", "bench@example.com")
	userRecord.Set("username", "benchuser")
	userRecord.SetPassword("password")
	app.Save(userRecord)

	fileCollection := app.FindCollectionByNameOrId("files")
	if fileCollection == nil {
		b.Skip("files collection not found")
	}
	fileRecord := core.NewRecord(fileCollection)
	fileRecord.Set("user", userRecord.Id)
	fileRecord.Set("name", "bench.txt")
	fileRecord.Set("s3_key", "bench-key")
	fileRecord.Set("size", 1024)
	fileRecord.Set("mime_type", "text/plain")
	app.Save(fileRecord)

	share, _ := service.CreateShare(CreateShareParams{
		UserID:         userRecord.Id,
		ResourceType:   "file",
		ResourceID:     fileRecord.Id,
		PermissionType: "read",
		Password:       "secret123",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ValidateShareAccess(share.ShareToken, "secret123")
	}
}
