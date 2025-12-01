//go:build skip_all_integration_tests

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/jd-boyd/filesonthego/tests"
	"github.com/jd-boyd/filesonthego/models"
)

func TestFileUpload_Integration_ValidUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Upload small file to root directory", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create and authenticate user
		user := app.CreateTestUser(t, "uploadtest@example.com", "uploaduser", "testpassword", false)
		token := app.AuthenticateUser(t, user.Email, "testpassword")

		// Prepare file content
		content := []byte("test file content for integration test")
		body, contentType := tests.CreateMultipartUpload("integration-test.txt", content, nil)

		// Create upload request
		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		// Execute request
		w := app.ExecuteRequest(t, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		tests.AssertJSONResponse(t, w, &response)

		// Verify file was created
		fileData, ok := response["file"].(map[string]interface{})
		require.True(t, ok, "Response should contain file data")

		filename, ok := fileData["filename"].(string)
		assert.True(t, ok)
		assert.Equal(t, "integration-test.txt", filename)

		size, ok := fileData["size"].(float64)
		assert.True(t, ok)
		assert.Equal(t, float64(len(content)), size)

		// Verify file exists in database
		var file models.File
		err := app.DB.Where("user = ? AND name = ?", user.ID, "integration-test.txt").First(&file).Error
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), file.Size)
		assert.Equal(t, user.ID, file.User)

		// Verify user quota was updated
		var updatedUser models.User
		err = app.DB.First(&updatedUser, user.ID).Error
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), updatedUser.StorageUsed)
	})

	t.Run("Upload file to subdirectory", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create and authenticate user
		user := app.CreateTestUser(t, "dirtest@example.com", "diruser", "testpassword", false)
		token := app.AuthenticateUser(t, user.Email, "testpassword")

		// Create a subdirectory
		directory := app.CreateTestDirectory(t, user.ID, "TestDir", nil)

		// Prepare file content
		content := []byte("test file in subdirectory")
		body, contentType := tests.CreateMultipartUpload("dir-test.txt", content, map[string]string{
			"directory_id": directory.ID,
		})

		// Create upload request
		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		// Execute request
		w := app.ExecuteRequest(t, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify file exists in database with correct directory
		var file models.File
		err := app.DB.Where("user = ? AND name = ?", user.ID, "dir-test.txt").First(&file).Error
		require.NoError(t, err)
		assert.Equal(t, directory.ID, file.ParentDirectory)
	})

	t.Run("Upload with share token (read permission)", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create user and file
		user := app.CreateTestUser(t, "shareuser@example.com", "shareuser", "testpassword", false)
		file := app.CreateTestFile(t, user.ID, "share-test.txt", "share-test-key", "text/plain", 1024)

		// Create read-only share
		share := app.CreateTestShare(t, user.ID, file.ID, "file", "read", "", nil)

		// Prepare file content
		content := []byte("new file upload")
		body, contentType := tests.CreateMultipartUpload("new-file.txt", content, map[string]string{
			"share_token": share.ShareToken,
		})

		// Create upload request (no auth token, using share token)
		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)

		// Execute request
		w := app.ExecuteRequest(t, req)

		// Should fail because read-only share doesn't allow upload
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestFileUpload_Integration_PermissionDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Upload without authentication", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Prepare file content
		content := []byte("unauthenticated upload attempt")
		body, contentType := tests.CreateMultipartUpload("unauth.txt", content, nil)

		// Create upload request without auth token
		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)

		// Execute request
		w := app.ExecuteRequest(t, req)

		// Should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Upload to directory owned by another user", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create two users
		user1 := app.CreateTestUser(t, "user1@example.com", "user1", "password1", false)
		user2 := app.CreateTestUser(t, "user2@example.com", "user2", "password2", false)

		// User 1 creates directory
		directory := app.CreateTestDirectory(t, user1.ID, "User1Dir", nil)

		// User 2 tries to upload to User 1's directory
		token2 := app.AuthenticateUser(t, user2.Email, "password2")
		content := []byte("unauthorized upload")
		body, contentType := tests.CreateMultipartUpload("unauthorized.txt", content, map[string]string{
			"directory_id": directory.ID,
		})

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token2)

		w := app.ExecuteRequest(t, req)

		// Should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Upload with expired share token", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create user and file
		user := app.CreateTestUser(t, "shareuser@example.com", "shareuser", "testpassword", false)
		file := app.CreateTestFile(t, user.ID, "share-test.txt", "share-test-key", "text/plain", 1024)

		// Create expired share
		expiresAt := time.Now().Add(-1 * time.Hour) // Already expired
		share := app.CreateTestShare(t, user.ID, file.ID, "file", "read_upload", "", &expiresAt)

		// Try to upload with expired share token
		content := []byte("upload with expired token")
		body, contentType := tests.CreateMultipartUpload("expired-upload.txt", content, map[string]string{
			"share_token": share.ShareToken,
		})

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)

		w := app.ExecuteRequest(t, req)

		// Should be forbidden due to expired share
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Upload with read-only share token", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create user and file
		user := app.CreateTestUser(t, "shareuser@example.com", "shareuser", "testpassword", false)
		file := app.CreateTestFile(t, user.ID, "share-test.txt", "share-test-key", "text/plain", 1024)

		// Create read-only share
		share := app.CreateTestShare(t, user.ID, file.ID, "file", "read", "", nil)

		// Try to upload with read-only share token
		content := []byte("upload with read-only token")
		body, contentType := tests.CreateMultipartUpload("readonly-upload.txt", content, map[string]string{
			"share_token": share.ShareToken,
		})

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)

		w := app.ExecuteRequest(t, req)

		// Should be forbidden because read-only doesn't allow upload
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestFileUpload_Integration_QuotaEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Upload exceeds user quota", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create user with small quota (2KB)
		user := app.CreateTestUser(t, "quotauser@example.com", "quotauser", "testpassword", false)

		// Update user to have small quota
		err := app.DB.Model(user).Update("storage_quota", int64(2048)).Error
		require.NoError(t, err)

		token := app.AuthenticateUser(t, user.Email, "testpassword")

		// Try to upload file larger than quota (3KB)
		content := make([]byte, 3072) // 3KB
		for i := range content {
			content[i] = byte('A' + (i % 26))
		}

		body, contentType := tests.CreateMultipartUpload("too-large.txt", content, nil)

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		w := app.ExecuteRequest(t, req)

		// Should be rejected due to quota exceeded
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

		// Verify no file was created in database
		var fileCount int64
		err = app.DB.Model(&models.File{}).Where("user = ??", user.ID).Count(&fileCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), fileCount)
	})

	t.Run("Upload at quota limit", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		// Create user with exactly 2KB quota
		user := app.CreateTestUser(t, "limituser@example.com", "limituser", "testpassword", false)

		// Update user to have 2KB quota
		err := app.DB.Model(user).Update("storage_quota", int64(2048)).Error
		require.NoError(t, err)

		token := app.AuthenticateUser(t, user.Email, "testpassword")

		// Upload file that exactly matches quota (2KB)
		content := make([]byte, 2048) // 2KB
		for i := range content {
			content[i] = byte('B' + (i % 24))
		}

		body, contentType := tests.CreateMultipartUpload("exact-size.txt", content, nil)

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		w := app.ExecuteRequest(t, req)

		// Should succeed
		assert.Equal(t, http.StatusOK, w.Code)

		// Try to upload one more byte
		smallContent := []byte("x")
		smallBody, smallContentType := tests.CreateMultipartUpload("too-much.txt", smallContent, nil)

		smallReq := httptest.NewRequest("POST", "/api/files/upload", smallBody)
		smallReq.Header.Set("Content-Type", smallContentType)
		smallReq.Header.Set("Authorization", "Bearer "+token)

		smallW := app.ExecuteRequest(t, smallReq)

		// Should be rejected due to quota exceeded
		assert.Equal(t, http.StatusRequestEntityTooLarge, smallW.Code)
	})

	t.Run("Quota correctly updated after upload", func(t *testing.T) {
		app := tests.SetupTestApp(t)
		defer app.Cleanup()

		user := app.CreateTestUser(t, "quotatest@example.com", "quotatest", "testpassword", false)
		token := app.AuthenticateUser(t, user.Email, "testpassword")

		// Upload file
		content := []byte("test content for quota verification")
		body, contentType := tests.CreateMultipartUpload("quota-test.txt", content, nil)

		req := httptest.NewRequest("POST", "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		w := app.ExecuteRequest(t, req)

		// Should succeed
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify user quota was updated
		var updatedUser models.User
		err := app.DB.First(&updatedUser, user.ID).Error
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), updatedUser.StorageUsed)
	})
}

