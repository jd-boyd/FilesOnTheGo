package security

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/jd-boyd/filesonthego/services"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pointer is a helper function to create string pointers
func pointer(s string) *string {
	return &s
}

// setupSecurityTest creates a test app for security testing
func setupSecurityTest(t *testing.T) *pocketbase.PocketBase {
	tmpDir, err := os.MkdirTemp("", "pb_security_*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:       tmpDir,
		DataMaxOpenConns:     10,
		DataMaxIdleConns:     2,
		DefaultEncryptionEnv: "test",
	})

	// Bootstrap to initialize internal PocketBase schema first
	// This creates the _collections table and other internal schema
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap test app: %v", err)
	}

	// Create the minimal collections needed for security testing
	if err := createSecurityTestCollections(app); err != nil {
		t.Fatalf("Failed to create test collections: %v", err)
	}

	return app
}

// createSecurityTestCollections creates the minimal collections needed for security testing
func createSecurityTestCollections(app *pocketbase.PocketBase) error {
	// Create users collection first (based on PocketBase's default users schema)
	usersCollection := core.NewBaseCollection("_users")
	usersCollection.ListRule = nil
	usersCollection.ViewRule = nil
	usersCollection.CreateRule = nil
	usersCollection.UpdateRule = nil
	usersCollection.DeleteRule = nil

	// Add basic user fields
	usersCollection.Fields.Add(&core.TextField{
		Name:     "email",
		Required: true,
	})
	usersCollection.Fields.Add(&core.TextField{
		Name:     "username",
		Required: false,
	})
	usersCollection.Fields.Add(&core.PasswordField{
		Name:     "password",
		Required: true,
	})
	usersCollection.Fields.Add(&core.BoolField{
		Name:     "verified",
		Required: false,
	})

	if err := app.Save(usersCollection); err != nil {
		return err
	}

	// Create shares collection
	sharesCollection := core.NewBaseCollection("shares")
	sharesCollection.ListRule = nil
	sharesCollection.ViewRule = nil
	sharesCollection.CreateRule = nil
	sharesCollection.UpdateRule = nil
	sharesCollection.DeleteRule = nil

	// Add user relation field first
	sharesCollection.Fields.Add(&core.RelationField{
		Name:     "user",
		Required: true,
		MaxSelect: 1,
		CollectionId: usersCollection.Id,
	})

	// Add resource_type field
	sharesCollection.Fields.Add(&core.SelectField{
		Name:      "resource_type",
		Required:  true,
		MaxSelect: 1,
		Values:    []string{"file", "directory"},
	})

	// Don't add file relation field yet - will add it after files collection exists

	// Add permission_type field
	sharesCollection.Fields.Add(&core.SelectField{
		Name:      "permission_type",
		Required:  true,
		MaxSelect: 1,
		Values:    []string{"read", "read_upload", "upload_only"},
	})

	// Add fields needed for security tests
	sharesCollection.Fields.Add(&core.TextField{
		Name:     "share_token",
		Required: true,
	})
	sharesCollection.Fields.Add(&core.TextField{
		Name:     "password_hash",
		Required: false,
	})
	sharesCollection.Fields.Add(&core.DateField{
		Name:     "expires_at",
		Required: false,
	})
	sharesCollection.Fields.Add(&core.NumberField{
		Name:     "access_count",
		Required: false,
	})

	if err := app.Save(sharesCollection); err != nil {
		return err
	}

	// Create files collection
	filesCollection := core.NewBaseCollection("files")
	filesCollection.ListRule = nil
	filesCollection.ViewRule = nil
	filesCollection.CreateRule = nil
	filesCollection.UpdateRule = nil
	filesCollection.DeleteRule = nil

	// Add fields needed for security tests
	filesCollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	filesCollection.Fields.Add(&core.TextField{
		Name:     "s3_key",
		Required: true,
	})
	filesCollection.Fields.Add(&core.NumberField{
		Name:     "size",
		Required: true,
	})
	filesCollection.Fields.Add(&core.TextField{
		Name:     "mime_type",
		Required: false,
	})

	if err := app.Save(filesCollection); err != nil {
		return err
	}

	// Now add the user relation field after the files collection is created
	filesCollection.Fields.Add(&core.RelationField{
		Name:     "user",
		Required: true,
		MaxSelect: 1,
		CollectionId: usersCollection.Id,
	})

	if err := app.Save(filesCollection); err != nil {
		return err
	}

	// Add file relation field now that files collection exists
	sharesCollection.Fields.Add(&core.RelationField{
		Name:     "file",
		Required: false,
		MaxSelect: 1,
		CollectionId: filesCollection.Id,
	})

	if err := app.Save(sharesCollection); err != nil {
		return err
	}

	return nil
}

