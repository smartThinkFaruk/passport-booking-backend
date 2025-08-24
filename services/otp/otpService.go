package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"passport-booking/httpServices/sms"
	"passport-booking/models/otp"
	"time"

	"gorm.io/gorm"
)

// Service handles OTP operations
type Service struct {
	DB         *gorm.DB
	SMSService *sms.SMSService
}

// NewOTPService creates a new OTP service
func NewOTPService(db *gorm.DB) *Service {
	return &Service{
		DB:         db,
		SMSService: sms.NewSMSService(),
	}
}

// GenerateOTP generates a random 6-digit OTP
func (s *Service) GenerateOTP() (string, error) {
	max := big.NewInt(999999)
	min := big.NewInt(100000)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Ensure the number is at least 6 digits
	n.Add(n, min)
	if n.Cmp(max) > 0 {
		n.Sub(n, max)
		n.Add(n, min)
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

// SendOTP creates and stores an OTP for the given phone number with retry handling (for non-booking purposes)
func (s *Service) SendOTP(phone string, purpose otp.OTPPurpose) (*otp.OTP, error) {
	// For non-booking OTPs, we'll use booking ID 0 as a default
	defaultBookingID := uint(0)
	return s.SendOTPWithBookingID(phone, purpose, &defaultBookingID)
}

// SendOTPWithBookingID creates and stores an OTP for the given phone number with optional booking ID
func (s *Service) SendOTPWithBookingID(phone string, purpose otp.OTPPurpose, bookingID *uint) (*otp.OTP, error) {
	// Ensure we have a valid booking ID
	if bookingID == nil {
		return nil, fmt.Errorf("booking ID is required for OTP generation")
	}

	// Check if there's an existing active OTP for this phone and purpose
	existingOTP, err := s.GetOTPStatus(phone, purpose)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing OTP: %w", err)
	}

	// If there's an existing OTP that hasn't expired yet, don't send new OTP
	if existingOTP != nil && !existingOTP.IsExpired() && !existingOTP.IsUsed {
		return nil, fmt.Errorf("an OTP for this phone number is still active and hasn't expired yet. Please wait until it expires or use the existing OTP")
	}

	// If there's an expired OTP, mark it as used to clean up
	if existingOTP != nil && existingOTP.IsExpired() && !existingOTP.IsUsed {
		existingOTP.IsUsed = true
		if err := s.DB.Save(existingOTP).Error; err != nil {
			// Log error but continue
			fmt.Printf("Failed to mark expired OTP as used: %v\n", err)
		}
	}

	// If there's a valid existing OTP, return it (don't generate a new one)
	if existingOTP != nil && existingOTP.IsValid() {
		return existingOTP, nil
	}

	// Check if user is blocked due to too many attempts
	if existingOTP != nil && existingOTP.IsCurrentlyBlocked() {
		blockTime := "permanently"
		if existingOTP.BlockedUntil != nil {
			blockTime = fmt.Sprintf("until %s", existingOTP.BlockedUntil.Format("15:04:05"))
		}
		return nil, fmt.Errorf("OTP requests are blocked %s due to too many failed attempts", blockTime)
	}

	// Generate OTP code
	otpCode, err := s.GenerateOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Invalidate any existing unused OTPs for this phone and purpose
	err = s.DB.Model(&otp.OTP{}).
		Where("phone = ? AND purpose = ? AND is_used = false", phone, purpose).
		Update("is_used", true).Error
	if err != nil {
		return nil, fmt.Errorf("failed to invalidate existing OTPs: %w", err)
	}

	// Create new OTP record with retry settings
	newOTP := &otp.OTP{
		BookingID:  *bookingID,
		Phone:      phone,
		OTPCode:    otpCode,
		Purpose:    purpose,
		IsUsed:     false,
		RetryCount: 0,
		MaxRetries: 3, // Default max retries
		IsBlocked:  false,
		ExpiresAt:  time.Now().Add(5 * time.Minute), // 5 minutes expiry
	}

	if err := s.DB.Create(newOTP).Error; err != nil {
		return nil, fmt.Errorf("failed to create OTP record: %w", err)
	}

	// Send OTP via SMS
	if err := s.SMSService.SendOTP(phone, otpCode); err != nil {
		// Log the error but don't fail the OTP creation
		// The OTP is still valid and can be used for testing/fallback
		fmt.Printf("Failed to send OTP SMS to %s: %v\n", phone, err)
		fmt.Printf("OTP for %s: %s (Purpose: %s) - SMS delivery failed, showing for testing\n", phone, otpCode, purpose)
	} else {
		fmt.Printf("OTP sent via SMS to %s (Purpose: %s)\n", phone, purpose)
	}

	return newOTP, nil
}

// VerifyOTP verifies the provided OTP code for the given phone number and purpose with retry handling
func (s *Service) VerifyOTP(phone, otpCode string, purpose otp.OTPPurpose) (bool, error) {
	var otpRecord otp.OTP

	err := s.DB.Where("phone = ? AND purpose = ? AND is_used = false",
		phone, purpose).
		Order("created_at DESC").
		First(&otpRecord).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // No OTP found
		}
		return false, fmt.Errorf("failed to find OTP record: %w", err)
	}

	// Check if OTP is blocked
	if otpRecord.IsCurrentlyBlocked() {
		blockTime := "permanently"
		if otpRecord.BlockedUntil != nil {
			blockTime = fmt.Sprintf("until %s", otpRecord.BlockedUntil.Format("15:04:05"))
		}
		return false, fmt.Errorf("OTP verification is blocked %s due to too many failed attempts", blockTime)
	}

	// Check if OTP has expired
	if otpRecord.IsExpired() {
		return false, fmt.Errorf("OTP has expired")
	}

	// Check if the OTP code matches
	if otpRecord.OTPCode != otpCode {
		// Increment retry count for failed attempt
		otpRecord.IncrementRetry()
		if err := s.DB.Save(&otpRecord).Error; err != nil {
			return false, fmt.Errorf("failed to update retry count: %w", err)
		}

		remainingAttempts := otpRecord.MaxRetries - otpRecord.RetryCount
		if remainingAttempts <= 0 {
			return false, fmt.Errorf("invalid OTP. Maximum attempts exceeded. OTP is now blocked")
		}
		return false, fmt.Errorf("invalid OTP. %d attempts remaining", remainingAttempts)
	}

	// OTP is valid, mark as used
	otpRecord.IsUsed = true
	if err := s.DB.Save(&otpRecord).Error; err != nil {
		return false, fmt.Errorf("failed to mark OTP as used: %w", err)
	}

	return true, nil
}

