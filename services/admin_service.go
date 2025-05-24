package services

import (
	"assistdeck/models"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"gorm.io/gorm"
)

type AdminService struct {
	DB        *gorm.DB
	SecretKey []byte // For encrypting sensitive settings
}

func NewAdminService(db *gorm.DB, secretKey string) *AdminService {
	// Use the JWT secret as the encryption key for settings
	// We need exactly 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(secretKey))

	return &AdminService{
		DB:        db,
		SecretKey: key,
	}
}

// GetSetting retrieves a setting by key
func (s *AdminService) GetSetting(key string) (*models.AdminSettings, error) {
	var setting models.AdminSettings
	err := s.DB.Where("setting_key = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}

	// Decrypt the value if it's encrypted
	if setting.IsEncrypted {
		decryptedValue, err := s.decrypt(setting.SettingValue)
		if err != nil {
			return nil, err
		}
		setting.SettingValue = decryptedValue
	}

	return &setting, nil
}

// GetSettings retrieves all settings, optionally filtered by category
func (s *AdminService) GetSettings(category string) ([]models.AdminSettings, error) {
	var settings []models.AdminSettings
	query := s.DB.Order("setting_category, setting_key")

	if category != "" {
		query = query.Where("setting_category = ?", category)
	}

	if err := query.Find(&settings).Error; err != nil {
		return nil, err
	}

	// Decrypt encrypted values
	for i, setting := range settings {
		if setting.IsEncrypted {
			decryptedValue, err := s.decrypt(setting.SettingValue)
			if err != nil {
				continue // Skip this setting if decryption fails
			}
			settings[i].SettingValue = decryptedValue
		}
	}

	return settings, nil
}

// SaveSetting creates or updates a setting
func (s *AdminService) SaveSetting(key, value, category, description string, isEncrypted bool) (*models.AdminSettings, error) {
	var setting models.AdminSettings
	result := s.DB.Where("setting_key = ?", key).First(&setting)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Encrypt the value if needed
	valueToStore := value
	if isEncrypted {
		var err error
		valueToStore, err = s.encrypt(value)
		if err != nil {
			return nil, err
		}
	}

	// Create or update the setting
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Create new setting
		setting = models.AdminSettings{
			SettingKey:      key,
			SettingValue:    valueToStore,
			SettingCategory: category,
			Description:     description,
			IsEncrypted:     isEncrypted,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := s.DB.Create(&setting).Error; err != nil {
			return nil, err
		}
	} else {
		// Update existing setting
		setting.SettingValue = valueToStore
		setting.SettingCategory = category
		setting.Description = description
		setting.IsEncrypted = isEncrypted
		setting.UpdatedAt = time.Now()
		if err := s.DB.Save(&setting).Error; err != nil {
			return nil, err
		}
	}

	// Return the setting with the decrypted value
	setting.SettingValue = value
	return &setting, nil
}

// DeleteSetting deletes a setting by key
func (s *AdminService) DeleteSetting(key string) error {
	result := s.DB.Where("setting_key = ?", key).Delete(&models.AdminSettings{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetUserStats returns statistics about users
func (s *AdminService) GetUserStats() (map[string]interface{}, error) {
	var totalUsers int64
	var activeUsers int64
	var newUsersLast30Days int64

	// Total users
	if err := s.DB.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return nil, err
	}

	// Active users (with activity in the last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := s.DB.Model(&models.User{}).
		Where("updated_at > ?", thirtyDaysAgo).
		Count(&activeUsers).Error; err != nil {
		return nil, err
	}

	// New users in the last 30 days
	if err := s.DB.Model(&models.User{}).
		Where("created_at > ?", thirtyDaysAgo).
		Count(&newUsersLast30Days).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_users":            totalUsers,
		"active_users":           activeUsers,
		"new_users_last_30_days": newUsersLast30Days,
	}, nil
}

// encrypt encrypts a string using AES-GCM
func (s *AdminService) encrypt(text string) (string, error) {
	// Create a new cipher block from the key
	block, err := aes.NewCipher(s.SecretKey)
	if err != nil {
		return "", err
	}

	// Create a new GCM cipher
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce (used only once)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(text), nil)

	// Return the encrypted data as a base64 string
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string using AES-GCM
func (s *AdminService) decrypt(encryptedText string) (string, error) {
	// Decode the base64 string
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	// Create a new cipher block from the key
	block, err := aes.NewCipher(s.SecretKey)
	if err != nil {
		return "", err
	}

	// Create a new GCM cipher
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract the nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
