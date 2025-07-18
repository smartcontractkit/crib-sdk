package cmd

import (
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestMain(m *testing.M) {
	v := m.Run()

	// After all tests have run `go-snaps` can check for unused snapshots
	snaps.Clean(m)

	os.Exit(v)
}
