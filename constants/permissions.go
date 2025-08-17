package constants

// Organization permissions
const (
	// Admin permissions
	PermEkdakDPMGFull     = "ekdak.dpmg.full-permit"
	PermPassportDPMGFull  = "passport-booking.dpmg.full-permit"
	PermPostOfficeFull    = "passport-booking.post-master.full-permit"
	PermOrgSupervisorFull = "passport-booking.supervisor.full-permit"
	PermOperatorHasFull   = "passport-booking.operator.has-full-permit"
	PermAgentHasFull      = "passport-booking.agent.full-permit"
	PermPostmanFull       = "passport-booking.postman.has-full-permit"

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
