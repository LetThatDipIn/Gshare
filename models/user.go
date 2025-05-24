package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name               string    `gorm:"type:text"`
	Email              string    `gorm:"uniqueIndex;type:text"`
	Role               string    `gorm:"type:text"` // "student" or "entrepreneur"
	Plan               string    `gorm:"type:text"` // free / trial / student / entrepreneur
	TrialEndsAt        *time.Time
	SubscriptionEndsAt *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func MigrateDB(db *gorm.DB) {
	// Create the uuid-ossp extension if it doesn't exist
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")

	err := db.AutoMigrate(&User{}, &Subscription{})
	if err != nil {
		panic("❌ Failed to migrate database: " + err.Error())
	}
}
