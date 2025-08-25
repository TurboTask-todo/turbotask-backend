package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/repository"
	"macwrite-auth-api/pkg/redis"

	"github.com/google/uuid"
)

// ReleaseVersionService handles release version business logic
type ReleaseVersionService struct {
	releaseRepo *repository.ReleaseVersionRepository
	projectRepo *repository.ProjectRepository
	redisClient redis.Client
}

// NewReleaseVersionService creates a new release version service
func NewReleaseVersionService(
	releaseRepo *repository.ReleaseVersionRepository,
	projectRepo *repository.ProjectRepository,
	redisClient redis.Client,
) *ReleaseVersionService {
	return &ReleaseVersionService{
		releaseRepo: releaseRepo,
		projectRepo: projectRepo,
		redisClient: redisClient,
	}
}

// Cache key constants for release versions
const (
	releaseCachePrefix     = "release:"
	projectReleasePrefix   = "project_releases:"
	releaseStatsPrefix     = "release_stats:"
	upcomingReleasesPrefix = "upcoming_releases:"
	overdueReleasesPrefix  = "overdue_releases:"
	releaseCacheTTL        = 25 * time.Minute
	releaseStatsCacheTTL   = 15 * time.Minute
)

// CreateReleaseVersion creates a new release version
func (s *ReleaseVersionService) CreateReleaseVersion(ctx context.Context, userID uuid.UUID, version *models.ReleaseVersion) (*models.ReleaseVersion, error) {
	// Verify project ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, version.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Validate release version
	if err := s.validateReleaseVersion(version); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if version number already exists for this project
	existing, err := s.releaseRepo.GetByVersionNumber(ctx, version.ProjectID, version.VersionNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing version: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("version number already exists for this project")
	}

	// Create release version
	if err := s.releaseRepo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create release version: %w", err)
	}

	// Invalidate caches
	s.invalidateReleaseCaches(version.ProjectID, userID)

	return version, nil
}

// GetReleaseVersion retrieves a release version by ID
func (s *ReleaseVersionService) GetReleaseVersion(ctx context.Context, releaseID, userID uuid.UUID, includeCounts bool) (*models.ReleaseVersion, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:counts_%t", releaseCachePrefix, releaseID.String(), includeCounts)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var version models.ReleaseVersion
		if err := json.Unmarshal(cached, &version); err == nil {
			// Verify ownership through project
			exists, err := s.projectRepo.ValidateProjectOwnership(ctx, version.ProjectID, userID)
			if err != nil || !exists {
				return nil, fmt.Errorf("release version not found")
			}
			return &version, nil
		}
	}

	// Get from database
	version, err := s.releaseRepo.GetByID(ctx, releaseID, includeCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to get release version: %w", err)
	}
	if version == nil {
		return nil, fmt.Errorf("release version not found")
	}

	// Verify ownership through project
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, version.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("release version not found")
	}

	// Cache the result
	if versionData, err := json.Marshal(version); err == nil {
		s.redisClient.Set(ctx, cacheKey, versionData, releaseCacheTTL)
	}

	return version, nil
}

// GetProjectReleaseVersions retrieves all release versions for a project
func (s *ReleaseVersionService) GetProjectReleaseVersions(ctx context.Context, projectID, userID uuid.UUID, includeCounts bool) ([]*models.ReleaseVersion, error) {
	// Verify project ownership
	// exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to validate project ownership: %w", err)
	// }
	// if !exists {
	// 	return nil, fmt.Errorf("project not found")
	// }

	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:counts_%t", projectReleasePrefix, projectID.String(), includeCounts)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var versions []*models.ReleaseVersion
		if err := json.Unmarshal(cached, &versions); err == nil {
			return versions, nil
		}
	}

	// Get from database
	versions, err := s.releaseRepo.GetByProjectID(ctx, projectID, includeCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to get project release versions: %w", err)
	}

	// Cache the result
	if versionsData, err := json.Marshal(versions); err == nil {
		s.redisClient.Set(ctx, cacheKey, versionsData, releaseCacheTTL)
	}

	return versions, nil
}

