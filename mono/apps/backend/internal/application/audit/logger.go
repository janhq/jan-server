package audit

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AdminAuditLogger records admin actions to the audit_logs table.
type AdminAuditLogger struct {
	db     *gorm.DB
	logger zerolog.Logger
}

func NewAdminAuditLogger(db *gorm.DB, logger zerolog.Logger) *AdminAuditLogger {
	return &AdminAuditLogger{db: db, logger: logger}
}

type AdminAuditEntry struct {
	AdminUserID string
	AdminEmail  string
	Action      string
	Resource    string
	ResourceID  string
	Payload     any
	StatusCode  int
	IPAddress   string
	UserAgent   string
	Error       error
}

// Log persists the admin action; best-effort (logs warning on failure).
func (l *AdminAuditLogger) Log(ctx context.Context, entry AdminAuditEntry) {
	if l == nil || l.db == nil {
		return
	}

	var payloadJSON []byte
	if entry.Payload != nil {
		if b, err := json.Marshal(entry.Payload); err == nil {
			payloadJSON = b
		}
	}

	sql := `
INSERT INTO llm_api.audit_logs 
    (admin_user_id, admin_email, action, resource_type, resource_id, payload, ip_address, user_agent, status_code, error_message)
VALUES 
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	if err := l.db.WithContext(ctx).Exec(sql,
		entry.AdminUserID,
		entry.AdminEmail,
		entry.Action,
		entry.Resource,
		entry.ResourceID,
		payloadJSON,
		entry.IPAddress,
		entry.UserAgent,
		entry.StatusCode,
		errorString(entry.Error),
	).Error; err != nil {
		l.logger.Warn().Err(err).Str("action", entry.Action).Msg("failed to write admin audit log")
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
