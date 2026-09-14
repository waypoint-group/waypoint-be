//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/waypoint-group/waypoint-be/tests/testlib"
)

func TestHealth_HealthCheckOk(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	response, err := uut.Request(t.Context(), http.MethodGet, "/healthz", nil)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", response.StatusCode)
	}
}

func TestHealth_ReadyCheckOk(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	response, err := uut.Request(t.Context(), http.MethodGet, "/readyz", nil)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", response.StatusCode)
	}
}
