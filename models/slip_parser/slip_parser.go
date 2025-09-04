package slip_parser

import (
	"time"

	"gorm.io/gorm"
)

// SlipParserRequest represents a passport slip parsing request
type SlipParserRequest struct {
	ID               uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestID        string `json:"request_id" gorm:"type:varchar(24);uniqueIndex;not null"` // 24 character unique ID
	OriginalFileName string `json:"original_file_name" gorm:"type:varchar(255);not null"`
	SavedFileName    string `json:"saved_file_name" gorm:"type:varchar(255);not null"`
	FileHash         string `json:"file_hash" gorm:"type:varchar(128);index"` // SHA256 hash
	FilePath         string `json:"file_path" gorm:"type:varchar(500);not null"`
	FileSize         int64  `json:"file_size" gorm:"not null"`
	MimeType         string `json:"mime_type" gorm:"type:varchar(100);not null"`
	Status           string `json:"status" gorm:"type:varchar(50);not null;default:'processing';index"` // processing, success, failed
	ProcessingTimeMs int64  `json:"processing_time_ms" gorm:"default:0"`

	// Parsed data fields
	AppOrOrderID          string `json:"app_or_order_id" gorm:"type:varchar(100);index;default:''"`
	Name                  string `json:"name" gorm:"type:varchar(255);default:''"`
	FatherName            string `json:"father_name" gorm:"type:varchar(255);default:''"`
	MotherName            string `json:"mother_name" gorm:"type:varchar(255);default:''"`
	Phone                 string `json:"phone" gorm:"type:varchar(20);index;default:''"`
	Address               string `json:"address" gorm:"type:text;default:''"`
	EmergencyContactName  string `json:"emergency_contact_name" gorm:"type:varchar(255);default:''"`
	EmergencyContactPhone string `json:"emergency_contact_phone" gorm:"type:varchar(20);default:''"`

	// Error information
	ErrorMessage string `json:"error_message" gorm:"type:text;default:''"`

	// Metadata
	IPAddress string `json:"ip_address" gorm:"type:varchar(45);index;default:''"` // Support IPv6
	UserAgent string `json:"user_agent" gorm:"type:text;default:''"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName returns the table name for SlipParserRequest
func (SlipParserRequest) TableName() string {
	return "slip_parser_requests"
}

// BeforeCreate hook to set default values
func (spr *SlipParserRequest) BeforeCreate(tx *gorm.DB) error {
	if spr.Status == "" {
		spr.Status = "processing"
	}
	return nil
}

// IsProcessing checks if the request is still processing
func (spr *SlipParserRequest) IsProcessing() bool {
	return spr.Status == "processing"
}

// IsSuccess checks if the request was successful
func (spr *SlipParserRequest) IsSuccess() bool {
	return spr.Status == "success"
}

// IsFailed checks if the request failed
func (spr *SlipParserRequest) IsFailed() bool {
	return spr.Status == "failed"
}

// MarkAsSuccess marks the request as successful and saves parsed data
func (spr *SlipParserRequest) MarkAsSuccess(db *gorm.DB, parsedData *SlipParserResponse) error {
	spr.Status = "success"
	spr.AppOrOrderID = parsedData.AppOrOrderID
	spr.Name = parsedData.Name
	spr.FatherName = parsedData.FatherName
	spr.MotherName = parsedData.MotherName
	spr.Phone = parsedData.Phone
	spr.Address = parsedData.Address
	spr.EmergencyContactName = parsedData.EmergencyContactName
	spr.EmergencyContactPhone = parsedData.EmergencyContactPhone
	spr.ProcessingTimeMs = parsedData.ProcessingTimeMs

	return db.Save(spr).Error
}

// MarkAsFailed marks the request as failed with error message
func (spr *SlipParserRequest) MarkAsFailed(db *gorm.DB, errorMsg string, processingTime int64) error {
	spr.Status = "failed"
	spr.ErrorMessage = errorMsg
	spr.ProcessingTimeMs = processingTime

	return db.Save(spr).Error
}

// SlipParserResponse represents the parsed data response
type SlipParserResponse struct {
	RequestID             string `json:"request_id"`
	AppOrOrderID          string `json:"app_or_order_id"`
	Name                  string `json:"name"`
	FatherName            string `json:"father_name"`
	MotherName            string `json:"mother_name"`
	Phone                 string `json:"phone"`
	Address               string `json:"address"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	ProcessingTimeMs      int64  `json:"processing_time_ms"`
}
