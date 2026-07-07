package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/handler"
)

func TestHandleError_NotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.HandleError(rec, domain.ErrNotFound)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleError_Validation(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.HandleError(rec, domain.ErrOrderNumberRequired)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleError_Internal(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.HandleError(rec, errors.New("unexpected"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleError_SyncNotCancellable(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.HandleError(rec, domain.ErrSyncNotCancellable)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}
