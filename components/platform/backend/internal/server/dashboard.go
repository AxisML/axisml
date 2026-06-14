package server

import "time"

// DashboardOverview is the aggregated platform overview. All fields optional.
type DashboardOverview struct {
	TenantCount           int     `json:"tenantCount,omitempty"`
	ActiveTenantCount     int     `json:"activeTenantCount,omitempty"`
	ActiveJobCount        int     `json:"activeJobCount,omitempty"`
	RunningServiceCount   int     `json:"runningServiceCount,omitempty"`
	RunningWorkspaceCount int     `json:"runningWorkspaceCount,omitempty"`
	GPUTotal              float64 `json:"gpuTotal,omitempty"`
	GPUUsed               float64 `json:"gpuUsed,omitempty"`
	CPUTotalCores         float64 `json:"cpuTotalCores,omitempty"`
	CPUUsedCores          float64 `json:"cpuUsedCores,omitempty"`
	MemoryTotalGiB        float64 `json:"memoryTotalGiB,omitempty"`
	MemoryUsedGiB         float64 `json:"memoryUsedGiB,omitempty"`
	ModelCount            int     `json:"modelCount,omitempty"`
}

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
