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
		errors.Is(err, domain.ErrInvalidDeliveryDate),
		errors.Is(err, domain.ErrInvalidOrderStatus),
		errors.Is(err, domain.ErrInvalidItemStatus),
		errors.Is(err, domain.ErrItemStatusImmutable),
		errors.Is(err, domain.ErrItemNotSyncable),
		errors.Is(err, domain.ErrDeliveryDateInPast),
		errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrOrderClosed),
		errors.Is(err, domain.ErrInvalidStatusTransition),
		errors.Is(err, domain.ErrSyncNotCancellable),
		errors.Is(err, domain.ErrSyncAlreadyActive):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrSAPSyncFailed):
		writeError(w, http.StatusBadGateway, "sap sync failed")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
