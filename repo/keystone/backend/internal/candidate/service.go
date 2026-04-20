package candidate

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

const (
	maxFileSizeBytes = 20 * 1024 * 1024 // 20 MB
)

var allowedMIMETypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

// AuditLogger is the minimal interface needed for writing audit events.
type AuditLogger interface {
	Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error
}

// Service implements business logic for candidates.
type Service struct {
	repo  *Repository
	audit AuditLogger
}

// NewService creates a new candidate Service.
func NewService(repo *Repository, audit AuditLogger) *Service {
	return &Service{repo: repo, audit: audit}
}

// CreateDraft creates a new candidate draft.
func (s *Service) CreateDraft(createdBy, siteID, orgID, deviceID, ip string, req CreateCandidateRequest) (*CandidateDTO, error) {
	candidate := &db.Candidate{
		CreatedBy:           createdBy,
		SiteID:              siteID,
		OrganizationID:      orgID,
		Status:              "DRAFT",
		CompletenessStatus:  ComputeCompleteness(req),
		Demographics:        req.Demographics,
		ExamScores:          req.ExamScores,
		ApplicationDetails:  req.ApplicationDetails,
		TransferPreferences: req.TransferPreferences,
	}

	if err := s.repo.CreateCandidate(candidate); err != nil {
		return nil, err
	}

	_ = s.audit.Log(createdBy, "CANDIDATE_CREATED", "candidate", candidate.ID, deviceID, ip, nil, candidate)

	return toDTO(candidate), nil
}

// GetCandidates returns paginated candidates scoped by the caller's role.
func (s *Service) GetCandidates(callerID, role string, filters map[string]string, page, limit int) ([]CandidateDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// INTAKE_SPECIALIST can only see their own candidates.
	if role == "INTAKE_SPECIALIST" {
		filters["createdBy"] = callerID
	}

	candidates, total, err := s.repo.GetCandidates(filters, page, limit)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]CandidateDTO, len(candidates))
	for i, c := range candidates {
		dtos[i] = *toDTO(&c)
	}
	return dtos, total, nil
}

// GetCandidate retrieves a single candidate, enforcing ownership for INTAKE_SPECIALIST and org/site scope.
func (s *Service) GetCandidate(id, callerID, role, orgID, siteID string) (*CandidateDTO, error) {
	candidate, err := s.repo.GetCandidateByID(id)
	if err != nil {
		return nil, err
	}
	if role == "INTAKE_SPECIALIST" && candidate.CreatedBy != callerID {
		return nil, errors.New("not authorized to view this candidate")
	}
	if role != "ADMIN" && orgID != "" && candidate.OrganizationID != orgID {
		return nil, gorm.ErrRecordNotFound
	}
	if role != "ADMIN" && siteID != "" && candidate.SiteID != siteID {
		return nil, gorm.ErrRecordNotFound
	}
	return toDTO(candidate), nil
}

// UpdateDraft updates a candidate in DRAFT status. Only the creator or an ADMIN may update.
func (s *Service) UpdateDraft(id, callerID, role, deviceID, ip string, req CreateCandidateRequest) (*CandidateDTO, error) {
	candidate, err := s.repo.GetCandidateByID(id)
	if err != nil {
		return nil, err
	}

	if candidate.Status != "DRAFT" {
		return nil, errors.New("only DRAFT candidates can be updated")
	}

	if candidate.CreatedBy != callerID && role != "ADMIN" {
		return nil, errors.New("not authorized to update this candidate")
	}

	before := *candidate
	candidate.Demographics = req.Demographics
	candidate.ExamScores = req.ExamScores
	candidate.ApplicationDetails = req.ApplicationDetails
	candidate.TransferPreferences = req.TransferPreferences
	candidate.CompletenessStatus = ComputeCompleteness(req)

	if err := s.repo.UpdateCandidate(candidate); err != nil {
		return nil, err
	}

	_ = s.audit.Log(callerID, "CANDIDATE_UPDATED", "candidate", candidate.ID, deviceID, ip, before, candidate)

	return toDTO(candidate), nil
}

