package handler

import (
	"net/http"

	"github.com/brufelix/sap-orders-api/internal/service"
	"github.com/go-chi/chi/v5"
)

type SyncHandler struct {
	service *service.SyncService
}

func NewSyncHandler(service *service.SyncService) *SyncHandler {
	return &SyncHandler{service: service}
}

func (h *SyncHandler) SyncItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	itemID, err := parseUUID(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	entry, err := h.service.EnqueueSync(r.Context(), orderID, itemID)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, entry)
}

func (h *SyncHandler) GetLatestStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	itemID, err := parseUUID(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	status, err := h.service.GetLatestSyncStatus(r.Context(), orderID, itemID)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *SyncHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	itemID, err := parseUUID(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	outboxID, err := parseUUID(chi.URLParam(r, "outboxId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid outbox id")
		return
	}

	status, err := h.service.GetSyncStatus(r.Context(), orderID, itemID, outboxID)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *SyncHandler) CancelSync(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	itemID, err := parseUUID(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	outboxID, err := parseUUID(chi.URLParam(r, "outboxId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid outbox id")
		return
	}

	entry, err := h.service.CancelSync(r.Context(), orderID, itemID, outboxID)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *SyncHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	itemID, err := parseUUID(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	logs, err := h.service.ListLogs(r.Context(), orderID, itemID)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}
