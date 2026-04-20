package lostfound

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/keystone/backend/internal/db"
	"github.com/keystone/backend/pkg/similarity"
)

var usLocationRe = regexp.MustCompile(`(?i)^[A-Za-z\s\.\-]+,\s*[A-Z]{2}$`)

const duplicateThreshold = 0.8

var allowedCategories = map[string]bool{
	"accessories": true,
	"bags":        true,
	"clothing":    true,
	"documents":   true,
	"electronics": true,
	"jewelry":     true,
	"keys":        true,
	"other":       true,
	"pets":        true,
	"wallet":      true,
}

// AuditLogger is the minimal interface for writing audit events.
type AuditLogger interface {
	Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error
}

// Service implements business logic for lost & found listings.
type Service struct {
	repo  *Repository
	audit AuditLogger
}

// NewService creates a new lostfound Service.
func NewService(repo *Repository, audit AuditLogger) *Service {
	return &Service{repo: repo, audit: audit}
}

// CreateListing validates and creates a new listing.
func (s *Service) CreateListing(createdBy, siteID, orgID, deviceID, ip string, req CreateListingRequest) (*ListingDTO, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("title is required")
	}
	cat := strings.ToLower(strings.TrimSpace(req.Category))
	if cat == "" {
		return nil, errors.New("category is required")
	}
	if !allowedCategories[cat] {
		return nil, errors.New("invalid category; allowed: accessories, bags, clothing, documents, electronics, jewelry, keys, other, pets, wallet")
	}
	req.Category = cat

	loc := strings.TrimSpace(req.LocationDescription)
	if loc == "" {
		return nil, errors.New("location description is required")
	}
	if !usLocationRe.MatchString(loc) {
		return nil, errors.New("location must be in \"City, ST\" format, e.g. \"Austin, TX\"")
	}
	if req.TimeWindowStart.IsZero() || req.TimeWindowEnd.IsZero() {
		return nil, errors.New("time_window_start and time_window_end are required")
	}
	if req.TimeWindowEnd.Before(req.TimeWindowStart) {
		return nil, errors.New("time_window_end must be after time_window_start")
	}

	status := "PUBLISHED"

	listing := &db.Listing{
		CreatedBy:           createdBy,
		SiteID:              siteID,
		OrganizationID:      orgID,
		Title:               req.Title,
		Category:            req.Category,
		LocationDescription: loc,
		Status:              status,
		TimeWindowStart:     &req.TimeWindowStart,
		TimeWindowEnd:       &req.TimeWindowEnd,
	}

	// Duplicate detection: check similar titles in same category within 24h.
	similars, err := s.repo.GetSimilarListings(req.Category, true)
	if err == nil {
		for _, existing := range similars {
			if similarity.IsSimilar(strings.ToLower(req.Title), strings.ToLower(existing.Title), duplicateThreshold) {
				listing.IsDuplicateFlagged = true
				listing.Status = "PENDING_REVIEW"
				break
			}
		}
	}

	if err := s.repo.CreateListing(listing); err != nil {
		return nil, err
	}

	_ = s.audit.Log(createdBy, "LISTING_CREATED", "listing", listing.ID, deviceID, ip, nil, listing)

	return toDTO(listing), nil
}

// GetListings returns paginated listings, filtered by the caller's role.
func (s *Service) GetListings(role string, filters map[string]string, page, limit int) ([]ListingDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var visibleStatuses []string
	switch role {
	case "ADMIN", "AUDITOR":
		// Can see all statuses.
		visibleStatuses = nil
	default:
		visibleStatuses = []string{"PUBLISHED"}
	}

	listings, total, err := s.repo.GetListings(filters, visibleStatuses, page, limit)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]ListingDTO, len(listings))
	for i, l := range listings {
		dtos[i] = *toDTO(&l)
	}
	return dtos, total, nil
}

// GetListing retrieves a single listing, respecting role-based status visibility and org/site scope.
func (s *Service) GetListing(id, role, orgID, siteID string) (*ListingDTO, error) {
	listing, err := s.repo.GetListingByID(id)
	if err != nil {
		return nil, err
	}
	// Non-admin/auditor roles can only see PUBLISHED listings.
	if role != "ADMIN" && role != "AUDITOR" && listing.Status != "PUBLISHED" {
		return nil, errors.New("listing not found")
	}
	// Org/site scope enforcement.
	if role != "ADMIN" && orgID != "" && listing.OrganizationID != orgID {
		return nil, errors.New("listing not found")
	}
	if role != "ADMIN" && siteID != "" && listing.SiteID != siteID {
		return nil, errors.New("listing not found")
	}
	return toDTO(listing), nil
}