// Submit changes a candidate status from DRAFT to SUBMITTED.
func (s *Service) Submit(id, submittedBy, role, deviceID, ip string) error {
	candidate, err := s.repo.GetCandidateByID(id)
	if err != nil {
		return err
	}

	if role != "ADMIN" && candidate.CreatedBy != submittedBy {
		return errors.New("not authorized to submit this candidate")
	}

	if candidate.Status != "DRAFT" {
		return errors.New("only DRAFT candidates can be submitted")
	}

	if candidate.CompletenessStatus != "complete" {
		return errors.New("candidate record is not complete")
	}

	docCount, err := s.repo.CountDocumentsByCandidateID(id)
	if err != nil {
		return err
	}
	if docCount == 0 {
		return errors.New("at least one document must be uploaded before submission")
	}

	now := time.Now()
	candidate.Status = "SUBMITTED"
	candidate.SubmittedAt = &now

	if err := s.repo.UpdateCandidate(candidate); err != nil {
		return err
	}

	_ = s.audit.Log(submittedBy, "CANDIDATE_SUBMITTED", "candidate", candidate.ID, deviceID, ip, nil, candidate)
	return nil
}

// Approve marks a SUBMITTED candidate as APPROVED.
func (s *Service) Approve(id, reviewerID, comments, deviceID, ip string) error {
	if comments == "" {
		return errors.New("approval comments are required")
	}

	candidate, err := s.repo.GetCandidateByID(id)
	if err != nil {
		return err
	}

	if candidate.Status != "SUBMITTED" {
		return errors.New("only SUBMITTED candidates can be approved")
	}

	now := time.Now()
	candidate.Status = "APPROVED"
	candidate.ReviewerID = &reviewerID
	candidate.ReviewerComments = comments
	candidate.ReviewedAt = &now

	if err := s.repo.UpdateCandidate(candidate); err != nil {
		return err
	}

	_ = s.audit.Log(reviewerID, "CANDIDATE_APPROVED", "candidate", candidate.ID, deviceID, ip, nil, candidate)
	return nil
}

// Reject marks a SUBMITTED candidate as REJECTED.
func (s *Service) Reject(id, reviewerID, comments, deviceID, ip string) error {
	if comments == "" {
		return errors.New("rejection comments are required")
	}

	candidate, err := s.repo.GetCandidateByID(id)
	if err != nil {
		return err
	}

	if candidate.Status != "SUBMITTED" {
		return errors.New("only SUBMITTED candidates can be rejected")
	}

	now := time.Now()
	candidate.Status = "REJECTED"
	candidate.ReviewerID = &reviewerID
	candidate.ReviewerComments = comments
	candidate.ReviewedAt = &now

	if err := s.repo.UpdateCandidate(candidate); err != nil {
		return err
	}

	_ = s.audit.Log(reviewerID, "CANDIDATE_REJECTED", "candidate", candidate.ID, deviceID, ip, nil, candidate)
	return nil
}

