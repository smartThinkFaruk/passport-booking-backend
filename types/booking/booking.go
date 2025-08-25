package booking

import (
	"fmt"
)

type BookingCreateRequest struct {
	AppOrOrderID          string `json:"app_or_order_id" validate:"required,min=1,max=255"`
	Name                  string `json:"name" validate:"required,min=1,max=255"`
	FatherName            string `json:"father_name" validate:"required,min=1,max=255"`
	MotherName            string `json:"mother_name" validate:"required,min=1,max=255"`
	Phone                 string `json:"phone" validate:"required,phone"`
	Address               string `json:"address" validate:"required,min=1"`
	EmergencyContactName  string `json:"emergency_contact_name" validate:"omitempty,max=255"`
	EmergencyContactPhone string `json:"emergency_contact_phone" validate:"omitempty,phone"`
}

// BookingCreateRequest represents the request payload for creating a booking
type BookingStoreUpdateRequest struct {
	// DeliveryBranchCode required
	DeliveryBranchCode string `json:"delivery_branch_code" validate:"required,min=1,max=100"`
	ReceiverName  string `json:"receiver_name" validate:"omitempty,max=255"`
	Division      string `json:"division" validate:"required,min=1,max=255"`
	District      string `json:"district" validate:"required,min=1,max=255"`
	PoliceStation string `json:"police_station" validate:"required,min=1,max=255"`
	PostOffice    string `json:"post_office" validate:"required,min=1,max=255"`
	StreetAddress string `json:"street_address" validate:"required,min=1,max=255"`
	AddressType   string `json:"address_type" validate:"required,oneof=home office"`
}

// use first step validation
func (b BookingCreateRequest) Validate() error {
	if b.AppOrOrderID == "" {
		return fmt.Errorf("AppOrOrderID is required")
	}
	if b.Name == "" {
		return fmt.Errorf("name is required")
	}
	if b.FatherName == "" {
		return fmt.Errorf("FatherName is required")
	}
	if b.MotherName == "" {
		return fmt.Errorf("MotherName is required")
	}
	if b.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if b.Address == "" {
		return fmt.Errorf("address is required")
	}
	return nil
}

// use second step validation
func (b BookingStoreUpdateRequest) Validate() error {
	if b.DeliveryBranchCode == "" {
		return fmt.Errorf("deliveryBranchCode is required")	
	}
	if b.ReceiverName == "" {
		return fmt.Errorf("receiverName is required")
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
	if b.AddressType != "home" && b.AddressType != "office" {
		return fmt.Errorf("AddressType must be either 'home' or 'office'")
	}
	return nil
}
