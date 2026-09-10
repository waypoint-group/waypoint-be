//go:build integration

package httpapi_test

import (
	"testing"

	"github.com/waypoint-group/waypoint-be/internal/testlib"
)

func TestHealthCheck(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer func() {
		_ = uut.Close(t.Context())
	}()

	response, err := uut.Get("/healthz")
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", response.StatusCode)
	}
}

func TestReadyCheck(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer func() {
		_ = uut.Close(t.Context())
	}()

	response, err := uut.Get("/readyz")
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() {
		_ = uut.Close(t.Context())
	}()

	if response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", response.StatusCode)
	}
}
