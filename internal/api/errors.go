package api

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"
)

// HTTPError carries an explicitly public domain error to the generated server.
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }

// HandleError preserves plain-text API errors for generated server failures.
func HandleError(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	var publicError *HTTPError
	var paramError *ogenerrors.DecodeParamsError
	var decodeError *ogenerrors.DecodeRequestError
	var securityError *ogenerrors.SecurityError
	switch {
	case errors.As(err, &publicError):
		http.Error(w, publicError.Message, publicError.Status)
	case errors.As(err, &paramError):
		http.Error(w, "invalid request parameters", http.StatusBadRequest)
	case errors.As(err, &decodeError):
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
	case errors.As(err, &securityError):
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	default:
		log.Printf("API error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
