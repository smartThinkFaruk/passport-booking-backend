package booking

import "fmt"

// BookingCreateRequest represents the request payload for creating a booking
type BookingCreateRequest struct {
	AppOrOrderID          string `json:"app_or_order_id" validate:"required,min=1,max=255"`
	CurrentBagID          string `json:"current_bag_id" validate:"omitempty,max=255"`
	Barcode               string `json:"barcode" validate:"omitempty,max=255"`
	Name                  string `json:"name" validate:"required,min=1,max=255"`
	FatherName            string `json:"father_name" validate:"required,min=1,max=255"`
	MotherName            string `json:"mother_name" validate:"required,min=1,max=255"`
	Phone                 string `json:"phone" validate:"required,min=1,max=20"`
	Address               string `json:"address" validate:"required,min=1"`
	EmergencyContactName  string `json:"emergency_contact_name" validate:"omitempty,max=255"`
	EmergencyContactPhone string `json:"emergency_contact_phone" validate:"omitempty,max=20"`
	Division              string `json:"division" validate:"required,min=1,max=255"`
	District              string `json:"district" validate:"required,min=1,max=255"`
	PoliceStation         string `json:"police_station" validate:"required,min=1,max=255"`
	PostOffice            string `json:"post_office" validate:"required,min=1,max=255"`
	StreetAddress         string `json:"street_address" validate:"required,min=1,max=255"`
	AddressType           string `json:"address_type" validate:"required,oneof=home office"`
}

func (r *BookingCreateRequest) Validate() error {
	if r.AppOrOrderID == "" {
		return fmt.Errorf("app_or_order_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.FatherName == "" {
		return fmt.Errorf("father_name is required")
	}
	if r.MotherName == "" {
		return fmt.Errorf("mother_name is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if r.Address == "" {
		return fmt.Errorf("address is required")
	}
	if r.Division == "" {
		return fmt.Errorf("division is required")
	}
	if r.District == "" {
		return fmt.Errorf("district is required")
	}
	if r.PoliceStation == "" {
		return fmt.Errorf("police_station is required")
	}
	if r.PostOffice == "" {
		return fmt.Errorf("post_office is required")
	}
	if r.StreetAddress == "" {
		return fmt.Errorf("street_address is required")
	}
	if r.AddressType == "" {
		return fmt.Errorf("address_type is required")
	}
	if r.AddressType != "home" && r.AddressType != "office" {
		return fmt.Errorf("address_type must be either 'home' or 'office'")
	}
	return nil
}
