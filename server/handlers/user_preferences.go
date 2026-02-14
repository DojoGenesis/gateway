package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type UserPreferences struct {
	ID                      string      `json:"id"`
	NotifyMemoryCompression bool        `json:"notify_memory_compression"`
	NotifySeedsExtracted    bool        `json:"notify_seeds_extracted"`
	NotifyGoalMilestone     bool        `json:"notify_goal_milestone"`
	NotifyProjectDormancy   bool        `json:"notify_project_dormancy"`
	NotifyCommonPatterns    bool        `json:"notify_common_patterns"`
	QuietHours              []QuietHour `json:"quiet_hours,omitempty"`
	NotificationRateLimit   int         `json:"notification_rate_limit"`
	LayoutAutoSwitch        bool        `json:"layout_auto_switch"`
	LayoutAnimation         bool        `json:"layout_animation"`
	UpdatedAt               time.Time   `json:"updated_at"`
}

type QuietHour struct {
	Day       int `json:"day"`        // 0 (Sunday) - 6 (Saturday)
	StartHour int `json:"start_hour"` // 0-23
	EndHour   int `json:"end_hour"`   // 0-23
}

// PreferencesHandler handles user preferences HTTP requests.
type PreferencesHandler struct {
	db *sql.DB
}

// NewPreferencesHandler creates a new PreferencesHandler.
func NewPreferencesHandler(db *sql.DB) *PreferencesHandler {
	return &PreferencesHandler{db: db}
}

func (h *PreferencesHandler) GetUserPreferences(c *gin.Context) {
	if h.db == nil {
		respondInternalError(c, "Preferences database not initialized")
		return
	}

	prefs, err := getUserPreferences(h.db)
	if err != nil {
		slog.Error("failed to get preferences", "error", err)
		respondInternalError(c, "Failed to get preferences")
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (h *PreferencesHandler) UpdateUserPreferences(c *gin.Context) {
	if h.db == nil {
		respondInternalError(c, "Preferences database not initialized")
		return
	}

	var input UserPreferences
	if err := c.ShouldBindJSON(&input); err != nil {
		respondBadRequest(c, "Invalid request body")
		return
	}

	if input.NotificationRateLimit < 1 {
		respondBadRequest(c, "notification_rate_limit must be at least 1 minute")
		return
	}

	for _, qh := range input.QuietHours {
		if qh.Day < 0 || qh.Day > 6 {
			respondBadRequest(c, "quiet_hours day must be 0-6")
			return
		}
		if qh.StartHour < 0 || qh.StartHour > 23 || qh.EndHour < 0 || qh.EndHour > 23 {
			respondBadRequest(c, "quiet_hours hours must be 0-23")
			return
		}
	}

	prefs, err := updateUserPreferences(h.db, &input)
	if err != nil {
		slog.Error("failed to update preferences", "error", err)
		respondInternalError(c, "Failed to update preferences")
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func getUserPreferences(db *sql.DB) (*UserPreferences, error) {
	query := `
		SELECT id, notify_memory_compression, notify_seeds_extracted, 
		       notify_goal_milestone, notify_project_dormancy, notify_common_patterns,
		       quiet_hours, notification_rate_limit, layout_auto_switch, layout_animation, updated_at
		FROM user_preferences
		WHERE id = 'default'
	`

	var prefs UserPreferences
	var quietHoursJSON sql.NullString

	err := db.QueryRow(query).Scan(
		&prefs.ID,
		&prefs.NotifyMemoryCompression,
		&prefs.NotifySeedsExtracted,
		&prefs.NotifyGoalMilestone,
		&prefs.NotifyProjectDormancy,
		&prefs.NotifyCommonPatterns,
		&quietHoursJSON,
		&prefs.NotificationRateLimit,
		&prefs.LayoutAutoSwitch,
		&prefs.LayoutAnimation,
		&prefs.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}

	if quietHoursJSON.Valid && quietHoursJSON.String != "" {
		if err := json.Unmarshal([]byte(quietHoursJSON.String), &prefs.QuietHours); err != nil {
			return nil, fmt.Errorf("failed to parse quiet_hours: %w", err)
		}
	}

	return &prefs, nil
}

func updateUserPreferences(db *sql.DB, prefs *UserPreferences) (*UserPreferences, error) {
	var quietHoursJSON interface{}
	if len(prefs.QuietHours) > 0 {
		data, err := json.Marshal(prefs.QuietHours)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal quiet_hours: %w", err)
		}
		quietHoursJSON = string(data)
	} else {
		quietHoursJSON = nil
	}

	now := time.Now()
	query := `
		UPDATE user_preferences
		SET notify_memory_compression = ?,
		    notify_seeds_extracted = ?,
		    notify_goal_milestone = ?,
		    notify_project_dormancy = ?,
		    notify_common_patterns = ?,
		    quiet_hours = ?,
		    notification_rate_limit = ?,
		    layout_auto_switch = ?,
		    layout_animation = ?,
		    updated_at = ?
		WHERE id = 'default'
	`

	_, err := db.Exec(query,
		prefs.NotifyMemoryCompression,
		prefs.NotifySeedsExtracted,
		prefs.NotifyGoalMilestone,
		prefs.NotifyProjectDormancy,
		prefs.NotifyCommonPatterns,
		quietHoursJSON,
		prefs.NotificationRateLimit,
		prefs.LayoutAutoSwitch,
		prefs.LayoutAnimation,
		now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	return getUserPreferences(db)
}
