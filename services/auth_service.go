package services

import (
	"assistdeck/config"
	"assistdeck/models"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthService struct {
	DB     *gorm.DB
	Config *config.Config
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{DB: db, Config: cfg}
}

func (s *AuthService) CreateUser(user *models.User) error {
	return s.DB.Create(user).Error
}

func (s *AuthService) GenerateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days token
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.Config.JWTSecret))
}

func (s *AuthService) FindOrCreateUser(email string, name string) (*models.User, error) {
	var user models.User
	err := s.DB.Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create user if not found
			user = models.User{
				Email: email,
				Name:  name,
			}
			if err := s.DB.Create(&user).Error; err != nil {
				return nil, err
			}
			return &user, nil
		}
		return nil, err
	}

	return &user, nil
}
