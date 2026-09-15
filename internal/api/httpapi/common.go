// Package httpapi provides the HTTP transport for the Waypoint service.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const maxRequestBodyBytes = 1 << 20

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return fmt.Errorf("decode trailing request data: %w", err)
	}

	return nil
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(payload)
	return err
}

func writeInternalServerError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func writeInvalidRequestBody(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
}
