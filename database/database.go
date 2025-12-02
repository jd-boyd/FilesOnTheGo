package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jd-boyd/filesonthego/models"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
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

	// Open database connection using modernc SQLite driver directly
	sqlDB, err := sql.Open("sqlite", config.DSN)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Create GORM instance
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

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
