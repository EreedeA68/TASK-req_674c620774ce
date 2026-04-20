package db

import (
	"encoding/json"
	"time"
)

// User represents a system user.
type User struct {
	ID             string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username       string          `gorm:"uniqueIndex;not null" json:"username"`
	Email          string          `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash   string          `gorm:"not null" json:"-"`
	Role           string          `gorm:"not null;default:'INTAKE_SPECIALIST'" json:"role"`
	MFAEnabled     bool            `gorm:"default:false" json:"mfaEnabled"`
	MFASecret      string          `gorm:"column:mfa_secret" json:"-"`
	IsLocked       bool            `gorm:"default:false" json:"isLocked"`
	FailedAttempts int             `gorm:"default:0" json:"failedAttempts"`
	LockTime       *time.Time      `gorm:"column:lock_time" json:"lockTime,omitempty"`
	SiteID         string          `gorm:"type:varchar(255);column:site_id" json:"siteId,omitempty"`
	OrganizationID string          `gorm:"type:varchar(255);column:organization_id" json:"organizationId,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	DeletedAt      *time.Time      `gorm:"index" json:"-"`
	Permissions    json.RawMessage `gorm:"type:jsonb;column:permissions" json:"permissions,omitempty"`
}

func (User) TableName() string { return "users" }

// Session represents an active JWT session.
type Session struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      string    `gorm:"type:uuid;not null;index" json:"userId"`
	Token       string    `gorm:"uniqueIndex;not null" json:"token"`
	DeviceID    string    `gorm:"column:device_id" json:"deviceId"`
	IPAddress   string    `gorm:"column:ip_address" json:"ipAddress"`
	ExpiresAt   time.Time `gorm:"column:expires_at" json:"expiresAt"`
	Invalidated bool      `gorm:"default:false" json:"invalidated"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (Session) TableName() string { return "sessions" }

// Candidate represents a candidate application.
type Candidate struct {
	ID                  string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedBy           string          `gorm:"type:uuid;not null" json:"createdBy"`
	Status              string          `gorm:"not null;default:'DRAFT'" json:"status"`
	CompletenessStatus  string          `gorm:"column:completeness_status;default:'incomplete'" json:"completenessStatus"`
	Demographics        json.RawMessage `gorm:"type:jsonb" json:"demographics"`
	ExamScores          json.RawMessage `gorm:"type:jsonb;column:exam_scores" json:"examScores"`
	ApplicationDetails  json.RawMessage `gorm:"type:jsonb;column:application_details" json:"applicationDetails"`
	TransferPreferences json.RawMessage `gorm:"type:jsonb;column:transfer_preferences" json:"transferPreferences"`
	SiteID              string          `gorm:"type:varchar(255);column:site_id" json:"siteId,omitempty"`
	OrganizationID      string          `gorm:"type:varchar(255);column:organization_id" json:"organizationId,omitempty"`
	ReviewerID          *string         `gorm:"type:uuid;column:reviewer_id" json:"reviewerId,omitempty"`
	ReviewerComments    string          `gorm:"column:reviewer_comments" json:"reviewerComments"`
	SubmittedAt         *time.Time      `gorm:"column:submitted_at" json:"submittedAt,omitempty"`
	ReviewedAt          *time.Time      `gorm:"column:reviewed_at" json:"reviewedAt,omitempty"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
	DeletedAt           *time.Time      `gorm:"index" json:"-"`
}

func (Candidate) TableName() string { return "candidates" }

// CandidateDocument represents a document attached to a candidate.
type CandidateDocument struct {
	ID               string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CandidateID      string    `gorm:"type:uuid;not null;index;column:candidate_id" json:"candidateId"`
	FileName         string    `gorm:"column:file_name;not null" json:"fileName"`
	FilePath         string    `gorm:"column:file_path;not null" json:"-"`
	FileSize         int64     `gorm:"column:file_size" json:"fileSize"`
	MimeType         string    `gorm:"column:mime_type" json:"mimeType"`
	SHA256Hash       string    `gorm:"column:sha256_hash;uniqueIndex" json:"sha256Hash"`
	UploaderID       string    `gorm:"type:uuid;column:uploader_id" json:"uploaderId"`
	WatermarkEnabled bool      `gorm:"column:watermark_enabled;default:false" json:"watermarkEnabled"`
	UploadedAt       time.Time `gorm:"column:uploaded_at;autoCreateTime" json:"uploadedAt"`
}

func (CandidateDocument) TableName() string { return "candidate_documents" }

