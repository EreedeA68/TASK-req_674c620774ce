package candidate

import "encoding/json"

// CreateCandidateRequest is the payload for creating or updating a candidate draft.
type CreateCandidateRequest struct {
	Demographics        json.RawMessage `json:"demographics"`
	ExamScores          json.RawMessage `json:"examScores"`
	ApplicationDetails  json.RawMessage `json:"applicationDetails"`
	TransferPreferences json.RawMessage `json:"transferPreferences"`
}

// CandidateDTO is the public representation of a candidate.
type CandidateDTO struct {
	ID                  string          `json:"id"`
	Status              string          `json:"status"`
	CompletenessStatus  string          `json:"completenessStatus"`
	Demographics        json.RawMessage `json:"demographics"`
	ExamScores          json.RawMessage `json:"examScores"`
	ApplicationDetails  json.RawMessage `json:"applicationDetails"`
	TransferPreferences json.RawMessage `json:"transferPreferences"`
	CreatedAt           string          `json:"createdAt"`
	UpdatedAt           string          `json:"updatedAt"`
	SubmittedAt         string          `json:"submittedAt"`
	ReviewedAt          string          `json:"reviewedAt"`
	ReviewerID          string          `json:"reviewerId"`
	ReviewerComments    string          `json:"reviewerComments"`
}

// ApproveRejectRequest is the payload for approving or rejecting a candidate.
type ApproveRejectRequest struct {
	Comments string `json:"comments" validate:"required"`
}

// DocumentDTO is the public representation of a candidate document.
type DocumentDTO struct {
	ID               string `json:"id"`
	FileName         string `json:"fileName"`
	FileSize         int64  `json:"fileSize"`
	MimeType         string `json:"mimeType"`
	SHA256Hash       string `json:"sha256Hash"`
	UploadedAt       string `json:"uploadedAt"`
	WatermarkEnabled bool   `json:"watermarkEnabled"`
}

// CandidateListResponse wraps paginated candidates.
type CandidateListResponse struct {
	Items []CandidateDTO `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}