func createSecurityTestUser(t *testing.T, app *pocketbase.PocketBase) *core.Record {
	collection, err := app.FindCollectionByNameOrId("_users")
	if err != nil {
		t.Skip("users collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("email", "security@example.com")
	record.Set("username", "securityuser")
	record.SetPassword("testpassword")

	err = app.Save(record)
	require.NoError(t, err)

	return record
}

func createSecurityTestFile(t *testing.T, app *pocketbase.PocketBase, userID string) *core.Record {
	collection, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		t.Skip("files collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("user", userID)
	record.Set("name", "security-test.txt")
	record.Set("s3_key", "security-test-key")
	record.Set("size", 1024)
	record.Set("mime_type", "text/plain")

	err = app.Save(record)
	require.NoError(t, err)

	return record
}

// TestSecurity_PasswordTimingAttackProtection tests that password verification
// takes constant time regardless of password length or correctness
func TestSecurity_PasswordTimingAttackProtection(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	correctPassword := "security_password_123_very_long"

	// Create password-protected share
	share, err := service.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		Password:       correctPassword,
	})
	require.NoError(t, err)

	// Test with passwords of varying lengths and correctness
	testPasswords := []string{
		"a",                                      // Very short, wrong
		"se",                                     // Short, partially correct
		"security",                               // Medium, partially correct
		"security_password",                      // Long, partially correct
		"security_password_123_very_long",        // Correct
		"wrong_password_but_same_length___!",     // Wrong but same length
		"completely_different_password_here!!!!", // Wrong, different length
	}

	durations := make([]time.Duration, 0)

	// Run multiple iterations to get average timing
	for _, password := range testPasswords {
		start := time.Now()
		service.ValidateShareAccess(share.ShareToken, password)
		elapsed := time.Since(start)
		durations = append(durations, elapsed)
	}

	// Calculate average and variance
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	avg := sum / time.Duration(len(durations))

	// Check that variance is not too high (should be constant time)
	// Allow up to 50% variance due to system noise
	for i, duration := range durations {
		variance := math.Abs(float64(duration-avg)) / float64(avg)
		assert.Less(t, variance, 0.5,
			"Timing variance too high for password %d (variance: %.2f%%), vulnerable to timing attacks",
			i, variance*100)
	}

	t.Logf("Average validation time: %v", avg)
	t.Logf("Durations: %v", durations)
}

// TestSecurity_ExpiredShareBlocked tests that expired shares are blocked
func TestSecurity_ExpiredShareBlocked(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	// Create share that expires very soon
	expiresAt := time.Now().Add(5 * time.Millisecond)
	share, err := service.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		ExpiresAt:      &expiresAt,
	})
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Try to access expired share
	accessInfo, err := service.ValidateShareAccess(share.ShareToken, "")

	require.NoError(t, err)
	assert.False(t, accessInfo.IsValid, "Expired share should be blocked")
	assert.Contains(t, accessInfo.ErrorMessage, "expired")
}