// Listing represents a lost & found listing.
type Listing struct {
	ID                  string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedBy           string     `gorm:"type:uuid;not null;column:created_by" json:"createdBy"`
	Title               string     `gorm:"not null" json:"title"`
	Category            string     `gorm:"not null" json:"category"`
	LocationDescription string     `gorm:"column:location_description;not null" json:"locationDescription"`
	Status              string     `gorm:"not null;default:'PUBLISHED'" json:"status"`
	IsDuplicateFlagged  bool       `gorm:"column:is_duplicate_flagged;default:false" json:"isDuplicateFlagged"`
	SiteID              string     `gorm:"type:varchar(255);column:site_id" json:"siteId,omitempty"`
	OrganizationID      string     `gorm:"type:varchar(255);column:organization_id" json:"organizationId,omitempty"`
	TimeWindowStart     *time.Time `gorm:"column:time_window_start" json:"timeWindowStart,omitempty"`
	TimeWindowEnd       *time.Time `gorm:"column:time_window_end" json:"timeWindowEnd,omitempty"`
	UnlistedAt          *time.Time `gorm:"column:unlisted_at" json:"unlistedAt,omitempty"`
	UnlistedBy          *string    `gorm:"type:uuid;column:unlisted_by" json:"unlistedBy,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	DeletedAt           *time.Time `gorm:"index" json:"-"`
}

func (Listing) TableName() string { return "listings" }

// Part represents an automotive part.
type Part struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PartNumber       string     `gorm:"uniqueIndex;not null;column:part_number" json:"partNumber"`
	Name             string     `gorm:"not null" json:"name"`
	Description      string     `json:"description"`
	Status           string     `gorm:"not null;default:'ACTIVE'" json:"status"`
	CurrentVersionID *string    `gorm:"type:uuid;column:current_version_id" json:"currentVersionId,omitempty"`
	SiteID           string     `gorm:"type:varchar(255);column:site_id" json:"siteId,omitempty"`
	OrganizationID   string     `gorm:"type:varchar(255);column:organization_id" json:"organizationId,omitempty"`
	CreatedBy        string     `gorm:"type:uuid;column:created_by" json:"createdBy"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	DeletedAt        *time.Time `gorm:"index" json:"-"`
}

func (Part) TableName() string { return "parts" }

// PartVersion represents a version of a part's technical data.
type PartVersion struct {
	ID            string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PartID        string          `gorm:"type:uuid;not null;index;column:part_id" json:"partId"`
	VersionNumber int             `gorm:"column:version_number;not null" json:"versionNumber"`
	Fitment       json.RawMessage `gorm:"type:jsonb" json:"fitment"`
	OEMMappings   json.RawMessage `gorm:"type:jsonb;column:oem_mappings" json:"oemMappings"`
	Attributes    json.RawMessage `gorm:"type:jsonb" json:"attributes"`
	ChangeSummary string          `gorm:"column:change_summary" json:"changeSummary"`
	ChangedBy     string          `gorm:"type:uuid;column:changed_by" json:"changedBy"`
	CreatedAt     time.Time       `json:"createdAt"`
}

func (PartVersion) TableName() string { return "part_versions" }

// PartFitment represents a fitment entry for a part.
type PartFitment struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PartID    string    `gorm:"type:uuid;not null;index;column:part_id" json:"partId"`
	Make      string    `json:"make"`
	Model     string    `json:"model"`
	YearStart int       `gorm:"column:year_start" json:"yearStart"`
	YearEnd   int       `gorm:"column:year_end" json:"yearEnd"`
	Engine    string    `json:"engine"`
	CreatedAt time.Time `json:"createdAt"`
}

func (PartFitment) TableName() string { return "part_fitments" }

// AuditLog represents an immutable audit trail entry.
type AuditLog struct {
	ID           string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ActorID      string          `gorm:"type:uuid;column:actor_id;index" json:"actorId"`
	Action       string          `gorm:"not null" json:"action"`
	ResourceType string          `gorm:"column:resource_type;not null" json:"resourceType"`
	ResourceID   string          `gorm:"type:uuid;column:resource_id;index" json:"resourceId"`
	DeviceID     string          `gorm:"column:device_id" json:"deviceId"`
	IPAddress    string          `gorm:"column:ip_address" json:"ipAddress"`
	BeforeState  json.RawMessage `gorm:"type:jsonb;column:before_state" json:"beforeState,omitempty"`
	AfterState   json.RawMessage `gorm:"type:jsonb;column:after_state" json:"afterState,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// DownloadPermission represents explicit download access for a resource.
type DownloadPermission struct {
	ID           string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       string     `gorm:"type:uuid;not null;index;column:user_id" json:"userId"`
	ResourceType string     `gorm:"column:resource_type;not null" json:"resourceType"`
	ResourceID   string     `gorm:"type:uuid;column:resource_id;index" json:"resourceId"`
	GrantedBy    string     `gorm:"type:uuid;column:granted_by" json:"grantedBy"`
	ExpiresAt    *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func (DownloadPermission) TableName() string { return "download_permissions" }

// DownloadLog records each document download event.
type DownloadLog struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       string    `gorm:"type:uuid;not null;index;column:user_id" json:"userId"`
	ResourceType string    `gorm:"column:resource_type;not null" json:"resourceType"`
	ResourceID   string    `gorm:"type:uuid;column:resource_id;index" json:"resourceId"`
	DeviceID     string    `gorm:"column:device_id" json:"deviceId"`
	DownloadedAt time.Time `gorm:"column:downloaded_at;autoCreateTime" json:"downloadedAt"`
}

func (DownloadLog) TableName() string { return "download_logs" }
