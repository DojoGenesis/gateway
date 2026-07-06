package server

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	// Lower bcrypt cost for tests to avoid suite-wide timeout.
	bcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}