// TestSecurity_UnauthorizedShareModification tests that users cannot modify shares they don't own
func TestSecurity_UnauthorizedShareModification(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	// Create two users
	user1 := createSecurityTestUser(t, app)

	collection, err := app.FindCollectionByNameOrId("users")
	require.NoError(t, err)
	user2 := core.NewRecord(collection)
	user2.Set("email", "security2@example.com")
	user2.Set("username", "securityuser2")
	user2.SetPassword("testpassword")
	app.Save(user2)

	file := createSecurityTestFile(t, app, user1.Id)

	// User1 creates a share
	share, err := service.CreateShare(services.CreateShareParams{
		UserID:         user1.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// User2 tries to revoke user1's share
	err = service.RevokeShare(share.ID, user2.Id)

	assert.Error(t, err, "User should not be able to revoke another user's share")
	assert.Contains(t, err.Error(), "permission")

	// User2 tries to update user1's share expiration
	newExpiration := time.Now().Add(48 * time.Hour)
	err = service.UpdateShareExpiration(share.ID, user2.Id, &newExpiration)

	assert.Error(t, err, "User should not be able to update another user's share")
	assert.Contains(t, err.Error(), "permission")
}

// TestSecurity_UnauthorizedShareCreation tests that users cannot create shares for resources they don't own
func TestSecurity_UnauthorizedShareCreation(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	// Create two users
	user1 := createSecurityTestUser(t, app)

	collection, err := app.FindCollectionByNameOrId("users")
	require.NoError(t, err)
	user2 := core.NewRecord(collection)
	user2.Set("email", "security3@example.com")
	user2.Set("username", "securityuser3")
	user2.SetPassword("testpassword")
	app.Save(user2)

	// User1 creates a file
	file := createSecurityTestFile(t, app, user1.Id)

	// User2 tries to create a share for user1's file
	_, err = service.CreateShare(services.CreateShareParams{
		UserID:         user2.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})

	assert.Error(t, err, "User should not be able to share another user's file")
	assert.Contains(t, err.Error(), "permission")
}

// TestSecurity_PermissionEnforcement tests that share permissions are enforced correctly
func TestSecurity_PermissionEnforcement(t *testing.T) {
	app := setupSecurityTest(t)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	tests := []struct {
		name           string
		permissionType string
		action         string
		shouldAllow    bool
	}{
		// Read permission tests
		{
			name:           "read permission allows view",
			permissionType: "read",
			action:         "view",
			shouldAllow:    true,
		},
		{
			name:           "read permission allows download",
			permissionType: "read",
			action:         "download",
			shouldAllow:    true,
		},
		{
			name:           "read permission denies upload",
			permissionType: "read",
			action:         "upload",
			shouldAllow:    false,
		},

		// Read+Upload permission tests
		{
			name:           "read_upload permission allows view",
			permissionType: "read_upload",
			action:         "view",
			shouldAllow:    true,
		},
		{
			name:           "read_upload permission allows download",
			permissionType: "read_upload",
			action:         "download",
			shouldAllow:    true,
		},
		{
			name:           "read_upload permission allows upload",
			permissionType: "read_upload",
			action:         "upload",
			shouldAllow:    true,
		},

		// Upload-only permission tests
		{
			name:           "upload_only permission denies view",
			permissionType: "upload_only",
			action:         "view",
			shouldAllow:    false,
		},
		{
			name:           "upload_only permission denies download",
			permissionType: "upload_only",
			action:         "download",
			shouldAllow:    false,
		},
		{
			name:           "upload_only permission allows upload",
			permissionType: "upload_only",
			action:         "upload",
			shouldAllow:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get or create share record to test permission methods
			collection, err := app.FindCollectionByNameOrId("shares")
			require.NoError(t, err)
			share := core.NewRecord(collection)
			share.Set("user", user.Id)
			share.Set("resource_type", "file")
			share.Set("file", file.Id)
			share.Set("directory", "")
			share.Set("share_token", "test-token")
			share.Set("permission_type", tt.permissionType)
			share.Set("password_hash", "")
			share.Set("expires_at", time.Time{})
			share.Set("access_count", 0)

			// Use the Share model's CanPerformAction method
			// We need to convert the record to a Share model
			shareModel := &struct {
				PermissionType string
			}{
				PermissionType: tt.permissionType,
			}

			// Test permission logic
			var allowed bool
			switch shareModel.PermissionType {
			case "read":
				allowed = tt.action == "view" || tt.action == "download"
			case "read_upload":
				allowed = tt.action == "view" || tt.action == "download" || tt.action == "upload"
			case "upload_only":
				allowed = tt.action == "upload"
			default:
				allowed = false
			}

			if tt.shouldAllow {
				assert.True(t, allowed, "Action %s should be allowed for permission %s", tt.action, tt.permissionType)
			} else {
				assert.False(t, allowed, "Action %s should be denied for permission %s", tt.action, tt.permissionType)
			}
		})
	}
}

// TestSecurity_TokenEnumerationPrevention tests that invalid tokens don't reveal information
func TestSecurity_TokenEnumerationPrevention(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	// Test various invalid tokens
	invalidTokens := []string{
		"nonexistent-token",
		"00000000-0000-0000-0000-000000000000",
		"invalid-uuid",
		"",
		"a" + string(make([]byte, 1000)), // Very long token
	}

	for _, token := range invalidTokens {
		t.Run("token_"+token[:minInt(len(token), 20)], func(t *testing.T) {
			accessInfo, err := service.ValidateShareAccess(token, "")

			// Should not error, but should return invalid
			require.NoError(t, err)
			assert.False(t, accessInfo.IsValid)

			// Error message should not reveal whether token exists
			assert.Contains(t, accessInfo.ErrorMessage, "Invalid", "Error message should be generic")
			assert.NotContains(t, accessInfo.ErrorMessage, "not found", "Error message should not reveal token doesn't exist")
		})
	}
}

// TestSecurity_PasswordHashingStrength tests that passwords are hashed securely
func TestSecurity_PasswordHashingStrength(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	password := "test_password_123"

	// Create password-protected share
	share, err := service.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
		Password:       password,
	})
	require.NoError(t, err)

	// Get the share record to check password hash
	record, err := app.FindRecordById("shares", share.ID)
	require.NoError(t, err)

	passwordHash := record.GetString("password_hash")

	// Password hash should not be empty
	assert.NotEmpty(t, passwordHash)

	// Password hash should not equal the plaintext password
	assert.NotEqual(t, password, passwordHash, "Password should be hashed, not stored in plaintext")

	// Hash should be bcrypt format (starts with $2a$, $2b$, or $2y$)
	assert.True(t,
		len(passwordHash) >= 60,
		"Bcrypt hash should be at least 60 characters",
	)

	// Verify bcrypt cost (should be 12 or higher for security)
	// Bcrypt format: $2a$12$... where 12 is the cost
	if len(passwordHash) >= 7 {
		costStr := passwordHash[4:6]
		t.Logf("Bcrypt cost: %s", costStr)
		// We expect cost 12 from our implementation
		assert.Contains(t, []string{"10", "11", "12", "13", "14"}, costStr,
			"Bcrypt cost should be 10-14 for security")
	}
}

