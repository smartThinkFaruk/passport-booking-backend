package bag

type BranchMappingRequest struct {
	Username     string `json:"username"`
	BranchCode   string `json:"branch_code"`
	Relationship string `json:"relationship"`
}

type CreateBagRequest struct {
	BagCategory    string `json:"bag_category"`
	BagID          string `json:"bag_id"`
	BagType        string `json:"bag_type"`
	DestOfficeCode string `json:"dest_office_code"`
	RMSInstruction string `json:"rms_instruction"`
}

type AddItemRequest struct {
	OrderId string `json:"order_id"`
	BagID   string `json:"bag_id"`
	ItemID  string `json:"item_id"`
	BagType string `json:"bag_type"`
	Index   int    `json:"index"`
}

type BookingRequest struct {
	FromNumber      string  `json:"form_number"`
	AdPodID         string  `json:"ad_pod_id"`
	ArticleDesc     string  `json:"article_desc"`
	ArticlePrice    int     `json:"article_price"`
	Barcode         string  `json:"barcode"`
	CityPostStatus  string  `json:"city_post_status"`
	DeliveryBranch  string  `json:"delivery_branch"`
	EmtsBranchCode  string  `json:"emts_branch_code"`
	Height          int     `json:"height"`
	HndDevice       string  `json:"hnd_device"`
	ImagePod        string  `json:"image_pod"`
	ImageSrc        string  `json:"image_src"`
	InsurancePrice  string  `json:"insurance_price"`
	IsBulkMail      string  `json:"is_bulk_mail"`
	IsCharge        string  `json:"is_charge"`
	IsCityPost      string  `json:"is_city_post"`
	IsInternational bool    `json:"is_international"`
	IsStation       string  `json:"is_station"`
	Length          int     `json:"length"`
	ServiceName     string  `json:"service_name"`
	SetAd           string  `json:"set_ad"`
	VasType         string  `json:"vas_type"`
	VpAmount        string  `json:"vp_amount"`
	VpService       string  `json:"vp_service"`
	Weight          int     `json:"weight"`
	Width           int     `json:"width"`
	Receiver        Address `json:"receiver"`
	Sender          Address `json:"sender"`
}

type Address struct {
	AddressType   string `json:"address_type"`
	Country       string `json:"country"`
	District      string `json:"district"`
	Division      string `json:"division"`
	PhoneNumber   string `json:"phone_number"`
	PoliceStation string `json:"police_station"`
	PostOffice    string `json:"post_office"`
	StreetAddress string `json:"street_address"`
	UserUUID      string `json:"user_uuid"`
	Username      string `json:"username"`
	Zone          string `json:"zone"`
}

type Booking struct {
	ID           uint   `gorm:"primaryKey"`
	AppOrOrderID string `gorm:"column:app_or_order_id"`
	// Add other fields as needed
}

type CloseBagRequest struct {
	BagID string `json:"bag_id"`
}
