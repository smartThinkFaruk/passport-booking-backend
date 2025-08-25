package booking_event

import (
	bookingModel "passport-booking/models/booking"
	"gorm.io/gorm"
)

// SnapshotBookingToEvent writes a full snapshot of a Booking row into BookingEvent with the given event type.
func SnapshotBookingToEvent(tx *gorm.DB, b *bookingModel.Booking, eventType string, updatedBy string) error {
	// Make sure relateds are present for event row (User, AddressInfo)
	// If caller already preloaded, these will be filled; else we fetch minimal required ids.
	if err := tx.Preload("User").Preload("AddressInfo").First(b, b.ID).Error; err != nil {
		return err
	}

	ev := bookingModel.BookingEvent{
		UserID:       b.UserID,
		User:         b.User, // optional; gorm will set by ID
		AppOrOrderID: b.AppOrOrderID,
		CurrentBagID: b.CurrentBagID,
		Barcode:      b.Barcode,
		Name:         b.Name,
		FatherName:   b.FatherName,
		MotherName:   b.MotherName,
		Phone:        b.Phone,

		ReceiverName: b.ReceiverName,
		DeliveryPhone: b.DeliveryPhone,

		DeliveryPhoneAppliedVerified:       b.DeliveryPhoneAppliedVerified,
		DeliveryPhoneAppliedOTPEncrypted:   b.DeliveryPhoneAppliedOTPEncrypted,
		DeliveryPhoneConfirmedVerified:     b.DeliveryPhoneConfirmedVerified,
		DeliveryPhoneConfirmedOTPEncrypted: b.DeliveryPhoneConfirmedOTPEncrypted,

		Address:               b.Address,
		EmergencyContactName:  b.EmergencyContactName,
		EmergencyContactPhone: b.EmergencyContactPhone,
		DeliveryBranchCode:	b.DeliveryBranchCode,

		AddressID:   b.AddressID,
		AddressInfo: b.AddressInfo,

		Status:      b.Status,
		BookingDate: b.BookingDate,
		CreatedBy:   b.CreatedBy,
		CreatedAt:   b.CreatedAt,
		UpdatedBy:   updatedBy,
		UpdatedAt:   b.UpdatedAt,
		DeletedAt:   b.DeletedAt,

		EventType: eventType,
	}

	return tx.Create(&ev).Error
}