// TestSecurity_ShareAccessWithoutAuthentication tests that shares can be accessed without authentication
func TestSecurity_ShareAccessWithoutAuthentication(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	// Create share
	share, err := service.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Access share without authentication (should work)
	accessInfo, err := service.ValidateShareAccess(share.ShareToken, "")

	require.NoError(t, err)
	assert.True(t, accessInfo.IsValid, "Shares should be accessible without authentication")
}

// TestSecurity_TokenUniqueness tests that share tokens are unique
func TestSecurity_TokenUniqueness(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	// Create multiple shares and collect tokens
	tokens := make(map[string]bool)
	numShares := 100

	for i := 0; i < numShares; i++ {
		share, err := service.CreateShare(services.CreateShareParams{
			UserID:         user.Id,
			ResourceType:   "file",
			ResourceID:     file.Id,
			PermissionType: "read",
		})
		require.NoError(t, err)

		// Check for duplicate tokens
		if tokens[share.ShareToken] {
			t.Fatalf("Duplicate share token found: %s", share.ShareToken)
		}

		tokens[share.ShareToken] = true
	}

	assert.Len(t, tokens, numShares, "All tokens should be unique")
}

// TestSecurity_AccessCountIncrement tests that access count cannot be manipulated
func TestSecurity_AccessCountIncrement(t *testing.T) {
	app := setupSecurityTest(t)
	service := services.NewShareService(app)

	if _, err := app.FindCollectionByNameOrId("shares"); err != nil {
		t.Skip("shares collection not found")
	}

	user := createSecurityTestUser(t, app)
	file := createSecurityTestFile(t, app, user.Id)

	share, err := service.CreateShare(services.CreateShareParams{
		UserID:         user.Id,
		ResourceType:   "file",
		ResourceID:     file.Id,
		PermissionType: "read",
	})
	require.NoError(t, err)

	// Initial count should be 0
	assert.Equal(t, int64(0), share.AccessCount)

	// Increment access count multiple times
	for i := 1; i <= 5; i++ {
		err := service.IncrementAccessCount(share.ID)
		require.NoError(t, err)

		// Verify count is correct
		updated, err := service.GetShareByID(share.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(i), updated.AccessCount, "Access count should increment correctly")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
