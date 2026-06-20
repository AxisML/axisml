package server

import "time"

// AuditLog is a single audit-log entry.
type AuditLog struct {
	ID        UUID           `json:"id"`
	UserID    *UUID          `json:"userId,omitempty"`
	Username  string         `json:"username,omitempty"`
	Action    string         `json:"action"`
	Target    string         `json:"target,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	IPAddress string         `json:"ipAddress,omitempty"`
	UserAgent string         `json:"userAgent,omitempty"`
	Result    AuditResult    `json:"result,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// AuditLogList is a page of AuditLog.
type AuditLogList struct {
	Items         []AuditLog `json:"items"`
	Count         int        `json:"count" binding:"min=0"`
	ContinueToken string     `json:"continueToken,omitempty"`
}
