package seeders

import (
	"log"

	"gorm.io/gorm"
	"passport-booking/models/regional_passport_office"
)

func SeedRegionalPassportOffices(db *gorm.DB) {
	log.Printf("🔍 Checking regional passport offices data integrity...")

	offices := []regional_passport_office.RegionalPassportOffice{
		{Code: "4000", Name: "AGARGAON", Address: "Regional Passport Office, AGARGAON", Mobile: "01733393323"},
		{Code: "4002", Name: "BARISAL", Address: "Regional Passport Office, BARISAL", Mobile: "01733393374"},
		{Code: "4100", Name: "BOGURA", Address: "Regional Passport Office, BOGURA", Mobile: "01733393342"},
		{Code: "4101", Name: "BRAHMANBARIA", Address: "Regional Passport Office, BRAHMANBARIA", Mobile: "01733393322"},
		{Code: "4200", Name: "BAGERHAT", Address: "Regional Passport Office, BAGERHAT", Mobile: "01733393368"},
		{Code: "4201", Name: "BANDARBAN", Address: "Regional Passport Office, BANDARBAN", Mobile: "01733393359"},
		{Code: "4203", Name: "BHOLA", Address: "Regional Passport Office, BHOLA", Mobile: "01733393376"},
		{Code: "4204", Name: "BARGUNA", Address: "Regional Passport Office, BARGUNA", Mobile: "01733393378"},
		{Code: "4107", Name: "FENI", Address: "Regional Passport Office, FENI", Mobile: "01733393353"},
		{Code: "4102", Name: "CHANDGAON (Ctg)", Address: "Regional Passport Office, CHANDGAON (Ctg)", Mobile: "01733393350"},
		{Code: "4103", Name: "CHANDPUR", Address: "Regional Passport Office, CHANDPUR", Mobile: "01733393355"},
		{Code: "4205", Name: "CHAPAINAWABGANJ", Address: "Regional Passport Office, CHAPAINAWABGANJ", Mobile: "01733393388"},
		{Code: "4206", Name: "CHUADANGA", Address: "Regional Passport Office, CHUADANGA", Mobile: "01733393373"},
		{Code: "4003", Name: "COMILLA", Address: "Regional Passport Office, COMILLA", Mobile: "01733393352"},
		{Code: "4104", Name: "COX'S BAZAR", Address: "Regional Passport Office, COX'S BAZAR", Mobile: "01733393354"},
		{Code: "4105", Name: "DINAJPUR", Address: "Regional Passport Office, DINAJPUR", Mobile: "01733393358"},
		{Code: "4106", Name: "FARIDPUR", Address: "Regional Passport Office, FARIDPUR", Mobile: "01733393342"},
		{Code: "4217", Name: "KHAGRACHORI", Address: "Regional Passport Office, KHAGRACHORI", Mobile: "01733393342"},
		{Code: "4211", Name: "GAIBANDHA", Address: "Regional Passport Office, GAIBANDHA", Mobile: "01733393390"},
		{Code: "4212", Name: "GAZIPUR", Address: "Regional Passport Office, GAZIPUR", Mobile: "01733393337"},
		{Code: "4108", Name: "GOPALGANJ", Address: "Regional Passport Office, GOPALGANJ", Mobile: "01733393346"},
		{Code: "4109", Name: "HABIGANJ", Address: "Regional Passport Office, HABIGANJ", Mobile: "01733393363"},
		{Code: "4213", Name: "JAMALPUR", Address: "Regional Passport Office, JAMALPUR", Mobile: "01733393344"},
		{Code: "4004", Name: "JATRABARI", Address: "Regional Passport Office, JATRABARI", Mobile: "01733393327"},
		{Code: "4110", Name: "JESSORE", Address: "Regional Passport Office, JESSORE", Mobile: "01733393365"},
		{Code: "4214", Name: "JHALOKATI", Address: "Regional Passport Office, JHALOKATI", Mobile: "01733393375"},
		{Code: "4215", Name: "JHENAIDAH", Address: "Regional Passport Office, JHENAIDAH", Mobile: "01733393366"},
		{Code: "4216", Name: "JOYPURHAT", Address: "Regional Passport Office, JOYPURHAT", Mobile: "01733393383"},
		{Code: "4223", Name: "MEHERPUR", Address: "Regional Passport Office, MEHERPUR", Mobile: "01733393371"},
		{Code: "4114", Name: "MOULOVIBAZAR", Address: "Regional Passport Office, MOULOVIBAZAR", Mobile: "01733393362"},
		{Code: "4005", Name: "KHULNA", Address: "Regional Passport Office, KHULNA", Mobile: "01733393364"},
		{Code: "4111", Name: "KISHOREGANJ", Address: "Regional Passport Office, KISHOREGANJ", Mobile: "01733393340"},
		{Code: "4218", Name: "KURIGRAM", Address: "Regional Passport Office, KURIGRAM", Mobile: "01733393395"},
		{Code: "4112", Name: "KUSHTIA", Address: "Regional Passport Office, KUSHTIA", Mobile: "01733393372"},
		{Code: "4219", Name: "LAKSHMIPUR", Address: "Regional Passport Office, LAKSHMIPUR", Mobile: "01733393357"},
		{Code: "4220", Name: "LALMONIRHAT", Address: "Regional Passport Office, LALMONIRHAT", Mobile: "01733393394"},
		{Code: "4221", Name: "MADARIPUR", Address: "Regional Passport Office, MADARIPUR", Mobile: "01733393347"},
		{Code: "4222", Name: "MAGURA", Address: "Regional Passport Office, MAGURA", Mobile: "01733393369"},
		{Code: "4113", Name: "MANIKGANJ", Address: "Regional Passport Office, MANIKGANJ", Mobile: "01733393335"},
		{Code: "4006", Name: "MANSURABAD(Ctg)", Address: "Regional Passport Office, MANSURABAD(Ctg)", Mobile: "01733393349"},
		{Code: "4008", Name: "NOAKHALI", Address: "Regional Passport Office, NOAKHALI", Mobile: "01733393381"},
		{Code: "4118", Name: "PABNA", Address: "Regional Passport Office, PABNA", Mobile: "01733393386"},
		{Code: "4115", Name: "MUNSHIGANJ", Address: "Regional Passport Office, MUNSHIGANJ", Mobile: "01733393339"},
		{Code: "4007", Name: "MYMENSINGH", Address: "Regional Passport Office, MYMENSINGH", Mobile: "01733393334"},
		{Code: "4228", Name: "NAOGAON", Address: "Regional Passport Office, NAOGAON", Mobile: "01733393387"},
		{Code: "4227", Name: "NARAIL", Address: "Regional Passport Office, NARAIL", Mobile: "01733393370"},
		{Code: "4116", Name: "NARAYANGONJ", Address: "Regional Passport Office, NARAYANGONJ", Mobile: "01733393336"},
		{Code: "4117", Name: "NARSHINGHDI", Address: "Regional Passport Office, NARSHINGHDI", Mobile: "01733393397"},
		{Code: "4224", Name: "NATORE", Address: "Regional Passport Office, NATORE", Mobile: "01733393385"},
		{Code: "4225", Name: "NETROKONA", Address: "Regional Passport Office, NETROKONA", Mobile: "01733393348"},
		{Code: "4226", Name: "NILPHAMARI", Address: "Regional Passport Office, NILPHAMARI", Mobile: "01733393393"},
		{Code: "4233", Name: "SHERPUR", Address: "Regional Passport Office, SHERPUR", Mobile: "01733393341"},
		{Code: "4122", Name: "SIRAJGANJ", Address: "Regional Passport Office, SIRAJGANJ", Mobile: "01733393384"},
		{Code: "4235", Name: "SUNAMGANJ", Address: "Regional Passport Office, SUNAMGANJ", Mobile: "01733393396"},
		{Code: "4202", Name: "BANGLADESH SECRETARIATE", Address: "Regional Passport Office, BANGLADESH SECRETARIATE", Mobile: "01732436080"},
		{Code: "4119", Name: "PATUAKHALI", Address: "Regional Passport Office, PATUAKHALI", Mobile: "01733393377"},
		{Code: "4229", Name: "PANCHAGAR", Address: "Regional Passport Office, PANCHAGAR", Mobile: "01733393391"},
		{Code: "4230", Name: "PIROJPUR", Address: "Regional Passport Office, PIROJPUR", Mobile: "01733393379"},
		{Code: "4231", Name: "RAJBARI", Address: "Regional Passport Office, RAJBARI", Mobile: "01733393343"},
		{Code: "4009", Name: "RAJSHAHI", Address: "Regional Passport Office, RAJSHAHI", Mobile: "01733393380"},
		{Code: "4120", Name: "RANGAMATI", Address: "Regional Passport Office, RANGAMATI", Mobile: "01733393356"},
		{Code: "4121", Name: "RANGPUR", Address: "Regional Passport Office, RANGPUR", Mobile: "01733393389"},
		{Code: "4232", Name: "SATKHIRA", Address: "Regional Passport Office, SATKHIRA", Mobile: "01733393367"},
		{Code: "4234", Name: "SHARIATPUR", Address: "Regional Passport Office, SHARIATPUR", Mobile: "01733393345"},
		{Code: "4010", Name: "SYLHET", Address: "Regional Passport Office, SYLHET", Mobile: "01733393361"},
		{Code: "4123", Name: "TANGAIL", Address: "Regional Passport Office, TANGAIL", Mobile: "01733393338"},
		{Code: "4236", Name: "THAKURGAON", Address: "Regional Passport Office, THAKURGAON", Mobile: "01733393392"},
		{Code: "4011", Name: "UTTARA", Address: "Regional Passport Office, UTTARA", Mobile: "01733393328"},
		{Code: "4210", Name: "DHAKA WEST", Address: "Regional Passport Office, DHAKA WEST", Mobile: "01717998857"},
		{Code: "4209", Name: "DHAKA EAST", Address: "Regional Passport Office, DHAKA EAST", Mobile: "01718612234"},
	}

	// Get all existing office codes from database
	var existingCodes []string
	if err := db.Model(&regional_passport_office.RegionalPassportOffice{}).Pluck("code", &existingCodes).Error; err != nil {
		log.Printf("❌ Failed to fetch existing office codes: %v", err)
		return
	}

	// Create a map for faster lookup of existing codes
	existingCodesMap := make(map[string]bool)
	for _, code := range existingCodes {
		existingCodesMap[code] = true
	}

	// Find missing offices
	var missingOffices []regional_passport_office.RegionalPassportOffice
	for _, office := range offices {
		if !existingCodesMap[office.Code] {
			missingOffices = append(missingOffices, office)
		}
	}

	// Report status
	totalExpected := len(offices)
	totalExisting := len(existingCodes)
	totalMissing := len(missingOffices)

	log.Printf("📊 Data integrity check:")
	log.Printf("   Expected offices: %d", totalExpected)
	log.Printf("   Existing offices: %d", totalExisting)
	log.Printf("   Missing offices: %d", totalMissing)

	// If no missing data, we're done
	if totalMissing == 0 {
		log.Printf("✅ All regional passport offices are already present. No seeding needed.")
		return
	}

	// Seed missing data
	log.Printf("🌱 Seeding %d missing regional passport offices...", totalMissing)

	successCount := 0
	failureCount := 0

	for _, office := range missingOffices {
		if err := db.Create(&office).Error; err != nil {
			log.Printf("❌ Failed to seed office %s (%s): %v", office.Name, office.Code, err)
			failureCount++
		} else {
			log.Printf("✅ Added: %s (%s)", office.Name, office.Code)
			successCount++
		}
	}

	log.Printf("🎉 Seeding completed! Successfully inserted %d offices, %d failures", successCount, failureCount)

	// Final verification
	var finalCount int64
	if err := db.Model(&regional_passport_office.RegionalPassportOffice{}).Count(&finalCount).Error; err == nil {
		log.Printf("📈 Database now contains %d regional passport offices", finalCount)
	}
}
