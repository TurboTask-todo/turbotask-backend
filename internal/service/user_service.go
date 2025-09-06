package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/repository"

	"github.com/google/uuid"
)

// UserService handles user-related business logic
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// GetByEmail gets a user by email
func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// GetByID gets a user by ID
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// CreateUserFromEmail creates a new user with default data from email
func (s *UserService) CreateUserFromEmail(ctx context.Context, email string) (*models.User, error) {
	// Generate username from email (before @ symbol)
	username := strings.Split(email, "@")[0]
	firstName := "User"
	lastName := ""

	// Create new user with default data
	user := &models.User{
		ID:            uuid.New(),
		Email:         email,
		Username:      &username,
		EmailVerified: true, // Since they verified via OTP
		FirstName:     &firstName,
		LastName:      &lastName,
		AvatarURL:     nil,
		Timezone:      "UTC",
		Locale:        "en",
		Preferences:   models.JSONB{}, // Empty JSONB object instead of nil
		Status:        models.UserStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create user in database
	err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (s *UserService) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	// For now, we'll assume there's no last_login field in the user table
	// This is just a placeholder method that could be implemented later
	// when the user table is updated with a last_login field

	// If the user table has a last_login field, this would be:
	// updates := map[string]interface{}{
	//     "last_login": time.Now(),
	// }
	// return s.userRepo.Update(ctx, userID, updates)

	// For now, just return nil
	_ = userID
	_ = time.Now()
	return nil
}
