package database

import (
	"fmt"
	"os"
	"strings"

	"passport-booking/logger"
	"passport-booking/models/address"
	"passport-booking/models/booking"
	"passport-booking/models/log"
	"passport-booking/models/otp"
	"passport-booking/models/parcel_booking"
	"passport-booking/models/regional_passport_office"
	"passport-booking/models/slip_parser"
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
	// Run auto migration for all models
	if err := autoMigrate(); err != nil {
		logger.Error("Failed to run auto migration", err)
		return nil, err
	}
	logger.Success("All auto migrations completed successfully")

	// Handle foreign key constraints after migrations
	if err := createForeignKeyConstraints(); err != nil {
		logger.Error("Failed to create foreign key constraints", err)
		// Don't return error here, just log and continue
		logger.Warning("Continuing without some foreign key constraints")
	} else {
		logger.Success("All foreign key constraints created successfully")
	}

	// Create indexes for better performance (after migrations)
	if err := createIndexes(); err != nil {
		logger.Error("Failed to create indexes", err)
		// Don't return error here, just log and continue
		logger.Warning("Continuing without some indexes")
	} else {
		logger.Success("All indexes created successfully")
	}

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
		&booking.BookingEvent{},
		&booking.BookingStatusEvent{},
		&otp.OTP{},
		&otp.OTPEvent{},
	}

	for _, model := range stage2Models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Stage 3: Remaining models
	remainingModels := []interface{}{
		// Logging
		&log.Log{},
		// Slip Parser
		&slip_parser.SlipParserRequest{},
		// Regional Passport Office
		&regional_passport_office.RegionalPassportOffice{},
		// Parcel Booking
		&parcel_booking.ParcelBooking{},
		&parcel_booking.ParcelBookingStatusEvent{},
	}

	for _, model := range remainingModels {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	return nil
}

