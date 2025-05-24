package models

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents a user's subscription
type Subscription struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	User          User       `gorm:"foreignKey:UserID" json:"-"`
	PlanType      string     `json:"plan_type"` // "student", "entrepreneur", "free", etc.
	Active        bool       `json:"active"`
	TrialStatus   bool       `json:"trial_status"`
	TrialEndsAt   time.Time  `json:"trial_ends_at"`
	EndsAt        *time.Time `json:"ends_at,omitempty"`
	BillingPeriod string     `json:"billing_period,omitempty"` // "monthly", "yearly"
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CanceledAt    *time.Time `json:"canceled_at,omitempty"`
}
