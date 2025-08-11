package main

import (
	"fmt"
	"os"

	"passport-booking/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run tools/migrate.go migrate       - Run dynamic migrations")
		fmt.Println("  go run tools/migrate.go generate file.sql - Generate migration file")
		return
	}

	command := os.Args[1]

	switch command {
	case "migrate":
		fmt.Println("🚀 Running dynamic database migrations...")
		if err := database.RunDynamicMigration(); err != nil {
			fmt.Printf("❌ Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Migration completed successfully!")

	case "generate":
		if len(os.Args) < 3 {
			fmt.Println("Please provide a filename for the migration file")
			fmt.Println("Example: go run tools/migrate.go generate migration.sql")
			return
		}

		filename := os.Args[2]
		fmt.Printf("📝 Generating migration file: %s\n", filename)

		if err := database.GenerateMigrationFile(filename); err != nil {
			fmt.Printf("❌ Failed to generate migration file: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: migrate, generate")
	}
}
