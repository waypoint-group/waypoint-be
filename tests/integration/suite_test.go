//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
	"github.com/waypoint-group/waypoint-be/tests/testlib"
)

var testEnv *testlib.TestEnvironment

func TestMain(m *testing.M) {
	var err error
	testEnv, err = testlib.NewEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create test environment: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	if err := testEnv.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close test environment: %v\n", err)
		code = 1
	}
	os.Exit(code)
}

func newWaypoint(t *testing.T) *testlib.TestWaypoint {
	t.Helper()

	cfg := waypoint.Config{
		JWT: testEnv.Keycloak().JWTConfig,
		// Always migrate in tests.
		Migrate: true,
	}

	w, err := testEnv.CreateWaypointInstance(&cfg)
	if err != nil {
		t.Fatalf("start Waypoint: %v", err)
	}

	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("close Waypoint fixture: %v", err)
		}
	})

	return w
}
