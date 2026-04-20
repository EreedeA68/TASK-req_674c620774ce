package candidate

import (
	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// Repository handles DB operations for candidates.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new candidate Repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// CreateCandidate inserts a new candidate record in a transaction.
func (r *Repository) CreateCandidate(candidate *db.Candidate) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(candidate).Error
	})
}

// GetCandidateByID retrieves a candidate by UUID.
func (r *Repository) GetCandidateByID(id string) (*db.Candidate, error) {
	var c db.Candidate
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCandidates retrieves paginated candidates with optional filters.
func (r *Repository) GetCandidates(filters map[string]string, page, limit int) ([]db.Candidate, int64, error) {
	var candidates []db.Candidate
	var total int64

	query := r.db.Model(&db.Candidate{}).Where("deleted_at IS NULL")

	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if createdBy, ok := filters["createdBy"]; ok && createdBy != "" {
		query = query.Where("created_by = ?", createdBy)
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
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&candidates).Error; err != nil {
		return nil, 0, err
	}

	return candidates, total, nil
}

// UpdateCandidate persists changes to a candidate record in a transaction.
func (r *Repository) UpdateCandidate(candidate *db.Candidate) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Save(candidate).Error
	})
}

// CreateDocument inserts a candidate document record in a transaction.
func (r *Repository) CreateDocument(doc *db.CandidateDocument) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(doc).Error
	})
}

// GetDocumentByHash retrieves a document by its SHA-256 hash.
func (r *Repository) GetDocumentByHash(hash string) (*db.CandidateDocument, error) {
	var doc db.CandidateDocument
	if err := r.db.Where("sha256_hash = ?", hash).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetDocumentByID retrieves a document by its UUID.
func (r *Repository) GetDocumentByID(id string) (*db.CandidateDocument, error) {
	var doc db.CandidateDocument
	if err := r.db.Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetDocumentsByCandidateID retrieves all documents for a candidate.
func (r *Repository) GetDocumentsByCandidateID(candidateID string) ([]db.CandidateDocument, error) {
	var docs []db.CandidateDocument
	if err := r.db.Where("candidate_id = ?", candidateID).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// CountDocumentsByCandidateID returns the number of uploaded documents for a candidate.
func (r *Repository) CountDocumentsByCandidateID(candidateID string) (int64, error) {
	var count int64
	if err := r.db.Model(&db.CandidateDocument{}).Where("candidate_id = ?", candidateID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
