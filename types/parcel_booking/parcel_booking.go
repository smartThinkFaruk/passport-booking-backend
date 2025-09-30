package parcel_booking

// StoreParcelBookingRequest represents the request structure for storing parcel booking
type StoreParcelBookingRequest struct {
	RpoAddress string `json:"rpo_address" validate:"required"`
	Phone      string `json:"phone" validate:"required"`
	PostCode   string `json:"post_code" validate:"required"`
	RpoName    string `json:"rpo_name" validate:"required"`
}

// StorePendingBookingRequest represents the request structure for updating parcel booking to pending status
type StorePendingBookingRequest struct {
	Barcode string `json:"barcode" validate:"required"`
}
