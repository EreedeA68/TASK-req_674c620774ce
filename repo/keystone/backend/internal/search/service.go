package search

import (
	"strings"
	"time"

	"github.com/keystone/backend/pkg/similarity"
	"gorm.io/gorm"
)

// SearchResult is a single combined search result entry.
type SearchResult struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// candidateRow is a minimal projection for search.
type candidateRow struct {
	ID          string
	Status      string
	CreatedAt   time.Time
	Demographics []byte
}

// listingRow is a minimal projection for search.
type listingRow struct {
	ID        string
	Title     string
	Status    string
	CreatedAt time.Time
}

// partRow is a minimal projection for search.
type partRow struct {
	ID        string
	Name      string
	PartNumber string
	Status    string
	CreatedAt time.Time
}

// Service implements combined search logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a new search Service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Search searches across candidates, listings, and parts.
func (s *Service) Search(query string, fuzzy bool, role, callerID, orgID, siteID string) ([]SearchResult, error) {
	var results []SearchResult

	lowerQ := strings.ToLower(strings.TrimSpace(query))
	if lowerQ == "" {
		return results, nil
	}

	// Determine which resource types the role can access.
	canSeeCandidates := role != "INVENTORY_CLERK"
	canSeeParts := role != "" // all roles can see parts except unauth
	canSeeListings := true

	// Build scope suffix for org/site (non-ADMIN only).
	scopeClause := ""
	var scopeArgs []interface{}
	if role != "ADMIN" {
		if orgID != "" {
			scopeClause += " AND organization_id = ?"
			scopeArgs = append(scopeArgs, orgID)
		}
		if siteID != "" {
			scopeClause += " AND site_id = ?"
			scopeArgs = append(scopeArgs, siteID)
		}
	}

	// Search listings.
	if canSeeListings {
		var rows []listingRow
		if fuzzy {
			var all []listingRow
			if err := s.db.Raw("SELECT id, title, status, created_at FROM listings WHERE deleted_at IS NULL AND status NOT IN ('DELETED')"+scopeClause, scopeArgs...).Scan(&all).Error; err != nil {
				return nil, err
			}
			for _, r := range all {
				if similarity.IsSimilar(lowerQ, strings.ToLower(r.Title), 0.6) {
					rows = append(rows, r)
				}
			}
		} else {
			args := append([]interface{}{"%" + lowerQ + "%"}, scopeArgs...)
			if err := s.db.Raw("SELECT id, title, status, created_at FROM listings WHERE deleted_at IS NULL AND status NOT IN ('DELETED') AND title ILIKE ?"+scopeClause, args...).Scan(&rows).Error; err != nil {
				return nil, err
			}
		}
		for _, r := range rows {
			results = append(results, SearchResult{
				Type:      "listing",
				ID:        r.ID,
				Title:     r.Title,
				Status:    r.Status,
				CreatedAt: r.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	// Search parts.
	if canSeeParts {
		var rows []partRow
		if fuzzy {
			var all []partRow
			if err := s.db.Raw("SELECT id, name, part_number, status, created_at FROM parts WHERE deleted_at IS NULL"+scopeClause, scopeArgs...).Scan(&all).Error; err != nil {
				return nil, err
			}
			for _, r := range all {
				if similarity.IsSimilar(lowerQ, strings.ToLower(r.Name), 0.6) ||
					similarity.IsSimilar(lowerQ, strings.ToLower(r.PartNumber), 0.6) {
					rows = append(rows, r)
				}
			}
		} else {
			args := append([]interface{}{"%" + lowerQ + "%", "%" + lowerQ + "%"}, scopeArgs...)
			if err := s.db.Raw("SELECT id, name, part_number, status, created_at FROM parts WHERE deleted_at IS NULL AND (name ILIKE ? OR part_number ILIKE ?)"+scopeClause, args...).Scan(&rows).Error; err != nil {
				return nil, err
			}
		}
		for _, r := range rows {
			title := r.Name
			if r.PartNumber != "" {
				title = r.PartNumber + " - " + r.Name
			}
			results = append(results, SearchResult{
				Type:      "part",
				ID:        r.ID,
				Title:     title,
				Status:    r.Status,
				CreatedAt: r.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	// Search candidates (ID-based, since personal data is in JSONB).
	if canSeeCandidates {
		var rows []candidateRow
		baseQuery := "SELECT id, status, created_at FROM candidates WHERE deleted_at IS NULL AND id::text ILIKE ?"
		args := []interface{}{"%" + lowerQ + "%"}
		// INTAKE_SPECIALIST can only see their own candidates.
		if role == "INTAKE_SPECIALIST" && callerID != "" {
			baseQuery += " AND created_by = ?"
			args = append(args, callerID)
		}
		baseQuery += scopeClause
		args = append(args, scopeArgs...)
		if err := s.db.Raw(baseQuery, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			results = append(results, SearchResult{
				Type:      "candidate",
				ID:        r.ID,
				Title:     "Candidate " + r.ID,
				Status:    r.Status,
				CreatedAt: r.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return results, nil
}
