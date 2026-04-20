package parts

import "encoding/json"

// CreatePartRequest is the payload for creating or updating a part.
type CreatePartRequest struct {
	PartNumber    string          `json:"partNumber" validate:"required"`
	Name          string          `json:"name" validate:"required"`
	Description   string          `json:"description"`
	Fitment       json.RawMessage `json:"fitment"`
	OEMMappings   json.RawMessage `json:"oemMappings"`
	Attributes    json.RawMessage `json:"attributes"`
	ChangeSummary string          `json:"changeSummary"`
}

// PartDTO is the public representation of a part.
type PartDTO struct {
	ID               string `json:"id"`
	PartNumber       string `json:"partNumber"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	CurrentVersionID string `json:"currentVersionId"`
	VersionNumber    int    `json:"versionNumber"`
	CreatedAt        string `json:"createdAt"`
}

// PartVersionDTO is the public representation of a part version.
type PartVersionDTO struct {
	ID            string          `json:"id"`
	PartID        string          `json:"partId"`
	VersionNumber int             `json:"versionNumber"`
	Fitment       json.RawMessage `json:"fitment"`
	OEMMappings   json.RawMessage `json:"oemMappings"`
	Attributes    json.RawMessage `json:"attributes"`
	ChangeSummary string          `json:"changeSummary"`
	ChangedBy     string          `json:"changedBy"`
	CreatedAt     string          `json:"createdAt"`
}

// DiffField describes the change in a single field between two versions.
type DiffField struct {
	OldValue interface{} `json:"oldValue"`
	NewValue interface{} `json:"newValue"`
	Changed  bool        `json:"changed"`
}

// VersionCompareDTO holds the comparison result between two part versions.
type VersionCompareDTO struct {
	VersionA PartVersionDTO       `json:"versionA"`
	VersionB PartVersionDTO       `json:"versionB"`
	Diff     map[string]DiffField `json:"diff"`
}

// CSVImportRow represents a single row parsed from a CSV import.
type CSVImportRow struct {
	PartNumber      string `csv:"part_number"`
	Name            string `csv:"name"`
	Description     string `csv:"description"`
	FitmentJSON     string `csv:"fitment"`
	OEMMappingsJSON string `csv:"oem_mappings"`
	AttributesJSON  string `csv:"attributes"`
}

// PartListResponse wraps paginated parts.
type PartListResponse struct {
	Items []PartDTO `json:"items"`
	Total int64     `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}
