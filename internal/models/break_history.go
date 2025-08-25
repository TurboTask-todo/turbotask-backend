package models

import (
	"time"

	"github.com/google/uuid"
)

// BreakHistory represents a break session taken during focus mode
type BreakHistory struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TodoID    uuid.UUID  `json:"todo_id" gorm:"type:uuid;not null;index"`
	ProjectID uuid.UUID  `json:"project_id" gorm:"type:uuid;not null;index"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	StartTime time.Time  `json:"start_time" gorm:"not null"`
	EndTime   *time.Time `json:"end_time" gorm:"default:null"`
	Duration  int        `json:"duration" gorm:"not null;default:0"`                           // Duration in seconds
	BreakType string     `json:"break_type" gorm:"type:varchar(50);not null;default:'manual'"` // manual, scheduled, etc.
	Notes     string     `json:"notes" gorm:"type:text"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Todo    *Todo    `json:"todo,omitempty" gorm:"foreignKey:TodoID"`
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName returns the table name for BreakHistory
func (BreakHistory) TableName() string {
	return "break_history"
}

// IsActive returns true if the break is currently active (no end time)
func (bh *BreakHistory) IsActive() bool {
	return bh.EndTime == nil
}

// GetDuration returns the duration of the break in seconds
func (bh *BreakHistory) GetDuration() int {
	if bh.EndTime == nil {
		// Calculate duration from start time to now
		return int(time.Since(bh.StartTime).Seconds())
	}
	return bh.Duration
}

// StartBreakRequest represents the request to start a break
type StartBreakRequest struct {
	TodoID    uuid.UUID `json:"todo_id" binding:"required"`
	ProjectID uuid.UUID `json:"project_id" binding:"required"`
	BreakType string    `json:"break_type,omitempty"`
	Notes     string    `json:"notes,omitempty"`
}

// StopBreakRequest represents the request to stop a break
type StopBreakRequest struct {
	Notes string `json:"notes,omitempty"`
}

// BreakStatsResponse represents break statistics
type BreakStatsResponse struct {
	TotalBreaks        int           `json:"total_breaks"`
	TotalDuration      int           `json:"total_duration"`   // in seconds
	AverageDuration    float64       `json:"average_duration"` // in seconds
	TodayBreaks        int           `json:"today_breaks"`
	TodayDuration      int           `json:"today_duration"` // in seconds
	ThisWeekBreaks     int           `json:"this_week_breaks"`
	ThisWeekDuration   int           `json:"this_week_duration"` // in seconds
	CurrentActiveBreak *BreakHistory `json:"current_active_break,omitempty"`
}
