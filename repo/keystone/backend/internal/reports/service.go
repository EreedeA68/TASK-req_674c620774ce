package reports

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const defaultQuota = 100

// KPIReport holds computed KPI metrics.
type KPIReport struct {
	ConversionRate    float64 `json:"conversionRate"`
	ReviewCycleTime   float64 `json:"reviewCycleTimeHours"`
	QuotaUtilization  float64 `json:"quotaUtilization"`
	TotalSubmitted    int64   `json:"totalSubmitted"`
	TotalApproved     int64   `json:"totalApproved"`
}

// candidateStat is used for raw queries.
type candidateStat struct {
	Status     string
	Count      int64
	AvgSeconds float64
}

// AuditLogger is the minimal interface for writing audit events.
type AuditLogger interface {
	Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error
}

// Service implements reports business logic.
type Service struct {
	db    *gorm.DB
	audit AuditLogger
}

// NewService creates a new reports Service.
func NewService(db *gorm.DB, audit AuditLogger) *Service {
	return &Service{db: db, audit: audit}
}

// GetKPIs computes conversion rate, review cycle time, and quota utilization.
func (s *Service) GetKPIs() (*KPIReport, error) {
	// Count submitted and approved candidates.
	var totalSubmitted, totalApproved int64
	if err := s.db.Raw("SELECT COUNT(*) FROM candidates WHERE status = 'SUBMITTED' AND deleted_at IS NULL").Scan(&totalSubmitted).Error; err != nil {
		return nil, err
	}
	if err := s.db.Raw("SELECT COUNT(*) FROM candidates WHERE status = 'APPROVED' AND deleted_at IS NULL").Scan(&totalApproved).Error; err != nil {
		return nil, err
	}

	// Conversion rate.
	var conversionRate float64
	if totalSubmitted > 0 {
		conversionRate = float64(totalApproved) / float64(totalSubmitted)
	}

	// Average review cycle time (submitted -> reviewed), in hours.
	var avgSeconds float64
	if err := s.db.Raw(`
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (reviewed_at - submitted_at))), 0)
		FROM candidates
		WHERE status IN ('APPROVED', 'REJECTED')
		  AND submitted_at IS NOT NULL
		  AND reviewed_at IS NOT NULL
		  AND deleted_at IS NULL
	`).Scan(&avgSeconds).Error; err != nil {
		return nil, err
	}
	reviewCycleTimeHours := avgSeconds / 3600.0

	// Quota utilization: current period (this calendar month).
	var periodSubmissions int64
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := s.db.Raw("SELECT COUNT(*) FROM candidates WHERE submitted_at >= ? AND deleted_at IS NULL", periodStart).Scan(&periodSubmissions).Error; err != nil {
		return nil, err
	}

	quotaUtilization := float64(periodSubmissions) / float64(defaultQuota)

	return &KPIReport{
		ConversionRate:   conversionRate,
		ReviewCycleTime:  reviewCycleTimeHours,
		QuotaUtilization: quotaUtilization,
		TotalSubmitted:   totalSubmitted,
		TotalApproved:    totalApproved,
	}, nil
}

// ExportReport generates a CSV report with selected non-sensitive fields and logs the export.
func (s *Service) ExportReport(fields []string, role, actorID, deviceID, ip string) ([]byte, error) {
	// Sensitive fields that are masked or excluded for non-admin roles.
	sensitiveFields := map[string]bool{
		"demographics":         true,
		"exam_scores":          true,
		"application_details":  true,
		"transfer_preferences": true,
	}

	// Default export fields (safe for all roles).
	safeFields := []string{"id", "status", "completeness_status", "submitted_at", "reviewed_at", "created_at"}
	allowedFields := map[string]bool{
		"id": true, "status": true, "completeness_status": true,
		"submitted_at": true, "reviewed_at": true, "created_at": true,
		"demographics": true, "exam_scores": true, "application_details": true, "transfer_preferences": true,
	}

	if len(fields) == 0 {
		fields = safeFields
	}

	// Reject unknown fields early.
	for _, f := range fields {
		if !allowedFields[f] {
			return nil, fmt.Errorf("unknown export field: %s", f)
		}
	}

	// Non-admins cannot export sensitive fields.
	if role != "ADMIN" {
		filtered := fields[:0]
		for _, f := range fields {
			if !sensitiveFields[f] {
				filtered = append(filtered, f)
			}
		}
		fields = filtered
	}

	type candidateExportRow struct {
		ID                 string
		Status             string
		CompletenessStatus string
		SubmittedAt        *time.Time
		ReviewedAt         *time.Time
		CreatedAt          time.Time
	}

	var rows []candidateExportRow
	if err := s.db.Raw(`
		SELECT id, status, completeness_status, submitted_at, reviewed_at, created_at
		FROM candidates WHERE deleted_at IS NULL ORDER BY created_at DESC
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(fields)

	for _, row := range rows {
		record := make([]string, len(fields))
		for i, f := range fields {
			switch f {
			case "id":
				record[i] = row.ID
			case "status":
				record[i] = row.Status
			case "completeness_status":
				record[i] = row.CompletenessStatus
			case "submitted_at":
				if row.SubmittedAt != nil {
					record[i] = row.SubmittedAt.Format(time.RFC3339)
				}
			case "reviewed_at":
				if row.ReviewedAt != nil {
					record[i] = row.ReviewedAt.Format(time.RFC3339)
				}
			case "created_at":
				record[i] = row.CreatedAt.Format(time.RFC3339)
			default:
				record[i] = ""
			}
		}
		_ = w.Write(record)
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	_ = s.LogExportAudit(actorID, deviceID, ip, len(rows))
	return buf.Bytes(), nil
}

// LogExportAudit records a report export audit event.
func (s *Service) LogExportAudit(actorID, deviceID, ip string, rowCount int) error {
	if s.audit != nil {
		return s.audit.Log(actorID, "REPORT_EXPORTED", "report", "", deviceID, ip, nil, map[string]int{"rowCount": rowCount})
	}
	return nil
}
