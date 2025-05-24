package services

import (
	"gorm.io/gorm"
)

type ChatService struct {
	DB *gorm.DB
}

func NewChatService(db *gorm.DB) *ChatService {
	return &ChatService{DB: db}
}