// tableExists checks if a table exists in the database
func tableExists(tableName string) bool {
	var exists bool
	err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = CURRENT_SCHEMA() 
			AND table_name = ?
			AND table_type = 'BASE TABLE'
		)`, tableName).Scan(&exists).Error
	return err == nil && exists
}

// createIndexes creates additional indexes for better performance
func createIndexes() error {
	// User indexes
	if tableExists("users") {
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
	}

	// Address indexes
	if tableExists("addresses") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_addresses_division ON addresses(division)").Error; err != nil {
			return fmt.Errorf("failed to create address division index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_addresses_district ON addresses(district)").Error; err != nil {
			return fmt.Errorf("failed to create address district index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_addresses_police_station ON addresses(police_station)").Error; err != nil {
			return fmt.Errorf("failed to create address police_station index: %w", err)
		}
	}

	// Booking indexes
	if tableExists("bookings") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_app_or_order_id ON bookings(app_or_order_id)").Error; err != nil {
			return fmt.Errorf("failed to create booking app_or_order_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_phone ON bookings(phone)").Error; err != nil {
			return fmt.Errorf("failed to create booking phone index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status)").Error; err != nil {
			return fmt.Errorf("failed to create booking status index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_delivery_address_id ON bookings(delivery_address_id)").Error; err != nil {
			return fmt.Errorf("failed to create booking delivery_address_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_created_at ON bookings(created_at)").Error; err != nil {
			return fmt.Errorf("failed to create booking created_at index: %w", err)
		}
	}

	// Log indexes
	if tableExists("logs") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_logs_method ON logs(method)").Error; err != nil {
			return fmt.Errorf("failed to create log method index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_logs_status_code ON logs(status_code)").Error; err != nil {
			return fmt.Errorf("failed to create log status_code index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at)").Error; err != nil {
			return fmt.Errorf("failed to create log created_at index: %w", err)
		}
	}

	// Slip Parser indexes (only if table exists)
	if tableExists("slip_parser_requests") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_request_id ON slip_parser_requests(request_id)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser request_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_status ON slip_parser_requests(status)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser status index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_app_or_order_id ON slip_parser_requests(app_or_order_id)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser app_or_order_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_phone ON slip_parser_requests(phone)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser phone index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_created_at ON slip_parser_requests(created_at)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser created_at index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_ip_address ON slip_parser_requests(ip_address)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser ip_address index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_slip_parser_requests_file_hash ON slip_parser_requests(file_hash)").Error; err != nil {
			return fmt.Errorf("failed to create slip parser file_hash index: %w", err)
		}
	}

	// Regional Passport Office indexes
	if tableExists("regional_passport_offices") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_regional_passport_offices_code ON regional_passport_offices(code)").Error; err != nil {
			return fmt.Errorf("failed to create regional passport office code index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_regional_passport_offices_name ON regional_passport_offices(name)").Error; err != nil {
			return fmt.Errorf("failed to create regional passport office name index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_regional_passport_offices_mobile ON regional_passport_offices(mobile)").Error; err != nil {
			return fmt.Errorf("failed to create regional passport office mobile index: %w", err)
		}
	}

	// Parcel Booking indexes (only if table exists)
	if tableExists("parcel_bookings") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_user_id ON parcel_bookings(user_id)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking user_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_barcode ON parcel_bookings(barcode)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking barcode index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_item_id ON parcel_bookings(item_id)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking item_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_phone ON parcel_bookings(phone)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking phone index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_post_code ON parcel_bookings(post_code)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking post_code index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_current_status ON parcel_bookings(current_status)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking current_status index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_service_type ON parcel_bookings(service_type)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking service_type index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_bookings_created_at ON parcel_bookings(created_at)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking created_at index: %w", err)
		}
	}

	// Parcel Booking Status Event indexes
	if tableExists("parcel_booking_status_events") {
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_booking_status_events_parcel_booking_id ON parcel_booking_status_events(parcel_booking_id)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking status event parcel_booking_id index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_booking_status_events_status ON parcel_booking_status_events(status)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking status event status index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_booking_status_events_created_by ON parcel_booking_status_events(created_by)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking status event created_by index: %w", err)
		}
		if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_parcel_booking_status_events_created_at ON parcel_booking_status_events(created_at)").Error; err != nil {
			return fmt.Errorf("failed to create parcel booking status event created_at index: %w", err)
		}
	}

	// Fix parcel booking foreign key constraint
	if err := fixParcelBookingForeignKeyConstraints(); err != nil {
		logger.Warning("Failed to fix parcel booking foreign key constraints: " + err.Error())
	}

	return nil
}

// fixParcelBookingForeignKeyConstraints fixes the foreign key constraint issue
func fixParcelBookingForeignKeyConstraints() error {
	if !tableExists("parcel_booking_status_events") || !tableExists("parcel_bookings") {
		return nil // Tables don't exist, no need to fix constraints
	}

	// Check if the problematic constraint exists
	var constraintExists bool
	checkSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE constraint_name = 'fk_parcel_booking_status_events_parcel_booking'
			AND table_name = 'parcel_booking_status_events'
		)`

	if err := DB.Raw(checkSQL).Scan(&constraintExists).Error; err != nil {
		return fmt.Errorf("failed to check constraint existence: %w", err)
	}

	if constraintExists {
		// Drop the existing restrictive constraint
		dropSQL := "ALTER TABLE parcel_booking_status_events DROP CONSTRAINT IF EXISTS fk_parcel_booking_status_events_parcel_booking"
		if err := DB.Exec(dropSQL).Error; err != nil {
			return fmt.Errorf("failed to drop existing constraint: %w", err)
		}
	}

	// Create new constraint without CASCADE/RESTRICT behavior
	createSQL := `
		ALTER TABLE parcel_booking_status_events 
		ADD CONSTRAINT fk_parcel_booking_status_events_parcel_booking 
		FOREIGN KEY (parcel_booking_id) REFERENCES parcel_bookings(id)`

	if err := DB.Exec(createSQL).Error; err != nil {
		// If constraint already exists with this name, ignore the error
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create new constraint: %w", err)
		}
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
				  FOREIGN KEY (delivery_address_id) REFERENCES addresses(id)
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
