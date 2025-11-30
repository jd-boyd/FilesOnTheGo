package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jd-boyd/filesonthego/models"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// UserService handles user-related business logic
type UserService struct {
	db     *gorm.DB
	logger zerolog.Logger
}

// NewUserService creates a new user service
func NewUserService(db *gorm.DB, logger zerolog.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(email, username, password string, isAdmin bool) (*models.User, error) {
	// Validate input
	if email == "" || username == "" || password == "" {
		return nil, errors.New("email, username, and password are required")
	}

	// Normalize email
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	// Create user
	user := &models.User{
		Email:           email,
		Username:        username,
		EmailVisibility: true,
		IsAdmin:         isAdmin,
	}

	// Set password
	if err := user.SetPassword(password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Validate user
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Save to database
	if err := s.db.Create(user).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			return nil, errors.New("email or username already exists")
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info().
		Str("user_id", user.ID).
		Str("email", email).
		Str("username", username).
		Bool("is_admin", isAdmin).
		Msg("User created successfully")

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	username = strings.TrimSpace(username)

	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// ListUsers retrieves all users with pagination
func (s *UserService) ListUsers(limit, offset int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// Get total count
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated users
	query := s.db.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// UpdateUser updates a user's information
func (s *UserService) UpdateUser(userID string, updates map[string]interface{}) (*models.User, error) {
	// Get existing user
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Validate and apply updates
	allowedFields := map[string]bool{
		"email":            true,
		"username":         true,
		"email_visibility": true,
		"storage_quota":    true,
		"is_admin":         true,
		"verified":         true,
	}

	filteredUpdates := make(map[string]interface{})
	for key, value := range updates {
		if allowedFields[key] {
			filteredUpdates[key] = value
		}
	}

	// Normalize email if present
	if email, ok := filteredUpdates["email"].(string); ok {
		filteredUpdates["email"] = strings.ToLower(strings.TrimSpace(email))
	}

	// Update user
	if err := s.db.Model(user).Updates(filteredUpdates).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			return nil, errors.New("email or username already exists")
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Reload user to get updated values
	user, err = s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("user_id", userID).
		Interface("updates", filteredUpdates).
		Msg("User updated successfully")

	return user, nil
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(userID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Verify old password
	if !user.ValidatePassword(oldPassword) {
		return errors.New("invalid current password")
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database
	if err := s.db.Model(user).Update("password_hash", user.PasswordHash).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.logger.Info().
		Str("user_id", userID).
		Msg("Password updated successfully")

	return nil
}

// ResetPassword resets a user's password (admin function)
func (s *UserService) ResetPassword(userID, newPassword string) error {
	// Get user
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password in database
	if err := s.db.Model(user).Update("password_hash", user.PasswordHash).Error; err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	s.logger.Warn().
		Str("user_id", userID).
		Msg("Password reset by admin")

	return nil
}

// DeleteUser deletes a user and all their data
func (s *UserService) DeleteUser(userID string) error {
	// Start transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get user to check if exists
		var user models.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("user not found")
			}
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Delete user's files
		if err := tx.Where("user = ?", userID).Delete(&models.File{}).Error; err != nil {
			return fmt.Errorf("failed to delete user files: %w", err)
		}

		// Delete user's directories
		if err := tx.Where("user = ?", userID).Delete(&models.Directory{}).Error; err != nil {
			return fmt.Errorf("failed to delete user directories: %w", err)
		}

		// Delete user's shares
		if err := tx.Where("user = ?", userID).Delete(&models.Share{}).Error; err != nil {
			return fmt.Errorf("failed to delete user shares: %w", err)
		}

		// Delete the user
		if err := tx.Delete(&user).Error; err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		s.logger.Warn().
			Str("user_id", userID).
			Str("email", user.Email).
			Msg("User and all associated data deleted")

		return nil
	})
}

// UpdateStorageUsed updates a user's storage usage
func (s *UserService) UpdateStorageUsed(userID string, delta int64) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	if delta > 0 {
		return user.IncrementStorageUsed(s.db, delta)
	} else if delta < 0 {
		return user.DecrementStorageUsed(s.db, -delta)
	}

	return nil
}

// GetUserStats returns statistics about a user
func (s *UserService) GetUserStats(userID string) (map[string]interface{}, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Count files
	var fileCount int64
	if err := s.db.Model(&models.File{}).Where("user = ?", userID).Count(&fileCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count files: %w", err)
	}

	// Count directories
	var dirCount int64
	if err := s.db.Model(&models.Directory{}).Where("user = ?", userID).Count(&dirCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count directories: %w", err)
	}

	// Count shares
	var shareCount int64
	if err := s.db.Model(&models.Share{}).Where("user = ?", userID).Count(&shareCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count shares: %w", err)
	}

	stats := map[string]interface{}{
		"user_id":          user.ID,
		"email":            user.Email,
		"username":         user.Username,
		"storage_used":     user.StorageUsed,
		"storage_quota":    user.StorageQuota,
		"storage_percent":  user.GetQuotaUsagePercent(),
		"available_quota":  user.GetAvailableQuota(),
		"file_count":       fileCount,
		"directory_count":  dirCount,
		"share_count":      shareCount,
		"is_admin":         user.IsAdmin,
		"verified":         user.Verified,
		"created_at":       user.CreatedAt,
	}

	return stats, nil
}

// SearchUsers searches for users by email or username
func (s *UserService) SearchUsers(query string, limit int) ([]*models.User, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*models.User{}, nil
	}

	var users []*models.User
	searchPattern := "%" + query + "%"

	err := s.db.Where("email LIKE ? OR username LIKE ?", searchPattern, searchPattern).
		Limit(limit).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	return users, nil
}
