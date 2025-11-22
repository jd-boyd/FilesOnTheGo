package handlers

import (
	"io"

	"github.com/jd-boyd/filesonthego/services"
	"github.com/stretchr/testify/mock"
)

// MockS3Service is a mock implementation of S3Service
type MockS3Service struct {
	mock.Mock
}

func (m *MockS3Service) UploadFile(key string, reader io.Reader, size int64, contentType string) error {
	args := m.Called(key, reader, size, contentType)
	return args.Error(0)
}

func (m *MockS3Service) UploadStream(key string, reader io.Reader) error {
	args := m.Called(key, reader)
	return args.Error(0)
}

func (m *MockS3Service) DownloadFile(key string) (io.ReadCloser, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockS3Service) DeleteFile(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockS3Service) DeleteFiles(keys []string) error {
	args := m.Called(keys)
	return args.Error(0)
}

func (m *MockS3Service) GetPresignedURL(key string, expirationMinutes int) (string, error) {
	args := m.Called(key, expirationMinutes)
	return args.String(0), args.Error(1)
}

func (m *MockS3Service) FileExists(key string) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *MockS3Service) GetFileMetadata(key string) (*services.FileMetadata, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.FileMetadata), args.Error(1)
}

// MockPermissionService is a mock implementation of PermissionService
type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) CanReadFile(userID, fileID, shareToken string) (bool, error) {
	args := m.Called(userID, fileID, shareToken)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanUploadFile(userID, directoryID, shareToken string) (bool, error) {
	args := m.Called(userID, directoryID, shareToken)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanDeleteFile(userID, fileID string) (bool, error) {
	args := m.Called(userID, fileID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanMoveFile(userID, fileID, targetDirID string) (bool, error) {
	args := m.Called(userID, fileID, targetDirID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanReadDirectory(userID, directoryID, shareToken string) (bool, error) {
	args := m.Called(userID, directoryID, shareToken)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanCreateDirectory(userID, parentDirID string) (bool, error) {
	args := m.Called(userID, parentDirID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanDeleteDirectory(userID, directoryID string) (bool, error) {
	args := m.Called(userID, directoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanCreateShare(userID, resourceID, resourceType string) (bool, error) {
	args := m.Called(userID, resourceID, resourceType)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanRevokeShare(userID, shareID string) (bool, error) {
	args := m.Called(userID, shareID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) ValidateShareToken(shareToken, password string) (*services.SharePermissions, error) {
	args := m.Called(shareToken, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.SharePermissions), args.Error(1)
}

func (m *MockPermissionService) CanUploadSize(userID string, fileSize int64) (bool, error) {
	args := m.Called(userID, fileSize)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) GetUserQuota(userID string) (*services.QuotaInfo, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.QuotaInfo), args.Error(1)
}
