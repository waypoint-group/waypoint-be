// Package httpx provides shared HTTP request helpers.
package httpx

import (
	"fmt"
	"net/http"
	"uuid"
)

// PathID parses a required, nonzero UUID path parameter.
func PathID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse %s: %w", name, err)
	}
	if id == (uuid.UUID{}) {
		return uuid.UUID{}, fmt.Errorf("%s must not be a zero UUID", name)
	}
	return id, nil
}
