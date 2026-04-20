package audit

import (
	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// Repository handles DB operations for audit logs.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new audit Repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// CreateLog inserts an immutable audit log entry.
func (r *Repository) CreateLog(log *db.AuditLog) error {
	return r.db.Create(log).Error
}

// GetLogs retrieves paginated audit log entries with optional filters.
func (r *Repository) GetLogs(page, limit int, filters map[string]string) ([]db.AuditLog, int64, error) {
	var logs []db.AuditLog
	var total int64

	query := r.db.Model(&db.AuditLog{})

	if resourceType, ok := filters["resourceType"]; ok && resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if actorID, ok := filters["actorId"]; ok && actorID != "" {
		query = query.Where("actor_id = ?", actorID)
	}
	if action, ok := filters["action"]; ok && action != "" {
		query = query.Where("action = ?", action)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// LogDownload records a download event.
func (r *Repository) LogDownload(log *db.DownloadLog) error {
	return r.db.Create(log).Error
}

// CheckDownloadPermission checks whether a user has explicit download permission for a resource.
func (r *Repository) CheckDownloadPermission(userID, resourceType, resourceID string) (bool, error) {
	var count int64
	err := r.db.Model(&db.DownloadPermission{}).
		Where("user_id = ? AND resource_type = ? AND resource_id = ? AND (expires_at IS NULL OR expires_at > NOW())",
			userID, resourceType, resourceID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GrantDownloadPermission creates an explicit download permission record.
func (r *Repository) GrantDownloadPermission(perm *db.DownloadPermission) error {
	return r.db.Create(perm).Error
}
