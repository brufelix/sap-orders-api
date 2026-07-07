package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

func HandleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, domain.ErrOrderNumberRequired),
		errors.Is(err, domain.ErrStatusRequired),
		errors.Is(err, domain.ErrItemFieldsRequired),
		errors.Is(err, domain.ErrInvalidDeliveryDate):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrSAPSyncFailed):
		writeError(w, http.StatusBadGateway, "sap sync failed")
	case errors.Is(err, domain.ErrSyncNotCancellable):
		writeError(w, http.StatusConflict, "sync can only be cancelled while pending")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