// EditListing updates an existing listing. Only the creator or an ADMIN may edit.
func (s *Service) EditListing(id, editorID, role, deviceID, ip string, req CreateListingRequest) (*ListingDTO, error) {
	listing, err := s.repo.GetListingByID(id)
	if err != nil {
		return nil, err
	}

	if listing.Status == "DELETED" {
		return nil, errors.New("cannot edit a deleted listing")
	}

	if listing.CreatedBy != editorID && role != "ADMIN" {
		return nil, errors.New("not authorized to edit this listing")
	}

	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("title is required")
	}
	cat := strings.ToLower(strings.TrimSpace(req.Category))
	if cat == "" {
		return nil, errors.New("category is required")
	}
	if !allowedCategories[cat] {
		return nil, errors.New("invalid category; allowed: accessories, bags, clothing, documents, electronics, jewelry, keys, other, pets, wallet")
	}
	loc := strings.TrimSpace(req.LocationDescription)
	if loc == "" {
		return nil, errors.New("location description is required")
	}
	if !usLocationRe.MatchString(loc) {
		return nil, errors.New("location must be in \"City, ST\" format, e.g. \"Austin, TX\"")
	}

	before := *listing
	listing.Title = req.Title
	listing.Category = cat
	listing.LocationDescription = loc

	if !req.TimeWindowStart.IsZero() {
		listing.TimeWindowStart = &req.TimeWindowStart
	}
	if !req.TimeWindowEnd.IsZero() {
		listing.TimeWindowEnd = &req.TimeWindowEnd
	}

	if listing.TimeWindowStart != nil && listing.TimeWindowEnd != nil &&
		listing.TimeWindowEnd.Before(*listing.TimeWindowStart) {
		return nil, errors.New("time_window_end must be after time_window_start")
	}

	if err := s.repo.UpdateListing(listing); err != nil {
		return nil, err
	}

	_ = s.audit.Log(editorID, "LISTING_UPDATED", "listing", listing.ID, deviceID, ip, before, listing)
	return toDTO(listing), nil
}

// UnlistListing changes a PUBLISHED listing to UNLISTED.
func (s *Service) UnlistListing(id, unlistBy, deviceID, ip string) error {
	listing, err := s.repo.GetListingByID(id)
	if err != nil {
		return err
	}

	if listing.Status != "PUBLISHED" {
		return errors.New("only PUBLISHED listings can be unlisted")
	}

	now := time.Now()
	listing.Status = "UNLISTED"
	listing.UnlistedAt = &now
	listing.UnlistedBy = &unlistBy

	if err := s.repo.UpdateListing(listing); err != nil {
		return err
	}

	_ = s.audit.Log(unlistBy, "LISTING_UNLISTED", "listing", listing.ID, deviceID, ip, nil, listing)
	return nil
}

// DeleteListing soft-deletes a listing (status = DELETED).
func (s *Service) DeleteListing(id, actorID, deviceID, ip string) error {
	listing, err := s.repo.GetListingByID(id)
	if err != nil {
		return err
	}
	before := *listing
	listing.Status = "DELETED"
	if err := s.repo.UpdateListing(listing); err != nil {
		return err
	}
	_ = s.audit.Log(actorID, "LISTING_DELETED", "listing", listing.ID, deviceID, ip, before, listing)
	return nil
}

// OverrideDuplicate clears the duplicate flag and restores PUBLISHED status.
func (s *Service) OverrideDuplicate(id, reviewerID string) error {
	listing, err := s.repo.GetListingByID(id)
	if err != nil {
		return err
	}

	listing.IsDuplicateFlagged = false
	if listing.Status == "PENDING_REVIEW" {
		listing.Status = "PUBLISHED"
	}

	if err := s.repo.UpdateListing(listing); err != nil {
		return err
	}

	_ = s.audit.Log(reviewerID, "LISTING_DUPLICATE_OVERRIDDEN", "listing", listing.ID, "", "", nil, listing)
	return nil
}

// RunAutoUnlistJob unlists all listings older than 90 days and logs the result.
func (s *Service) RunAutoUnlistJob() {
	count, err := s.repo.AutoUnlistOld()
	if err != nil {
		log.Printf("[AutoUnlist] error: %v", err)
		return
	}
	if count > 0 {
		log.Printf("[AutoUnlist] unlisted %d listings older than 90 days", count)
		_ = s.audit.Log("system", "AUTO_UNLIST", "listing", "", "", "", nil, map[string]interface{}{"count": count})
	}
}

// toDTO converts a db.Listing to a ListingDTO.
func toDTO(l *db.Listing) *ListingDTO {
	dto := &ListingDTO{
		ID:                  l.ID,
		Title:               l.Title,
		Category:            l.Category,
		LocationDescription: l.LocationDescription,
		Status:              l.Status,
		IsDuplicateFlagged:  l.IsDuplicateFlagged,
		CreatedAt:           l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           l.UpdatedAt.Format(time.RFC3339),
	}
	if l.TimeWindowStart != nil {
		dto.TimeWindowStart = l.TimeWindowStart.Format(time.RFC3339)
	}
	if l.TimeWindowEnd != nil {
		dto.TimeWindowEnd = l.TimeWindowEnd.Format(time.RFC3339)
	}
	return dto
}