// UploadDocument validates and stores a candidate document.
func (s *Service) UploadDocument(candidateID, uploaderID, uploaderRole, deviceID, ip string, file multipart.File, header *multipart.FileHeader) (*DocumentDTO, error) {
	// Ownership check: only the candidate creator or ADMIN may upload documents.
	candidate, err := s.repo.GetCandidateByID(candidateID)
	if err != nil {
		return nil, err
	}
	if uploaderRole != "ADMIN" && candidate.CreatedBy != uploaderID {
		return nil, errors.New("not authorized to upload documents for this candidate")
	}

	// Validate file size before reading.
	if header.Size > maxFileSizeBytes {
		return nil, fmt.Errorf("file too large; max size is 20MB")
	}

	// Read all bytes for server-side MIME detection and hashing.
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Server-side MIME detection — do not trust the client Content-Type header.
	sniff := http.DetectContentType(data)
	mimeType := strings.SplitN(sniff, ";", 2)[0]
	if !allowedMIMETypes[mimeType] {
		return nil, errors.New("unsupported file type; only PDF, JPG, PNG allowed")
	}

	// Compute SHA-256 hash.
	hashBytes := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", hashBytes)

	// Check for duplicates.
	if _, err := s.repo.GetDocumentByHash(hash); err == nil {
		return nil, errors.New("duplicate document: file already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Save to disk using a sanitized, hash-based storage name to prevent path traversal.
	docsPath := os.Getenv("DOCUMENTS_PATH")
	if docsPath == "" {
		docsPath = "/tmp/keystone/documents"
	}
	dir := filepath.Join(docsPath, candidateID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	ext := filepath.Ext(filepath.Base(header.Filename))
	storageName := hash[:16] + ext
	filePath := filepath.Join(dir, storageName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	originalName := filepath.Base(header.Filename)
	doc := &db.CandidateDocument{
		CandidateID: candidateID,
		FileName:    originalName,
		FilePath:    filePath,
		FileSize:    header.Size,
		MimeType:    mimeType,
		SHA256Hash:  hash,
		UploaderID:  uploaderID,
	}

	if err := s.repo.CreateDocument(doc); err != nil {
		// Cleanup orphan file.
		_ = os.Remove(filePath)
		return nil, err
	}

	_ = s.audit.Log(uploaderID, "DOCUMENT_UPLOADED", "candidate_document", doc.ID, deviceID, ip, nil, doc)

	return &DocumentDTO{
		ID:               doc.ID,
		FileName:         doc.FileName,
		FileSize:         doc.FileSize,
		MimeType:         doc.MimeType,
		SHA256Hash:       doc.SHA256Hash,
		UploadedAt:       doc.UploadedAt.Format(time.RFC3339),
		WatermarkEnabled: doc.WatermarkEnabled,
	}, nil
}

// GetDocuments returns all documents for a candidate.
func (s *Service) GetDocuments(candidateID string) ([]DocumentDTO, error) {
	docs, err := s.repo.GetDocumentsByCandidateID(candidateID)
	if err != nil {
		return nil, err
	}
	dtos := make([]DocumentDTO, len(docs))
	for i, d := range docs {
		dtos[i] = DocumentDTO{
			ID:               d.ID,
			FileName:         d.FileName,
			FileSize:         d.FileSize,
			MimeType:         d.MimeType,
			SHA256Hash:       d.SHA256Hash,
			UploadedAt:       d.UploadedAt.Format(time.RFC3339),
			WatermarkEnabled: d.WatermarkEnabled,
		}
	}
	return dtos, nil
}

// ComputeCompleteness determines whether a candidate has all required fields.
func ComputeCompleteness(req CreateCandidateRequest) string {
	if len(req.Demographics) == 0 || string(req.Demographics) == "null" {
		return "incomplete"
	}
	if len(req.ExamScores) == 0 || string(req.ExamScores) == "null" {
		return "incomplete"
	}
	if len(req.ApplicationDetails) == 0 || string(req.ApplicationDetails) == "null" {
		return "incomplete"
	}
	return "complete"
}

// toDTO converts a db.Candidate to a CandidateDTO.
func toDTO(c *db.Candidate) *CandidateDTO {
	dto := &CandidateDTO{
		ID:                  c.ID,
		Status:              c.Status,
		CompletenessStatus:  c.CompletenessStatus,
		Demographics:        c.Demographics,
		ExamScores:          c.ExamScores,
		ApplicationDetails:  c.ApplicationDetails,
		TransferPreferences: c.TransferPreferences,
		CreatedAt:           c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           c.UpdatedAt.Format(time.RFC3339),
		ReviewerComments:    c.ReviewerComments,
	}
	if c.ReviewerID != nil {
		dto.ReviewerID = *c.ReviewerID
	}
	if c.SubmittedAt != nil {
		dto.SubmittedAt = c.SubmittedAt.Format(time.RFC3339)
	}
	if c.ReviewedAt != nil {
		dto.ReviewedAt = c.ReviewedAt.Format(time.RFC3339)
	}
	return dto
}
