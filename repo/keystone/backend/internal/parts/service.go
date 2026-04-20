package parts

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// AuditLogger is the minimal interface for writing audit events.
type AuditLogger interface {
	Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error
}

// Service implements business logic for parts.
type Service struct {
	repo  *Repository
	audit AuditLogger
}

// NewService creates a new parts Service.
func NewService(repo *Repository, audit AuditLogger) *Service {
	return &Service{repo: repo, audit: audit}
}

// CreatePart creates a new part with its initial version.
func (s *Service) CreatePart(createdBy, siteID, orgID, deviceID, ip string, req CreatePartRequest) (*PartDTO, error) {
	part := &db.Part{
		PartNumber:     req.PartNumber,
		Name:           req.Name,
		Description:    req.Description,
		Status:         "ACTIVE",
		SiteID:         siteID,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}

	if err := s.repo.CreatePart(part); err != nil {
		return nil, err
	}

	version := &db.PartVersion{
		PartID:        part.ID,
		VersionNumber: 1,
		Fitment:       req.Fitment,
		OEMMappings:   req.OEMMappings,
		Attributes:    req.Attributes,
		ChangeSummary: req.ChangeSummary,
		ChangedBy:     createdBy,
	}

	if err := s.repo.CreateVersion(version); err != nil {
		return nil, err
	}

	// Set current_version_id.
	part.CurrentVersionID = &version.ID
	if err := s.repo.UpdatePart(part); err != nil {
		return nil, err
	}

	_ = s.audit.Log(createdBy, "PART_CREATED", "part", part.ID, deviceID, ip, nil, part)

	return toPartDTO(part, version), nil
}

// GetParts returns paginated parts.
func (s *Service) GetParts(filters map[string]string, page, limit int) ([]PartDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	parts, total, err := s.repo.GetParts(filters, page, limit)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]PartDTO, len(parts))
	for i, p := range parts {
		dtos[i] = *toPartDTO(&p, nil)
	}
	return dtos, total, nil
}

// GetPart retrieves a single part, enforcing org/site scope for non-ADMIN callers.
func (s *Service) GetPart(id, orgID, siteID, role string) (*PartDTO, error) {
	part, err := s.repo.GetPartByID(id)
	if err != nil {
		return nil, err
	}
	if role != "ADMIN" && orgID != "" && part.OrganizationID != orgID {
		return nil, gorm.ErrRecordNotFound
	}
	if role != "ADMIN" && siteID != "" && part.SiteID != siteID {
		return nil, gorm.ErrRecordNotFound
	}

	var version *db.PartVersion
	if part.CurrentVersionID != nil {
		version, _ = s.repo.GetVersionByID(*part.CurrentVersionID)
	}

	return toPartDTO(part, version), nil
}

// UpdatePart creates a new version for an existing part (never overwrites).
func (s *Service) UpdatePart(id, updatedBy string, req CreatePartRequest) (*PartVersionDTO, error) {
	part, err := s.repo.GetPartByID(id)
	if err != nil {
		return nil, err
	}

	before := *part
	part.Name = req.Name
	part.Description = req.Description
	if err := s.repo.UpdatePart(part); err != nil {
		return nil, err
	}

	maxVer, err := s.repo.GetMaxVersionNumber(id)
	if err != nil {
		return nil, err
	}

	version := &db.PartVersion{
		PartID:        id,
		VersionNumber: maxVer + 1,
		Fitment:       req.Fitment,
		OEMMappings:   req.OEMMappings,
		Attributes:    req.Attributes,
		ChangeSummary: req.ChangeSummary,
		ChangedBy:     updatedBy,
	}

	if err := s.repo.CreateVersion(version); err != nil {
		return nil, err
	}

	_ = s.audit.Log(updatedBy, "PART_UPDATED", "part", part.ID, "", "", before, version)

	return toVersionDTO(version), nil
}

