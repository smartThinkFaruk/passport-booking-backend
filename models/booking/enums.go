package booking

// Helper methods for BookingStatus
func (bs BookingStatus) String() string {
	return string(bs)
}

func (bs BookingStatus) IsValid() bool {
	switch bs {
	case BookingStatusInitial, BookingStatusPreBooked, BookingStatusBooked, BookingStatusReceivedByPostman, BookingStatusReturn, BookingStatusDelivered:
		return true
	default:
		return false
	}
}

// IsCompleted returns true if the booking is in a completed state
func (bs BookingStatus) IsCompleted() bool {
	return bs == BookingStatusDelivered || bs == BookingStatusReturn
}

// CanBePrinted returns true if the booking can be printed
func (bs BookingStatus) CanBePrinted() bool {
	return bs == BookingStatusPreBooked || bs == BookingStatusBooked
}

// CanBeUpdated returns true if the booking status can be updated
func (bs BookingStatus) CanBeUpdated() bool {
	return bs != BookingStatusDelivered && bs != BookingStatusReturn
}

// GetAllBookingStatuses returns all valid booking statuses
func GetAllBookingStatuses() []BookingStatus {
	return []BookingStatus{
		BookingStatusInitial,
		BookingStatusPreBooked,
		BookingStatusBooked,
		BookingStatusReceivedByPostMaster,
		BookingStatusReceivedByPostman,
		BookingStatusReturn,
		BookingStatusDelivered,
	}
}
