package lostfound

import (
	"time"

	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// Repository handles DB operations for listings.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new lostfound Repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// CreateListing inserts a new listing in a transaction.
func (r *Repository) CreateListing(listing *db.Listing) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(listing).Error
	})
}

// GetListingByID retrieves a listing by UUID.
func (r *Repository) GetListingByID(id string) (*db.Listing, error) {
	var l db.Listing
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// GetListings retrieves paginated listings with optional filters.
func (r *Repository) GetListings(filters map[string]string, visibleStatuses []string, page, limit int) ([]db.Listing, int64, error) {
	var listings []db.Listing
	var total int64

	query := r.db.Model(&db.Listing{}).Where("deleted_at IS NULL")

	if len(visibleStatuses) > 0 {
		query = query.Where("status IN ?", visibleStatuses)
	}
	if category, ok := filters["category"]; ok && category != "" {
		query = query.Where("category = ?", category)
	}
	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
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
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&listings).Error; err != nil {
		return nil, 0, err
	}

	return listings, total, nil
}

// UpdateListing persists changes to a listing in a transaction.
func (r *Repository) UpdateListing(listing *db.Listing) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Save(listing).Error
	})
}

// GetSimilarListings returns listings in the same category created within 24h.
func (r *Repository) GetSimilarListings(category string, within24h bool) ([]db.Listing, error) {
	var listings []db.Listing
	query := r.db.Where("category = ? AND status IN ? AND deleted_at IS NULL", category, []string{"PUBLISHED", "UNLISTED"})
	if within24h {
		cutoff := time.Now().Add(-24 * time.Hour)
		query = query.Where("created_at >= ?", cutoff)
	}
	if err := query.Find(&listings).Error; err != nil {
		return nil, err
	}
	return listings, nil
}

// GetListingsOlderThan90Days returns all published listings created more than 90 days ago.
func (r *Repository) GetListingsOlderThan90Days() ([]db.Listing, error) {
	var listings []db.Listing
	cutoff := time.Now().AddDate(0, 0, -90)
	if err := r.db.Where("status = 'PUBLISHED' AND created_at < ? AND deleted_at IS NULL", cutoff).Find(&listings).Error; err != nil {
		return nil, err
	}
	return listings, nil
}

// AutoUnlistOld updates old PUBLISHED listings to UNLISTED and returns how many were affected.
func (r *Repository) AutoUnlistOld() (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -90)
	now := time.Now()
	result := r.db.Model(&db.Listing{}).
		Where("status = 'PUBLISHED' AND created_at < ? AND deleted_at IS NULL", cutoff).
		Updates(map[string]interface{}{
			"status":      "UNLISTED",
			"unlisted_at": now,
		})
	return result.RowsAffected, result.Error
}
