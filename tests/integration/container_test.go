//go:build container

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ContainerTestConfig holds configuration for containerized tests
type ContainerTestConfig struct {
	BaseURL    string
	AdminEmail string
	AdminPass  string
	HTTPClient *http.Client
}

// SetupContainerTest configures the test environment for containerized testing
func SetupContainerTest(t *testing.T) *ContainerTestConfig {
	// Get configuration from environment or use defaults
	baseURL := getEnvOrDefault("APP_URL", "http://localhost:8090")
	adminEmail := getEnvOrDefault("ADMIN_EMAIL", "admin@filesonthego.local")
	adminPass := getEnvOrDefault("ADMIN_PASSWORD", "admin123")

	config := &ContainerTestConfig{
		BaseURL:    baseURL,
		AdminEmail: adminEmail,
		AdminPass:  adminPass,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Wait for the service to be ready
	t.Logf("Waiting for service at %s to be ready...", config.BaseURL)
	assert.Eventually(t, func() bool {
		resp, err := config.HTTPClient.Get(config.BaseURL + "/api/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 60*time.Second, 2*time.Second, "Service did not become ready within timeout")

	t.Logf("Service is ready at %s", config.BaseURL)
	return config
}

// LoginResponse represents the response from a login request
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"user"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version,omitempty"`
}

// TestContainer_LoginFlow tests the basic login functionality
func TestContainer_LoginFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping container test in short mode")
	}

	config := SetupContainerTest(t)

	// Test health endpoint first
	t.Run("Health_Check", func(t *testing.T) {
		resp, err := config.HTTPClient.Get(config.BaseURL + "/api/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "ok", health.Status)
		assert.NotEmpty(t, health.Timestamp)
	})

	// Test admin login
	t.Run("Admin_Login", func(t *testing.T) {
		loginData := map[string]string{
			"identity": config.AdminEmail,
			"password": config.AdminPass,
		}

		body, _ := json.Marshal(loginData)
		resp, err := config.HTTPClient.Post(
			config.BaseURL+"/api/collections/users/auth-with-password",
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var loginResp LoginResponse
		err = json.NewDecoder(resp.Body).Decode(&loginResp)
		require.NoError(t, err)

		assert.NotEmpty(t, loginResp.Token)
		assert.Equal(t, config.AdminEmail, loginResp.User.Email)
	})

	// Test invalid login
	t.Run("Invalid_Login", func(t *testing.T) {
		loginData := map[string]string{
			"identity": "invalid@example.com",
			"password": "wrongpassword",
		}

		body, _ := json.Marshal(loginData)
		resp, err := config.HTTPClient.Post(
			config.BaseURL+"/api/collections/users/auth-with-password",
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)

		assert.NotEmpty(t, errorResp.Error)
	})

	// Test authenticated request with admin token
	t.Run("Authenticated_Request", func(t *testing.T) {
		// First, login to get token
		token := getAuthToken(t, config)

		// Then use token to access protected endpoint
		req, err := http.NewRequest("GET", config.BaseURL+"/api/collections/files", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := config.HTTPClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test unauthorized access without token
	t.Run("Unauthorized_Access", func(t *testing.T) {
		resp, err := config.HTTPClient.Get(config.BaseURL + "/api/collections/files")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should either return 401 (Unauthorized) or 403 (Forbidden) depending on implementation
		assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden)
	})
}

// TestContainer_UserRegistration tests user registration functionality
func TestContainer_UserRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping container test in short mode")
	}

	config := SetupContainerTest(t)

	t.Run("Register_New_User", func(t *testing.T) {
		// Generate unique user data
		timestamp := time.Now().Unix()
		email := fmt.Sprintf("testuser%d@example.com", timestamp)
		username := fmt.Sprintf("testuser%d", timestamp)
		password := "testpassword123"

		registerData := map[string]string{
			"email":    email,
			"username": username,
			"password": password,
			"passwordConfirm": password,
		}

		body, _ := json.Marshal(registerData)
		resp, err := config.HTTPClient.Post(
			config.BaseURL+"/api/collections/users/records",
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Registration should succeed if public registration is enabled
		// or return 403 if it's disabled
		assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusForbidden)

		if resp.StatusCode == http.StatusCreated {
			// If registration succeeded, try to login with the new user
			loginData := map[string]string{
				"identity": email,
				"password": password,
			}

			body, _ = json.Marshal(loginData)
			loginResp, err := config.HTTPClient.Post(
				config.BaseURL+"/api/collections/users/auth-with-password",
				"application/json",
				bytes.NewReader(body),
			)
			require.NoError(t, err)
			defer loginResp.Body.Close()

			assert.Equal(t, http.StatusOK, loginResp.StatusCode)

			var loginToken LoginResponse
			err = json.NewDecoder(loginResp.Body).Decode(&loginToken)
			require.NoError(t, err)

			assert.NotEmpty(t, loginToken.Token)
			assert.Equal(t, email, loginToken.User.Email)
		}
	})
}

// getAuthToken performs login and returns the authentication token
func getAuthToken(t *testing.T, config *ContainerTestConfig) string {
	loginData := map[string]string{
		"identity": config.AdminEmail,
		"password": config.AdminPass,
	}

	body, _ := json.Marshal(loginData)
	resp, err := config.HTTPClient.Post(
		config.BaseURL+"/api/collections/users/auth-with-password",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(t, err)

	return loginResp.Token
}

// makeAuthenticatedRequest creates an authenticated HTTP request
func makeAuthenticatedRequest(t *testing.T, config *ContainerTestConfig, method, url string, body io.Reader) (*http.Request, string) {
	token := getAuthToken(t, config)

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	req.Header.Set("Content-Type", "application/json")

	return req, token
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	// This would typically use os.Getenv, but for container tests we'll hardcode defaults
	// since the environment will be set by the test runner script
	switch key {
	case "APP_URL":
		return "http://localhost:8090"
	case "ADMIN_EMAIL":
		return "admin@filesonthego.local"
	case "ADMIN_PASSWORD":
		return "admin123"
	default:
		return defaultValue
	}
}