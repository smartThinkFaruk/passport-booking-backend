package constants

// Organization permissions
const (
	// Admin permissions
	PermSuperAdminFull    = "passport-booking.super-admin.full-permit"
	PermEkdakDPMGFull     = "ekdak.dpmg.full-permit"
	PermPassportDPMGFull  = "passport-booking.dpmg.full-permit"
	PermPostOfficeFull    = "passport-booking.postmaster.full-permit"
	PermOrgSupervisorFull = "passport-booking.supervisor.full-permit"
	PermOperatorFull      = "passport-booking.operator.full-permit"
	PermAgentHasFull      = "passport-booking.agent.full-permit"
	PermPostmanFull       = "passport-booking.postman.full-permit"
	PermCustomerFull      = "passport-booking.customer.full-permit"

	// Special permissions
	PermAny = "any"
)

// Permission groups for convenience
var (
	OrganizationAdminPermissions = []string{
		PermEkdakDPMGFull,
		PermPassportDPMGFull,
		PermPostOfficeFull,
	}
)
