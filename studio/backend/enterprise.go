package backend

import (
	"net/http"
	"sync"
	"time"
)

// Role represents an RBAC role in the Enterprise edition.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// Permission represents a specific action.
type Permission string

const (
	PermPipelineRead    Permission = "pipeline:read"
	PermPipelineWrite   Permission = "pipeline:write"
	PermRunLaunch       Permission = "run:launch"
	PermRunView         Permission = "run:view"
	PermAdapterInstall  Permission = "adapter:install"
	PermSettingsManage  Permission = "settings:manage"
	PermScheduleManage  Permission = "schedule:manage"
)

// RolePermissions maps roles to their allowed permissions.
var RolePermissions = map[Role][]Permission{
	RoleAdmin:    {PermPipelineRead, PermPipelineWrite, PermRunLaunch, PermRunView, PermAdapterInstall, PermSettingsManage, PermScheduleManage},
	RoleOperator: {PermPipelineRead, PermPipelineWrite, PermRunLaunch, PermRunView},
	RoleViewer:   {PermPipelineRead, PermRunView},
}

// HasPermission checks if a role has a specific permission.
func HasPermission(role Role, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// AuditEntry records a single action in the audit trail.
type AuditEntry struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Outcome   string    `json:"outcome"`
	Details   string    `json:"details,omitempty"`
}

// AuditLog is a thread-safe append-only audit trail.
type AuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry
	nextID  int
}

// NewAuditLog creates a new empty audit log.
func NewAuditLog() *AuditLog {
	return &AuditLog{nextID: 1}
}

// Record appends an entry to the audit log.
func (a *AuditLog) Record(actor, action, resource, outcome string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, AuditEntry{
		ID:        a.nextID,
		Timestamp: time.Now(),
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		Outcome:   outcome,
	})
	a.nextID++
}

// Entries returns all audit entries.
func (a *AuditLog) Entries() []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]AuditEntry, len(a.entries))
	copy(result, a.entries)
	return result
}

// EntriesSince returns entries after the given ID.
func (a *AuditLog) EntriesSince(afterID int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []AuditEntry
	for _, e := range a.entries {
		if e.ID > afterID {
			result = append(result, e)
		}
	}
	return result
}

// LicenseInfo represents the enterprise license state.
type LicenseInfo struct {
	Valid       bool   `json:"valid"`
	Edition     string `json:"edition"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	MaxUsers    int    `json:"max_users,omitempty"`
	OrgName     string `json:"org_name,omitempty"`
}

// IsEnterprise returns true if the license grants enterprise features.
func (l LicenseInfo) IsEnterprise() bool {
	return l.Valid && l.Edition == "enterprise"
}

// RequireEnterprise is HTTP middleware that gates enterprise endpoints.
func RequireEnterprise(license LicenseInfo, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !license.IsEnterprise() {
			http.Error(w, `{"error":"enterprise license required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePermission is HTTP middleware that checks RBAC permissions.
// It reads the role from the X-Studio-Role header (simplified for PoC).
func RequirePermission(perm Permission, audit *AuditLog, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roleStr := r.Header.Get("X-Studio-Role")
		if roleStr == "" {
			roleStr = string(RoleViewer)
		}
		role := Role(roleStr)

		actor := r.Header.Get("X-Studio-User")
		if actor == "" {
			actor = "anonymous"
		}

		if !HasPermission(role, perm) {
			audit.Record(actor, string(perm), r.URL.Path, "denied")
			http.Error(w, `{"error":"permission denied"}`, http.StatusForbidden)
			return
		}

		audit.Record(actor, string(perm), r.URL.Path, "allowed")
		next.ServeHTTP(w, r)
	})
}