// checkPartScope returns gorm.ErrRecordNotFound if the part is outside the caller's org/site.
func (s *Service) checkPartScope(partID, orgID, siteID, role string) error {
	if role == "ADMIN" {
		return nil
	}
	part, err := s.repo.GetPartByID(partID)
	if err != nil {
		return err
	}
	if orgID != "" && part.OrganizationID != orgID {
		return gorm.ErrRecordNotFound
	}
	if siteID != "" && part.SiteID != siteID {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetVersions returns all versions of a part.
func (s *Service) GetVersions(partID, orgID, siteID, role string) ([]PartVersionDTO, error) {
	if err := s.checkPartScope(partID, orgID, siteID, role); err != nil {
		return nil, err
	}
	versions, err := s.repo.GetAllVersions(partID)
	if err != nil {
		return nil, err
	}

	dtos := make([]PartVersionDTO, len(versions))
	for i, v := range versions {
		dtos[i] = *toVersionDTO(&v)
	}
	return dtos, nil
}

// CompareVersions returns a field-by-field diff of two part versions.
func (s *Service) CompareVersions(partID, orgID, siteID, role string, v1Num, v2Num int) (*VersionCompareDTO, error) {
	if err := s.checkPartScope(partID, orgID, siteID, role); err != nil {
		return nil, err
	}
	v1, err := s.repo.GetVersion(partID, v1Num)
	if err != nil {
		return nil, fmt.Errorf("version %d not found", v1Num)
	}
	v2, err := s.repo.GetVersion(partID, v2Num)
	if err != nil {
		return nil, fmt.Errorf("version %d not found", v2Num)
	}

	diff := make(map[string]DiffField)
	fields := []string{"fitment", "oem_mappings", "attributes"}

	v1Vals := map[string]json.RawMessage{
		"fitment":     v1.Fitment,
		"oem_mappings": v1.OEMMappings,
		"attributes":  v1.Attributes,
	}
	v2Vals := map[string]json.RawMessage{
		"fitment":     v2.Fitment,
		"oem_mappings": v2.OEMMappings,
		"attributes":  v2.Attributes,
	}

	for _, field := range fields {
		aStr := string(v1Vals[field])
		bStr := string(v2Vals[field])
		diff[field] = DiffField{
			OldValue: json.RawMessage(v1Vals[field]),
			NewValue: json.RawMessage(v2Vals[field]),
			Changed:  aStr != bStr,
		}
	}

	return &VersionCompareDTO{
		VersionA: *toVersionDTO(v1),
		VersionB: *toVersionDTO(v2),
		Diff:     diff,
	}, nil
}

// PromoteVersion sets a specific version as the current version of a part.
func (s *Service) PromoteVersion(partID, versionID, promotedBy string) error {
	version, err := s.repo.GetVersionByID(versionID)
	if err != nil {
		return err
	}
	if version.PartID != partID {
		return errors.New("version does not belong to this part")
	}

	if err := s.repo.PromoteVersion(partID, versionID); err != nil {
		return err
	}

	_ = s.audit.Log(promotedBy, "PART_VERSION_PROMOTED", "part", partID, "", "", nil, map[string]string{"versionId": versionID})
	return nil
}

// ValidateRows checks import rows for required fields and returns any validation errors.
func (s *Service) ValidateRows(rows []CSVImportRow) []string {
	var errs []string
	for i, row := range rows {
		if strings.TrimSpace(row.PartNumber) == "" {
			errs = append(errs, fmt.Sprintf("row %d: partNumber is required", i+1))
		}
		if strings.TrimSpace(row.Name) == "" {
			errs = append(errs, fmt.Sprintf("row %d: name is required", i+1))
		}
	}
	return errs
}

// BulkImport validates and inserts parts from parsed CSV rows in a single transaction.
func (s *Service) BulkImport(rows []CSVImportRow, importedBy string) (int, []string, error) {
	if validationErrors := s.ValidateRows(rows); len(validationErrors) > 0 {
		return 0, validationErrors, errors.New("validation failed")
	}

	dbParts := make([]db.Part, 0, len(rows))
	dbVersions := make([]db.PartVersion, 0, len(rows))

	for _, row := range rows {
		p := db.Part{
			PartNumber:  row.PartNumber,
			Name:        row.Name,
			Description: row.Description,
			Status:      "ACTIVE",
			CreatedBy:   importedBy,
		}
		v := db.PartVersion{
			VersionNumber: 1,
			ChangedBy:     importedBy,
			ChangeSummary: "initial import",
		}
		if row.FitmentJSON != "" {
			v.Fitment = json.RawMessage(row.FitmentJSON)
		}
		if row.OEMMappingsJSON != "" {
			v.OEMMappings = json.RawMessage(row.OEMMappingsJSON)
		}
		if row.AttributesJSON != "" {
			v.Attributes = json.RawMessage(row.AttributesJSON)
		}
		dbParts = append(dbParts, p)
		dbVersions = append(dbVersions, v)
	}

	if err := s.repo.BulkCreatePartsAndVersions(dbParts, dbVersions); err != nil {
		return 0, nil, err
	}

	_ = s.audit.Log(importedBy, "PARTS_BULK_IMPORTED", "part", "", "", "", nil, map[string]int{"count": len(rows)})
	return len(rows), nil, nil
}

// ExportCSV generates a CSV file of all parts with the selected fields.
func (s *Service) ExportCSV(actorID, deviceID, ip string, fields []string) ([]byte, error) {
	parts, err := s.repo.GetAllParts()
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		fields = []string{"part_number", "name", "description", "status", "created_at"}
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write(fields)

	for _, p := range parts {
		row := make([]string, len(fields))
		for i, f := range fields {
			switch f {
			case "part_number":
				row[i] = p.PartNumber
			case "name":
				row[i] = p.Name
			case "description":
				row[i] = p.Description
			case "status":
				row[i] = p.Status
			case "created_at":
				row[i] = p.CreatedAt.Format(time.RFC3339)
			case "id":
				row[i] = p.ID
			default:
				row[i] = ""
			}
		}
		_ = w.Write(row)
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	_ = s.audit.Log(actorID, "PARTS_EXPORTED", "part", "", deviceID, ip, nil, map[string]interface{}{
		"fields": fields,
		"count":  len(parts),
	})

	return buf.Bytes(), nil
}

// ParseCSVRows parses raw CSV bytes into a slice of CSVImportRow.
func ParseCSVRows(data []byte) ([]CSVImportRow, error) {
	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, errors.New("CSV must have a header row and at least one data row")
	}

	// Build header index.
	headers := records[0]
	headerIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	get := func(record []string, key string) string {
		idx, ok := headerIdx[key]
		if !ok || idx >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[idx])
	}

	rows := make([]CSVImportRow, 0, len(records)-1)
	for _, record := range records[1:] {
		rows = append(rows, CSVImportRow{
			PartNumber:      get(record, "part_number"),
			Name:            get(record, "name"),
			Description:     get(record, "description"),
			FitmentJSON:     get(record, "fitment"),
			OEMMappingsJSON: get(record, "oem_mappings"),
			AttributesJSON:  get(record, "attributes"),
		})
	}
	return rows, nil
}

// toPartDTO converts db models to a PartDTO.
func toPartDTO(p *db.Part, v *db.PartVersion) *PartDTO {
	dto := &PartDTO{
		ID:          p.ID,
		PartNumber:  p.PartNumber,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
	}
	if p.CurrentVersionID != nil {
		dto.CurrentVersionID = *p.CurrentVersionID
	}
	if v != nil {
		dto.VersionNumber = v.VersionNumber
	}
	return dto
}

// toVersionDTO converts a db.PartVersion to a PartVersionDTO.
func toVersionDTO(v *db.PartVersion) *PartVersionDTO {
	return &PartVersionDTO{
		ID:            v.ID,
		PartID:        v.PartID,
		VersionNumber: v.VersionNumber,
		Fitment:       v.Fitment,
		OEMMappings:   v.OEMMappings,
		Attributes:    v.Attributes,
		ChangeSummary: v.ChangeSummary,
		ChangedBy:     v.ChangedBy,
		CreatedAt:     v.CreatedAt.Format(time.RFC3339),
	}
}
