package migrations

import (
	"fmt"
)

// Migrate runs the migration suite
func Migrate() {
	// A rather makeshift solution, but it works for our simple scenario
	err := V1()
	if err != nil {
		panic(fmt.Sprintln("Error migrating the database to V1", err))
	}
	err = V2()
	if err != nil {
		panic(fmt.Sprintln("Error migrating the database to V2", err))
	}
	err = V3()
	if err != nil {
		panic(fmt.Sprintln("Error migrating the database to V3", err))
	}
}
