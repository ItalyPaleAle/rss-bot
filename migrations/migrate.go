package migrations

import (
	"fmt"
)

// Migrate runs the migration suite
func Migrate() {
	// This is as bad as it seems, but it looked weird to use some complex tool for something as simple
	// as creating a few tables for this small app.
	err := V1()

	if err != nil {
		panic(fmt.Sprintln("Error migrating the database to the latest schema", err))
	}
}
