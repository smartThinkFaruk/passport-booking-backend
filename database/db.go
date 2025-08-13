package database

import (
	"fmt"
	"os"

	"passport-booking/logger"
	"passport-booking/models/address"
	"passport-booking/models/booking"
	"passport-booking/models/log"
	"passport-booking/models/user"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection with auto migration and indexing
func InitDB() (*gorm.DB, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading .env file", err)
	}

	// Get database configuration from environment variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	database := os.Getenv("DB_DATABASE")
	user := os.Getenv("DB_USERNAME")
	password := os.Getenv("DB_PASSWORD")
	sslmode := os.Getenv("DB_SSLMODE") // Optional: "disable", "require", etc.

	// Set default sslmode if not provided
	if sslmode == "" {
		sslmode = "disable"
	}

	// Build PostgreSQL DSN string
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, database, sslmode)

	fmt.Println("DSN:", dsn)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to the database", err)
		return nil, err
	}
	logger.Success("Successfully connected to the database")

	// Use dynamic migration system instead of simple AutoMigrate
	migrator := NewDynamicMigrator(DB)

	// Detect schema changes
	operations, err := migrator.DetectChanges()
	if err != nil {
		logger.Error("Failed to detect schema changes", err)
		return nil, err
	}

	// Execute migrations
	if err := migrator.ExecuteMigrations(operations); err != nil {
		logger.Error("Failed to execute migrations", err)
		return nil, err
	}
	logger.Success("All dynamic migrations completed successfully")

	// Handle foreign key constraints after migrations
	if err := createForeignKeyConstraints(); err != nil {
		logger.Error("Failed to create foreign key constraints", err)
		return nil, err
	}
	logger.Success("All foreign key constraints created successfully")

	// Create indexes for better performance
	if err := createIndexes(); err != nil {
		logger.Error("Failed to create indexes", err)
		return nil, err
	}
	logger.Success("All indexes created successfully")

	return DB, nil
}

// autoMigrate runs auto migration for all models
func autoMigrate() error {
	// First, migrate models without foreign key constraints in stages

	// Stage 1: Core foundation models
	stage1Models := []interface{}{
		&user.User{},
		&address.Address{},
	}

	for _, model := range stage1Models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Stage 2: Models with dependencies on Stage 1
	stage2Models := []interface{}{
		&booking.Booking{},
	}

	for _, model := range stage2Models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Stage 5: Remaining models
	remainingModels := []interface{}{
		// Logging
		&log.Log{},
	}

	for _, model := range remainingModels {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	return nil
}

// createIndexes creates additional indexes for better performance
func createIndexes() error {
	// User indexes
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_users_uuid ON users(uuid)").Error; err != nil {
		return fmt.Errorf("failed to create user uuid index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
		return fmt.Errorf("failed to create user username index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)").Error; err != nil {
		return fmt.Errorf("failed to create user email index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone)").Error; err != nil {
		return fmt.Errorf("failed to create user phone index: %w", err)
	}

	// Address indexes
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_addresses_division ON addresses(division)").Error; err != nil {
		return fmt.Errorf("failed to create address division index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_addresses_district ON addresses(district)").Error; err != nil {
		return fmt.Errorf("failed to create address district index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_addresses_police_station ON addresses(police_station)").Error; err != nil {
		return fmt.Errorf("failed to create address police_station index: %w", err)
	}

	// Booking indexes
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_app_or_order_id ON bookings(app_or_order_id)").Error; err != nil {
		return fmt.Errorf("failed to create booking app_or_order_id index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_phone ON bookings(phone)").Error; err != nil {
		return fmt.Errorf("failed to create booking phone index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status)").Error; err != nil {
		return fmt.Errorf("failed to create booking status index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_address_id ON bookings(address_id)").Error; err != nil {
		return fmt.Errorf("failed to create booking address_id index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_created_at ON bookings(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create booking created_at index: %w", err)
	}

	// Log indexes
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_logs_method ON logs(method)").Error; err != nil {
		return fmt.Errorf("failed to create log method index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_logs_status_code ON logs(status_code)").Error; err != nil {
		return fmt.Errorf("failed to create log status_code index: %w", err)
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create log created_at index: %w", err)
	}

	return nil
}

// createForeignKeyConstraints creates foreign key constraints after auto migration
func createForeignKeyConstraints() error {
	// Define constraints with their names for checking existence
	constraints := []struct {
		name string
		sql  string
	}{
		{
			name: "fk_bookings_address",
			sql: `ALTER TABLE bookings ADD CONSTRAINT fk_bookings_address 
				  FOREIGN KEY (address_id) REFERENCES addresses(id) 
				  ON UPDATE CASCADE ON DELETE RESTRICT`,
		},
	}

	for _, constraint := range constraints {
		// Check if constraint already exists
		var exists bool
		checkSQL := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints 
				WHERE constraint_name = $1
			)
		`

		err := DB.Raw(checkSQL, constraint.name).Scan(&exists).Error
		if err != nil {
			logger.Warning(fmt.Sprintf("Failed to check constraint existence: %s - Error: %v", constraint.name, err))
			continue
		}

		if !exists {
			if err := DB.Exec(constraint.sql).Error; err != nil {
				logger.Warning(fmt.Sprintf("Failed to create constraint: %s - Error: %v", constraint.name, err))
			} else {
				logger.Success(fmt.Sprintf("Successfully created constraint: %s", constraint.name))
			}
		} else {
			logger.Debug(fmt.Sprintf("Constraint already exists: %s", constraint.name))
		}
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Legacy function for backward compatibility
func ConnectDB() (*gorm.DB, error) {
	return InitDB()
}
