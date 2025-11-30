package database

import (
	"fmt"
	"time"

	"github.com/jd-boyd/filesonthego/models"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Config holds database configuration
type Config struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        logger.LogLevel
}

// Initialize initializes the database connection
func Initialize(config Config) error {
	log.Info().Str("dsn", config.DSN).Msg("Initializing database connection")

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(config.LogLevel)

	// Open database connection
	db, err := gorm.Open(sqlite.Open(config.DSN), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get generic database object to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database object: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Set global DB instance
	DB = db

	log.Info().Msg("Database connection initialized successfully")

	return nil
}

// AutoMigrate runs automatic migrations for all models
func AutoMigrate() error {
	log.Info().Msg("Running automatic database migrations")

	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Run auto-migration for all models
	err := DB.AutoMigrate(
		&models.User{},
		&models.File{},
		&models.Directory{},
		&models.Share{},
		&models.ShareAccessLog{},
	)

	if err != nil {
		return fmt.Errorf("failed to run auto-migration: %w", err)
	}

	log.Info().Msg("Database migrations completed successfully")

	return nil
}

// Close closes the database connection
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database object: %w", err)
	}

	log.Info().Msg("Closing database connection")
	return sqlDB.Close()
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Health checks the database connection health
func Health() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database object: %w", err)
	}

	return sqlDB.Ping()
}