// UpdateReleaseVersion updates a release version
func (s *ReleaseVersionService) UpdateReleaseVersion(ctx context.Context, releaseID, userID uuid.UUID, updates map[string]interface{}) (*models.ReleaseVersion, error) {
	// Verify ownership
	exists, err := s.releaseRepo.ValidateReleaseOwnership(ctx, releaseID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("release version not found")
	}

	// Get current version to get project ID for cache invalidation
	currentVersion, err := s.releaseRepo.GetByID(ctx, releaseID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// Validate updates
	if err := s.validateReleaseVersionUpdates(updates); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Update in database
	if err := s.releaseRepo.Update(ctx, releaseID, updates); err != nil {
		return nil, fmt.Errorf("failed to update release version: %w", err)
	}

	// Invalidate caches
	if currentVersion != nil {
		s.invalidateReleaseCaches(currentVersion.ProjectID, userID)
	}

	// Return updated version
	return s.GetReleaseVersion(ctx, releaseID, userID, true)
}

// DeleteReleaseVersion deletes a release version
func (s *ReleaseVersionService) DeleteReleaseVersion(ctx context.Context, releaseID, userID uuid.UUID) error {
	// Verify ownership
	exists, err := s.releaseRepo.ValidateReleaseOwnership(ctx, releaseID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("release version not found")
	}

	// Get current version to get project ID for cache invalidation
	currentVersion, err := s.releaseRepo.GetByID(ctx, releaseID, false)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Delete in database
	if err := s.releaseRepo.Delete(ctx, releaseID); err != nil {
		return fmt.Errorf("failed to delete release version: %w", err)
	}

	// Invalidate caches
	if currentVersion != nil {
		s.invalidateReleaseCaches(currentVersion.ProjectID, userID)
	}

	return nil
}

// MarkAsReleased marks a release version as released
func (s *ReleaseVersionService) MarkAsReleased(ctx context.Context, releaseID, userID uuid.UUID, releaseNotes string) error {
	// Verify ownership
	exists, err := s.releaseRepo.ValidateReleaseOwnership(ctx, releaseID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("release version not found")
	}

	// Get current version to get project ID for cache invalidation
	currentVersion, err := s.releaseRepo.GetByID(ctx, releaseID, false)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Mark as released
	if err := s.releaseRepo.MarkAsReleased(ctx, releaseID, releaseNotes); err != nil {
		return fmt.Errorf("failed to mark as released: %w", err)
	}

	// Invalidate caches
	if currentVersion != nil {
		s.invalidateReleaseCaches(currentVersion.ProjectID, userID)
	}

	return nil
}

// GetUpcomingReleases retrieves upcoming releases
func (s *ReleaseVersionService) GetUpcomingReleases(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, daysAhead int) ([]*models.ReleaseVersion, error) {
	// If projectID specified, verify ownership
	if projectID != nil {
		exists, err := s.projectRepo.ValidateProjectOwnership(ctx, *projectID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate project ownership: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("project not found")
		}
	}

	// Check cache first
	projectIDStr := "all"
	if projectID != nil {
		projectIDStr = projectID.String()
	}
	cacheKey := fmt.Sprintf("%s%s:project_%s:days_%d", upcomingReleasesPrefix, userID.String(), projectIDStr, daysAhead)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var releases []*models.ReleaseVersion
		if err := json.Unmarshal(cached, &releases); err == nil {
			return releases, nil
		}
	}

	// Get from database
	releases, err := s.releaseRepo.GetUpcomingReleases(ctx, projectID, daysAhead)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming releases: %w", err)
	}

	// Filter by user ownership if projectID is not specified
	if projectID == nil {
		var userReleases []*models.ReleaseVersion
		for _, release := range releases {
			exists, err := s.projectRepo.ValidateProjectOwnership(ctx, release.ProjectID, userID)
			if err == nil && exists {
				userReleases = append(userReleases, release)
			}
		}
		releases = userReleases
	}

	// Cache the result
	if releasesData, err := json.Marshal(releases); err == nil {
		s.redisClient.Set(ctx, cacheKey, releasesData, releaseStatsCacheTTL)
	}

	return releases, nil
}

// GetOverdueReleases retrieves overdue releases
func (s *ReleaseVersionService) GetOverdueReleases(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID) ([]*models.ReleaseVersion, error) {
	// If projectID specified, verify ownership
	if projectID != nil {
		exists, err := s.projectRepo.ValidateProjectOwnership(ctx, *projectID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate project ownership: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("project not found")
		}
	}

	// Check cache first
	projectIDStr := "all"
	if projectID != nil {
		projectIDStr = projectID.String()
	}
	cacheKey := fmt.Sprintf("%s%s:project_%s", overdueReleasesPrefix, userID.String(), projectIDStr)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var releases []*models.ReleaseVersion
		if err := json.Unmarshal(cached, &releases); err == nil {
			return releases, nil
		}
	}

	// Get from database
	releases, err := s.releaseRepo.GetOverdueReleases(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue releases: %w", err)
	}

	// Filter by user ownership if projectID is not specified
	if projectID == nil {
		var userReleases []*models.ReleaseVersion
		for _, release := range releases {
			exists, err := s.projectRepo.ValidateProjectOwnership(ctx, release.ProjectID, userID)
			if err == nil && exists {
				userReleases = append(userReleases, release)
			}
		}
		releases = userReleases
	}

	// Cache the result
	if releasesData, err := json.Marshal(releases); err == nil {
		s.redisClient.Set(ctx, cacheKey, releasesData, releaseStatsCacheTTL)
	}

	return releases, nil
}

