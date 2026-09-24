package main

import "testing"

func TestParseConfigCleanupInterval(t *testing.T) {
	for _, interval := range []string{"0s", "-1s", "invalid", "250ms"} {
		t.Run(interval, func(t *testing.T) {
			_, err := parseConfig(&ServeCommand{
				Port:                     8080,
				JWT:                      JWTConfig{Issuer: "https://issuer.example", Audience: "waypoint"},
				WorkspaceCleanupInterval: interval,
			})
			if (err == nil) != (interval == "250ms") {
				t.Fatalf("unexpected config validation result: %v", err)
			}
		})
	}
}
