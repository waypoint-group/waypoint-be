//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
	"github.com/waypoint-group/waypoint-be/tests/testlib"
)

var environment testlib.TestEnvironment

func TestMain(m *testing.M) {
	code := m.Run()
	if err := environment.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close test environment: %v\n", err)
		code = 1
	}
	os.Exit(code)
}

func newWaypoint(t *testing.T, authenticated bool) *testlib.TestWaypoint {
	t.Helper()

	cfg := waypoint.Config{}
	if authenticated {
		cfg.JWT = keycloak(t).JWTConfig
	}

	w, err := environment.CreateWaypointInstance(&cfg)
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

func keycloak(t *testing.T) *testlib.Keycloak {
	t.Helper()
	kc, err := environment.Keycloak()
	if err != nil {
		t.Fatalf("start Keycloak: %v", err)
	}
	return kc
}
