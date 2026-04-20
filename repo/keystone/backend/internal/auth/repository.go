package auth

import (
	"time"

	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// Repository handles all auth-related DB operations.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new auth Repository.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{db: database}
}

// GetUserByUsername retrieves a user by their username.
func (r *Repository) GetUserByUsername(username string) (*db.User, error) {
	var user db.User
	if err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by their UUID.
func (r *Repository) GetUserByID(id string) (*db.User, error) {
	var user db.User
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser persists changes to a user record.
func (r *Repository) UpdateUser(user *db.User) error {
	return r.db.Save(user).Error
}

// CreateUser inserts a new user.
func (r *Repository) CreateUser(user *db.User) error {
	return r.db.Create(user).Error
}

// CreateSession inserts a new session record.
func (r *Repository) CreateSession(session *db.Session) error {
	return r.db.Create(session).Error
}

// InvalidateSession marks a session token as invalid.
func (r *Repository) InvalidateSession(token string) error {
	return r.db.Model(&db.Session{}).
		Where("token = ?", token).
		Update("invalidated", true).Error
}

// UpdateFailedAttempts updates the failed login counter and optional lock time for a user.
func (r *Repository) UpdateFailedAttempts(userID string, attempts int, lockTime *time.Time) error {
	updates := map[string]interface{}{
		"failed_attempts": attempts,
	}
	if lockTime != nil {
		updates["is_locked"] = true
		updates["lock_time"] = lockTime
	} else if attempts == 0 {
		updates["is_locked"] = false
		updates["lock_time"] = nil
	}
	return r.db.Model(&db.User{}).Where("id = ?", userID).Updates(updates).Error
}

// ListUsers returns all non-deleted users.
func (r *Repository) ListUsers() ([]db.User, error) {
	var users []db.User
	if err := r.db.Where("deleted_at IS NULL").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
