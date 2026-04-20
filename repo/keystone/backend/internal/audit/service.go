package audit

import (
	"encoding/json"
	"time"

	"github.com/keystone/backend/internal/db"
)

// AuditLogDTO is the public representation of an audit log entry.
type AuditLogDTO struct {
	ID           string          `json:"id"`
	ActorID      string          `json:"actorId"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resourceType"`
	ResourceID   string          `json:"resourceId"`
	DeviceID     string          `json:"deviceId"`
	IPAddress    string          `json:"ipAddress"`
	BeforeState  json.RawMessage `json:"beforeState,omitempty"`
	AfterState   json.RawMessage `json:"afterState,omitempty"`
	CreatedAt    string          `json:"createdAt"`
}

// Service implements audit business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new audit Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Log creates an audit log entry. This method satisfies auth.AuditLogger.
func (s *Service) Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error {
	entry := &db.AuditLog{
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		DeviceID:     deviceID,
		IPAddress:    ip,
		CreatedAt:    time.Now(),
	}

	if before != nil {
		b, _ := json.Marshal(before)
		entry.BeforeState = b
	}
	if after != nil {
		a, _ := json.Marshal(after)
		entry.AfterState = a
	}

	return s.repo.CreateLog(entry)
}

// GetLogs returns paginated audit log entries.
func (s *Service) GetLogs(page, limit int, filters map[string]string) ([]AuditLogDTO, int64, error) {
	logs, total, err := s.repo.GetLogs(page, limit, filters)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]AuditLogDTO, len(logs))
	for i, l := range logs {
		dtos[i] = AuditLogDTO{
			ID:           l.ID,
			ActorID:      l.ActorID,
			Action:       l.Action,
			ResourceType: l.ResourceType,
			ResourceID:   l.ResourceID,
			DeviceID:     l.DeviceID,
			IPAddress:    l.IPAddress,
			BeforeState:  l.BeforeState,
			AfterState:   l.AfterState,
			CreatedAt:    l.CreatedAt.Format(time.RFC3339),
		}
	}

	return dtos, total, nil
}

// LogDownload records a document download event.
func (s *Service) LogDownload(userID, resourceType, resourceID, deviceID string) error {
	entry := &db.DownloadLog{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		DeviceID:     deviceID,
	}
	return s.repo.LogDownload(entry)
}

// CheckDownloadPermission checks whether a user can download a resource.
func (s *Service) CheckDownloadPermission(userID, resourceType, resourceID string) (bool, error) {
	return s.repo.CheckDownloadPermission(userID, resourceType, resourceID)
}