// VerifyOTPWithDetails verifies the provided OTP code and returns the OTP record details with retry handling
func (s *Service) VerifyOTPWithDetails(phone, otpCode string, purpose otp.OTPPurpose) (bool, *otp.OTP, error) {
	var otpRecord otp.OTP

	err := s.DB.Where("phone = ? AND purpose = ? AND is_used = false",
		phone, purpose).
		Order("created_at DESC").
		First(&otpRecord).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil // No OTP found
		}
		return false, nil, fmt.Errorf("failed to find OTP record: %w", err)
	}

	// Check if OTP is blocked
	if otpRecord.IsCurrentlyBlocked() {
		blockTime := "permanently"
		if otpRecord.BlockedUntil != nil {
			blockTime = fmt.Sprintf("until %s", otpRecord.BlockedUntil.Format("15:04:05"))
		}
		return false, &otpRecord, fmt.Errorf("OTP verification is blocked %s due to too many failed attempts", blockTime)
	}

	// Check if OTP has expired
	if otpRecord.IsExpired() {
		return false, &otpRecord, fmt.Errorf("OTP has expired")
	}

	// Check if the OTP code matches
	if otpRecord.OTPCode != otpCode {
		// Increment retry count for failed attempt
		otpRecord.IncrementRetry()
		if err := s.DB.Save(&otpRecord).Error; err != nil {
			return false, &otpRecord, fmt.Errorf("failed to update retry count: %w", err)
		}

		remainingAttempts := otpRecord.MaxRetries - otpRecord.RetryCount
		if remainingAttempts <= 0 {
			return false, &otpRecord, fmt.Errorf("invalid OTP. Maximum attempts exceeded. OTP is now blocked")
		}
		return false, &otpRecord, fmt.Errorf("invalid OTP. %d attempts remaining", remainingAttempts)
	}

	// OTP is valid, mark as used
	otpRecord.IsUsed = true
	if err := s.DB.Save(&otpRecord).Error; err != nil {
		return false, &otpRecord, fmt.Errorf("failed to mark OTP as used: %w", err)
	}

	return true, &otpRecord, nil
}

// CleanupExpiredOTPs removes expired OTP records from the database
func (s *Service) CleanupExpiredOTPs() error {
	return s.DB.Where("expires_at < ?", time.Now()).Delete(&otp.OTP{}).Error
}

