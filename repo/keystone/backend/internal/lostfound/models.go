package lostfound

import "time"

// CreateListingRequest is the payload for creating or updating a listing.
type CreateListingRequest struct {
	Title               string    `json:"title" validate:"required"`
	Category            string    `json:"category" validate:"required"`
	LocationDescription string    `json:"locationDescription" validate:"required"`
	TimeWindowStart     time.Time `json:"timeWindowStart"`
	TimeWindowEnd       time.Time `json:"timeWindowEnd"`
}

// ListingDTO is the public representation of a lost & found listing.
type ListingDTO struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	Category            string `json:"category"`
	LocationDescription string `json:"locationDescription"`
	Status              string `json:"status"`
	IsDuplicateFlagged  bool   `json:"isDuplicateFlagged"`
	TimeWindowStart     string `json:"timeWindowStart"`
	TimeWindowEnd       string `json:"timeWindowEnd"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

// ListingListResponse wraps paginated listings.
type ListingListResponse struct {
	Items []ListingDTO `json:"items"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}
