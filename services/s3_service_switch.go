package services

import (
	"github.com/jd-boyd/filesonthego/config"
)

// NewS3Service creates a new S3 service instance using the lightweight implementation
// This replaces the AWS SDK implementation for better build performance
func NewS3Service(cfg *config.Config) (S3Service, error) {
	return NewLightweightS3Service(cfg)
}