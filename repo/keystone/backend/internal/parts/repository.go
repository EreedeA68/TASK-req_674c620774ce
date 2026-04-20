package parts

import (
	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// Repository handles DB operations for parts.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new parts Repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// CreatePart inserts a new part record in a transaction.
func (r *Repository) CreatePart(part *db.Part) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(part).Error
	})
}

// GetPartByID retrieves a part by UUID.
func (r *Repository) GetPartByID(id string) (*db.Part, error) {
	var p db.Part
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPartByNumber retrieves a part by part number.
func (r *Repository) GetPartByNumber(partNumber string) (*db.Part, error) {
	var p db.Part
	if err := r.db.Where("part_number = ? AND deleted_at IS NULL", partNumber).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetParts retrieves paginated parts with optional filters.
func (r *Repository) GetParts(filters map[string]string, page, limit int) ([]db.Part, int64, error) {
	var parts []db.Part
	var total int64

	query := r.db.Model(&db.Part{}).Where("deleted_at IS NULL")

	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if search, ok := filters["search"]; ok && search != "" {
		query = query.Where("name ILIKE ? OR part_number ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if orgID, ok := filters["orgID"]; ok && orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}
	if siteID, ok := filters["siteID"]; ok && siteID != "" {
		query = query.Where("site_id = ?", siteID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&parts).Error; err != nil {
		return nil, 0, err
	}

	return parts, total, nil
}

// UpdatePart persists changes to a part in a transaction.
func (r *Repository) UpdatePart(part *db.Part) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Save(part).Error
	})
}

// GetAllVersions retrieves all versions of a part ordered by version number.
func (r *Repository) GetAllVersions(partID string) ([]db.PartVersion, error) {
	var versions []db.PartVersion
	if err := r.db.Where("part_id = ?", partID).Order("version_number ASC").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// GetVersion retrieves a specific version of a part.
func (r *Repository) GetVersion(partID string, versionNum int) (*db.PartVersion, error) {
	var version db.PartVersion
	if err := r.db.Where("part_id = ? AND version_number = ?", partID, versionNum).First(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// GetVersionByID retrieves a version by its UUID.
func (r *Repository) GetVersionByID(versionID string) (*db.PartVersion, error) {
	var version db.PartVersion
	if err := r.db.Where("id = ?", versionID).First(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// GetMaxVersionNumber returns the highest version number for a given part.
func (r *Repository) GetMaxVersionNumber(partID string) (int, error) {
	var maxVer int
	row := r.db.Model(&db.PartVersion{}).Select("COALESCE(MAX(version_number), 0)").Where("part_id = ?", partID).Row()
	if err := row.Scan(&maxVer); err != nil {
		return 0, err
	}
	return maxVer, nil
}

// CreateVersion inserts a new part version in a transaction.
func (r *Repository) CreateVersion(version *db.PartVersion) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(version).Error
	})
}

// PromoteVersion updates the current_version_id of a part.
func (r *Repository) PromoteVersion(partID, versionID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&db.Part{}).Where("id = ?", partID).Update("current_version_id", versionID).Error
	})
}

// GetAllParts retrieves all non-deleted parts (for export).
func (r *Repository) GetAllParts() ([]db.Part, error) {
	var parts []db.Part
	if err := r.db.Where("deleted_at IS NULL").Order("created_at ASC").Find(&parts).Error; err != nil {
		return nil, err
	}
	return parts, nil
}

// BulkCreatePartsAndVersions inserts multiple parts and their first versions atomically.
func (r *Repository) BulkCreatePartsAndVersions(parts []db.Part, versions []db.PartVersion) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range parts {
			if err := tx.Create(&parts[i]).Error; err != nil {
				return err
			}
			versions[i].PartID = parts[i].ID
			if err := tx.Create(&versions[i]).Error; err != nil {
				return err
			}
			// Update current_version_id.
			if err := tx.Model(&parts[i]).Update("current_version_id", versions[i].ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
