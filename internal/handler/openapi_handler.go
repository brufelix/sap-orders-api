package handler

import (
	"net/http"

	"github.com/brufelix/sap-orders-api/internal/openapi"
)

func OpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openapi.Spec)
}
