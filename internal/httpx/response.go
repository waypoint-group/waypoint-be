package httpx

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes value before writing the status and JSON response body.
func WriteJSON(w http.ResponseWriter, status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(payload)
	return err
}
