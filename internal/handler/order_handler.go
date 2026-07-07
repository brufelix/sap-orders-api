package handler

import (
	"encoding/json"
	"net/http"

	"github.com/brufelix/sap-orders-api/internal/auth"
	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/service"
	"github.com/go-chi/chi/v5"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(service *service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.ListOrders(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdBy := auth.UserEmailFromContext(r.Context())
	order, err := h.service.CreateOrder(r.Context(), input, createdBy)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var input domain.UpdateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.UpdateOrder(r.Context(), id, input)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var input domain.CreateOrderItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.AddItem(r.Context(), orderID, input)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *OrderHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
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

	var input domain.UpdateOrderItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.UpdateItem(r.Context(), orderID, itemID, input)
	if err != nil {
		HandleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