// GetOTPStatus checks if there's a valid OTP for the given phone and purpose
func (s *Service) GetOTPStatus(phone string, purpose otp.OTPPurpose) (*otp.OTP, error) {
	var otpRecord otp.OTP

	err := s.DB.Where("phone = ? AND purpose = ? AND is_used = false AND expires_at > ?",
		phone, purpose, time.Now()).
		Order("created_at DESC").
		First(&otpRecord).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No valid OTP found
		}
		return nil, fmt.Errorf("failed to find OTP record: %w", err)
	}

	return &otpRecord, nil
}

// GetOTPRetryInfo returns retry information for a phone number and purpose
func (s *Service) GetOTPRetryInfo(phone string, purpose otp.OTPPurpose) (*OTPRetryInfo, error) {
	var otpRecord otp.OTP

	err := s.DB.Where("phone = ? AND purpose = ?", phone, purpose).
		Order("created_at DESC").
		First(&otpRecord).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No OTP exists, user can request new OTP
			return &OTPRetryInfo{
				CanRequestNewOTP: true,
				CanRetryOTP:      false,
				IsBlocked:        false,
				RemainingRetries: 3,
				BlockedUntil:     nil,
				Message:          "You can request a new OTP",
			}, nil
		}
		return nil, fmt.Errorf("failed to find OTP record: %w", err)
	}

	info := &OTPRetryInfo{
		CanRequestNewOTP: !otpRecord.IsCurrentlyBlocked() && (otpRecord.IsUsed || otpRecord.IsExpired()),
		CanRetryOTP:      otpRecord.CanRetry(),
		IsBlocked:        otpRecord.IsCurrentlyBlocked(),
		RemainingRetries: otpRecord.MaxRetries - otpRecord.RetryCount,
		BlockedUntil:     otpRecord.BlockedUntil,
	}

	// Set appropriate message
	if info.IsBlocked {
		if info.BlockedUntil != nil {
			info.Message = fmt.Sprintf("OTP verification is blocked until %s", info.BlockedUntil.Format("15:04:05"))
		} else {
			info.Message = "OTP verification is permanently blocked"
		}
	} else if info.CanRetryOTP {
		info.Message = fmt.Sprintf("You have %d attempts remaining", info.RemainingRetries)
	} else if info.CanRequestNewOTP {
		info.Message = "You can request a new OTP"
	} else {
		info.Message = "Current OTP is still valid"
	}

	return info, nil
}

// UnblockOTP manually unblocks an OTP for a phone number and purpose (admin function)
func (s *Service) UnblockOTP(phone string, purpose otp.OTPPurpose) error {
	var otpRecord otp.OTP

	err := s.DB.Where("phone = ? AND purpose = ? AND is_blocked = true", phone, purpose).
		Order("created_at DESC").
		First(&otpRecord).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("no blocked OTP found for phone %s", phone)
		}
		return fmt.Errorf("failed to find blocked OTP: %w", err)
	}

	// Reset the OTP retry state
	otpRecord.Reset()

	if err := s.DB.Save(&otpRecord).Error; err != nil {
		return fmt.Errorf("failed to unblock OTP: %w", err)
	}

	return nil
}

// CleanupExpiredBlocks removes expired blocks and resets retry counts
func (s *Service) CleanupExpiredBlocks() error {
	now := time.Now()

	// Find all OTPs that are blocked but the block period has expired
	var expiredBlocks []otp.OTP
	err := s.DB.Where("is_blocked = true AND blocked_until IS NOT NULL AND blocked_until < ?", now).
		Find(&expiredBlocks).Error

	if err != nil {
		return fmt.Errorf("failed to find expired blocks: %w", err)
	}

	// Reset each expired block
	for _, otpRecord := range expiredBlocks {
		otpRecord.Reset()
		if err := s.DB.Save(&otpRecord).Error; err != nil {
			// Log error but continue with other records
			fmt.Printf("Failed to reset expired block for OTP ID %d: %v\n", otpRecord.ID, err)
		}
	}

	return nil
}

// OTPRetryInfo contains information about OTP retry status
type OTPRetryInfo struct {
	CanRequestNewOTP bool       `json:"can_request_new_otp"`
	CanRetryOTP      bool       `json:"can_retry_otp"`
	IsBlocked        bool       `json:"is_blocked"`
	RemainingRetries int        `json:"remaining_retries"`
	BlockedUntil     *time.Time `json:"blocked_until,omitempty"`
	Message          string     `json:"message"`
}
