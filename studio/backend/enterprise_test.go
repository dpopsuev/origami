package backend

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		role Role
		perm Permission
		want bool
	}{
		{RoleAdmin, PermRunLaunch, true},
		{RoleAdmin, PermSettingsManage, true},
		{RoleOperator, PermRunLaunch, true},
		{RoleOperator, PermSettingsManage, false},
		{RoleViewer, PermRunLaunch, false},
		{RoleViewer, PermCircuitRead, true},
		{Role("unknown"), PermCircuitRead, false},
	}

	for _, tt := range tests {
		got := HasPermission(tt.role, tt.perm)
		if got != tt.want {
			t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.perm, got, tt.want)
		}
	}
}

func TestAuditLog(t *testing.T) {
	log := NewAuditLog()
	log.Record("alice", "run:launch", "/api/runs", "allowed")
	log.Record("bob", "run:launch", "/api/runs", "denied")

	entries := log.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Actor != "alice" {
		t.Errorf("expected actor 'alice', got %q", entries[0].Actor)
	}

	since := log.EntriesSince(1)
	if len(since) != 1 {
		t.Errorf("expected 1 entry since ID 1, got %d", len(since))
	}
}

func TestLicenseInfo(t *testing.T) {
	community := LicenseInfo{Valid: true, Edition: "community"}
	if community.IsEnterprise() {
		t.Error("community license should not be enterprise")
	}

	enterprise := LicenseInfo{Valid: true, Edition: "enterprise"}
	if !enterprise.IsEnterprise() {
		t.Error("enterprise license should be enterprise")
	}

	invalid := LicenseInfo{Valid: false, Edition: "enterprise"}
	if invalid.IsEnterprise() {
		t.Error("invalid enterprise license should not be enterprise")
	}
}

func TestRequireEnterprise(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("blocked without enterprise", func(t *testing.T) {
		h := RequireEnterprise(LicenseInfo{Valid: true, Edition: "community"}, ok)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("allowed with enterprise", func(t *testing.T) {
		h := RequireEnterprise(LicenseInfo{Valid: true, Edition: "enterprise"}, ok)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

func TestRequirePermission(t *testing.T) {
	audit := NewAuditLog()
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := RequirePermission(PermRunLaunch, audit, ok)

	t.Run("viewer denied launch", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/runs", nil)
		req.Header.Set("X-Studio-Role", "viewer")
		req.Header.Set("X-Studio-User", "bob")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("operator allowed launch", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/runs", nil)
		req.Header.Set("X-Studio-Role", "operator")
		req.Header.Set("X-Studio-User", "alice")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	entries := audit.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(entries))
	}
	if entries[0].Outcome != "denied" {
		t.Errorf("expected 'denied', got %q", entries[0].Outcome)
	}
	if entries[1].Outcome != "allowed" {
		t.Errorf("expected 'allowed', got %q", entries[1].Outcome)
	}
}
