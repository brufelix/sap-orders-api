package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserClaims_HasScope_FromDelegatedScopes(t *testing.T) {
	claims := &UserClaims{
		Scopes: []string{ScopeOrdersRead, ScopeOrdersWrite},
	}

	if !claims.HasScope(ScopeOrdersRead) {
		t.Fatal("expected orders.read scope")
	}
	if !claims.HasScope(ScopeOrdersWrite) {
		t.Fatal("expected orders.write scope")
	}
}

func TestUserClaims_HasScope_FromAppRole(t *testing.T) {
	claims := &UserClaims{
		Roles: []string{RoleOrdersReader},
	}

	if !claims.HasScope(ScopeOrdersRead) {
		t.Fatal("expected mapped orders.read from Orders.Reader role")
	}
	if claims.HasScope(ScopeOrdersWrite) {
		t.Fatal("did not expect orders.write from Orders.Reader role")
	}
}

func TestUserClaims_HasScope_AdminRole(t *testing.T) {
	claims := &UserClaims{
		Roles: []string{RoleOrdersAdmin},
	}

	if !claims.HasScope(ScopeOrdersRead) || !claims.HasScope(ScopeOrdersWrite) {
		t.Fatal("expected admin role to grant read and write scopes")
	}
}

func TestRequireScope_Forbidden(t *testing.T) {
	handler := RequireScope(ScopeOrdersWrite)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req = req.WithContext(withUserClaims(context.Background(), &UserClaims{
		Email:  "reader@example.com",
		Scopes: []string{ScopeOrdersRead},
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireScope_Allowed(t *testing.T) {
	handler := RequireScope(ScopeOrdersWrite)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req = req.WithContext(withUserClaims(context.Background(), &UserClaims{
		Email:  "writer@example.com",
		Scopes: []string{ScopeOrdersWrite},
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestParseScopes(t *testing.T) {
	scopes := parseScopes("orders.read orders.write")
	if len(scopes) != 2 || scopes[0] != ScopeOrdersRead {
		t.Fatalf("unexpected scopes: %v", scopes)
	}
}