// GetReleaseStats retrieves comprehensive statistics for a release version
func (s *ReleaseVersionService) GetReleaseStats(ctx context.Context, releaseID, userID uuid.UUID) (map[string]interface{}, error) {
	// Verify ownership
	exists, err := s.releaseRepo.ValidateReleaseOwnership(ctx, releaseID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("release version not found")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s%s", releaseStatsPrefix, releaseID.String())
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var stats map[string]interface{}
		if err := json.Unmarshal(cached, &stats); err == nil {
			return stats, nil
		}
	}

	// Get from database
	stats, err := s.releaseRepo.GetReleaseStats(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release stats: %w", err)
	}

	// Cache the result
	if statsData, err := json.Marshal(stats); err == nil {
		s.redisClient.Set(ctx, cacheKey, statsData, releaseStatsCacheTTL)
	}

	return stats, nil
}

// GetReleaseTodosProgress retrieves todos with progress for a release
func (s *ReleaseVersionService) GetReleaseTodosProgress(ctx context.Context, releaseID, userID uuid.UUID) ([]*models.Todo, error) {
	// Verify ownership
	exists, err := s.releaseRepo.ValidateReleaseOwnership(ctx, releaseID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("release version not found")
	}

	// Get todos progress
	todos, err := s.releaseRepo.GetReleaseTodosProgress(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release todos progress: %w", err)
	}

	return todos, nil
}

// GetReleaseByVersionNumber retrieves a release version by version number
func (s *ReleaseVersionService) GetReleaseByVersionNumber(ctx context.Context, projectID, userID uuid.UUID, versionNumber string) (*models.ReleaseVersion, error) {
	// Verify project ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Get release version
	version, err := s.releaseRepo.GetByVersionNumber(ctx, projectID, versionNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get release version: %w", err)
	}

	return version, nil
}

// Validation methods

func (s *ReleaseVersionService) validateReleaseVersion(version *models.ReleaseVersion) error {
	if version.VersionName == "" {
		return fmt.Errorf("version name is required")
	}
	if len(version.VersionName) > 100 {
		return fmt.Errorf("version name must be less than 100 characters")
	}
	if version.VersionNumber == "" {
		return fmt.Errorf("version number is required")
	}
	if len(version.VersionNumber) > 20 {
		return fmt.Errorf("version number must be less than 20 characters")
	}
	if version.ProjectID == uuid.Nil {
		return fmt.Errorf("project ID is required")
	}
	if version.TargetDate != nil && version.TargetDate.Before(time.Now().Truncate(24*time.Hour)) {
		return fmt.Errorf("target date cannot be in the past")
	}
	return nil
}

func (s *ReleaseVersionService) validateReleaseVersionUpdates(updates map[string]interface{}) error {
	if versionName, exists := updates["version_name"]; exists {
		if str, ok := versionName.(string); ok {
			if str == "" {
				return fmt.Errorf("version name cannot be empty")
			}
			if len(str) > 100 {
				return fmt.Errorf("version name must be less than 100 characters")
			}
		}
	}
	if versionNumber, exists := updates["version_number"]; exists {
		if str, ok := versionNumber.(string); ok {
			if str == "" {
				return fmt.Errorf("version number cannot be empty")
			}
			if len(str) > 20 {
				return fmt.Errorf("version number must be less than 20 characters")
			}
		}
	}
	if targetDate, exists := updates["target_date"]; exists {
		if date, ok := targetDate.(*time.Time); ok && date != nil {
			if date.Before(time.Now().Truncate(24 * time.Hour)) {
				return fmt.Errorf("target date cannot be in the past")
			}
		}
	}
	return nil
}

// Cache invalidation methods

func (s *ReleaseVersionService) invalidateReleaseCaches(projectID, userID uuid.UUID) {
	ctx := context.Background()

	// Invalidate project releases cache
	projectPattern := fmt.Sprintf("%s%s:*", projectReleasePrefix, projectID.String())
	s.redisClient.DeletePattern(ctx, projectPattern)

	// Invalidate release stats cache for this project's releases
	statsPattern := fmt.Sprintf("%s*", releaseStatsPrefix)
	s.redisClient.DeletePattern(ctx, statsPattern)

	// Invalidate upcoming releases cache
	upcomingPattern := fmt.Sprintf("%s%s:*", upcomingReleasesPrefix, userID.String())
	s.redisClient.DeletePattern(ctx, upcomingPattern)

	// Invalidate overdue releases cache
	overduePattern := fmt.Sprintf("%s%s:*", overdueReleasesPrefix, userID.String())
	s.redisClient.DeletePattern(ctx, overduePattern)
}

// Helper methods

// GetReleaseProgress calculates and returns release progress
func (s *ReleaseVersionService) GetReleaseProgress(ctx context.Context, releaseID, userID uuid.UUID) (float64, error) {
	stats, err := s.GetReleaseStats(ctx, releaseID, userID)
	if err != nil {
		return 0, err
	}

	if completionPercentage, exists := stats["completion_percentage"]; exists {
		if percentage, ok := completionPercentage.(float64); ok {
			return percentage, nil
		}
	}

	return 0, nil
}
