package booking

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type BookingCreateRequest struct {
	RequestID          string `json:"request_id" validate:"required,min=1,max=255"`
	DeliveryBranchCode string `json:"delivery_branch_code" validate:"required,min=1,max=100"`
	Division           string `json:"division" validate:"required,min=1,max=255"`
	District           string `json:"district" validate:"required,min=1,max=255"`
	PoliceStation      string `json:"police_station" validate:"required,min=1,max=255"`
	PostOffice         string `json:"post_office" validate:"required,min=1,max=255"`
	StreetAddress      string `json:"street_address" validate:"required,min=1,max=255"`
}

// BookingCreateRequest represents the request payload for creating a booking
type BookingStoreUpdateRequest struct {
	// DeliveryBranchCode required
	ID                 uint   `json:"id" validate:"required"`
	DeliveryBranchCode string `json:"delivery_branch_code" validate:"required,min=1,max=100"`
	Division           string `json:"division" validate:"required,min=1,max=255"`
	District           string `json:"district" validate:"required,min=1,max=255"`
	PoliceStation      string `json:"police_station" validate:"required,min=1,max=255"`
	PostOffice         string `json:"post_office" validate:"required,min=1,max=255"`
	StreetAddress      string `json:"street_address" validate:"required,min=1,max=255"`
}

// use first step validation
func (b BookingCreateRequest) Validate() error {
	if b.RequestID == "" {
		return fmt.Errorf("RequestID is required")
	}
	return nil
}

// use second step validation
func (b BookingStoreUpdateRequest) Validate() error {
	if b.DeliveryBranchCode == "" {
		return fmt.Errorf("deliveryBranchCode is required")
	}
	if b.Division == "" {
		return fmt.Errorf("division is required")
	}
	if b.District == "" {
		return fmt.Errorf("district is required")
	}
	if b.PoliceStation == "" {
		return fmt.Errorf("policeStation is required")
	}
	if b.PostOffice == "" {
		return fmt.Errorf("postOffice is required")
	}
	if b.StreetAddress == "" {
		return fmt.Errorf("streetAddress is required")
	}
	return nil
}

// BookingIndexRequest represents the request for listing bookings with pagination and filters
type BookingIndexRequest struct {
	Page     int    `json:"page" query:"page"`
	PerPage  int    `json:"per_page" query:"per_page"`
	FromDate string `json:"from_date" query:"from_date"` // Format: "26:8:2026 11:39:23" or "2026-08-26 11:39:23"
	ToDate   string `json:"to_date" query:"to_date"`     // Format: "26:8:2026 11:39:23" or "2026-08-26 11:39:23"
	Status   string `json:"status" query:"status"`       // booking status filter
}

// BookingIndexResponse represents the response for listing bookings with pagination
type BookingIndexResponse struct {
	Data       interface{}        `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
}

// Validate validates the BookingIndexRequest
func (b *BookingIndexRequest) Validate() error {
	// Set defaults
	if b.Page <= 0 {
		b.Page = 1
	}
	if b.PerPage <= 0 {
		b.PerPage = 10
	}
	if b.PerPage > 100 {
		b.PerPage = 100 // Maximum limit
	}

	// Validate status if provided
	if b.Status != "" {
		validStatuses := []string{"initial", "pre_booked", "booked", "return", "delivered"}
		isValid := false
		for _, status := range validStatuses {
			if b.Status == status {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid status. Valid values are: %s", strings.Join(validStatuses, ", "))
		}
	}

	// Validate date formats if provided
	if b.FromDate != "" {
		if _, err := b.ParseFromDate(); err != nil {
			return fmt.Errorf("invalid from_date format. Use 'DD:MM:YYYY HH:MM:SS' or 'YYYY-MM-DD HH:MM:SS'")
		}
	}

	if b.ToDate != "" {
		if _, err := b.ParseToDate(); err != nil {
			return fmt.Errorf("invalid to_date format. Use 'DD:MM:YYYY HH:MM:SS' or 'YYYY-MM-DD HH:MM:SS'")
		}
	}

	// Validate date range if both dates are provided
	if b.FromDate != "" && b.ToDate != "" {
		fromTime, _ := b.ParseFromDate()
		toTime, _ := b.ParseToDate()
		if fromTime.After(toTime) {
			return fmt.Errorf("from_date cannot be after to_date")
		}
	}

	return nil
}

// ParseFromDate parses the from_date string to time.Time
func (b *BookingIndexRequest) ParseFromDate() (time.Time, error) {
	return parseDateTime(b.FromDate)
}

// ParseToDate parses the to_date string to time.Time
func (b *BookingIndexRequest) ParseToDate() (time.Time, error) {
	return parseDateTime(b.ToDate)
}

// parseDateTime parses date string in multiple formats
func parseDateTime(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try format: "26:8:2026 11:39:23" (DD:MM:YYYY HH:MM:SS)
	if strings.Contains(dateStr, ":") && len(strings.Split(dateStr, " ")) == 2 {
		parts := strings.Split(dateStr, " ")
		if len(parts) == 2 {
			datePart := parts[0]
			timePart := parts[1]

			dateParts := strings.Split(datePart, ":")
			if len(dateParts) == 3 {
				day, err1 := strconv.Atoi(dateParts[0])
				month, err2 := strconv.Atoi(dateParts[1])
				year, err3 := strconv.Atoi(dateParts[2])

				if err1 == nil && err2 == nil && err3 == nil {
					// Convert to standard format and parse
					standardFormat := fmt.Sprintf("%04d-%02d-%02d %s", year, month, day, timePart)
					return time.Parse("2006-01-02 15:04:05", standardFormat)
				}
			}
		}
	}

	// Try standard formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format")
}

// GetOffset calculates the offset for pagination
func (b *BookingIndexRequest) GetOffset() int {
	return (b.Page - 1) * b.PerPage
}

// GetLimit returns the limit for pagination
func (b *BookingIndexRequest) GetLimit() int {
	return b.PerPage
}
