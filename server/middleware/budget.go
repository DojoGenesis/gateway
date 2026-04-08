package middleware

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/DojoGenesis/gateway/server/services"
	"github.com/gin-gonic/gin"
)

// BudgetMiddleware creates a middleware that enforces budget limits
func BudgetMiddleware(tracker *services.BudgetTracker, costTracker *services.CostTracker) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Get user ID from context (set by auth middleware or extracted from request)
		userID := c.GetString("user_id")

		// Guest users (no userID) have no budget limits
		if userID == "" {
			c.Next()
			return
		}

		// Check budget limits before processing
		budgetTypes := []string{}

		// Check monthly budget
		if tracker.IsMonthlyLimitExceeded(userID) {
			c.Header("X-Budget-Exceeded", "true")
			budgetTypes = append(budgetTypes, "monthly")
		}

		// Check session budget
		if tracker.IsSessionLimitExceeded(userID) {
			c.Header("X-Budget-Warning", "true")
			if !tracker.IsMonthlyLimitExceeded(userID) {
				// Only set type to session if monthly not exceeded
				budgetTypes = append(budgetTypes, "session")
			}
		}

		// Set budget type (prefer monthly over session)
		if len(budgetTypes) > 0 {
			c.Header("X-Budget-Type", budgetTypes[0])
		}

		// Get remaining budget
		remaining, err := tracker.GetRemaining(userID)
		if err == nil {
			c.Header("X-Budget-Remaining", strconv.Itoa(remaining))
		}

		// Get limits
		queryLimit, sessionLimit, monthlyLimit := tracker.GetLimits()
		c.Header("X-Budget-Query-Limit", strconv.Itoa(queryLimit))
		c.Header("X-Budget-Session-Limit", strconv.Itoa(sessionLimit))
		c.Header("X-Budget-Monthly-Limit", strconv.Itoa(monthlyLimit))

		// Get current usage
		sessionUsage := tracker.GetSessionUsage(userID)
		monthlyUsage := tracker.GetMonthlyUsage(userID)
		c.Header("X-Budget-Session-Usage", strconv.Itoa(sessionUsage))
		c.Header("X-Budget-Monthly-Usage", strconv.Itoa(monthlyUsage))

		// Calculate percentage used for monthly budget
		percentUsed := 0.0
		if monthlyLimit > 0 {
			percentUsed = (float64(monthlyUsage) / float64(monthlyLimit)) * 100.0
		}
		c.Header("X-Budget-Percent-Used", fmt.Sprintf("%.2f", percentUsed))

		// Check for warning threshold (90%)
		if percentUsed >= 90.0 && percentUsed < 100.0 {
			c.Header("X-Budget-Warning", "true")
			c.Header("X-Budget-Warning-Message", "Approaching budget limit (90% used)")
		}

		// Process request
		c.Next()

		// Track usage after processing
		tokenUsage := c.GetInt("token_usage")
		if tokenUsage > 0 {
			// Check if this single query exceeds the query limit
			if tracker.IsQueryLimitExceeded(userID, tokenUsage) {
				c.Header("X-Budget-Query-Exceeded", "true")
			}

			// Track the usage
			if err := tracker.TrackUsage(userID, tokenUsage); err != nil {
				slog.Warn("budget tracking error", "user_id", userID, "error", err)
				c.Header("X-Budget-Error", "tracking_failed")
			}

			// Calculate cost if model is provided
			model := c.GetString("model")
			if model != "" && costTracker != nil {
				cost := costTracker.GetCost(model, tokenUsage)
				c.Header("X-Budget-Cost", fmt.Sprintf("%.6f", cost))

				// Get detailed budget info
				budgetInfo := costTracker.GetBudgetInfo(model, 3.0, monthlyUsage+tokenUsage)
				if budgetInfo != nil {
					c.Header("X-Budget-Remaining-Dollars", fmt.Sprintf("%.2f", budgetInfo.RemainingDollars))
					c.Header("X-Budget-Is-Warning", strconv.FormatBool(budgetInfo.IsWarning))
					c.Header("X-Budget-Is-Exceeded", strconv.FormatBool(budgetInfo.IsExceeded))
				}
			}
		}

		// Add request duration
		duration := time.Since(startTime)
		c.Header("X-Request-Duration", duration.String())
	}
}

// BudgetInfo extracts budget information from request headers
type BudgetInfo struct {
	Exceeded         bool
	Warning          bool
	RemainingTokens  int
	RemainingDollars float64
	PercentUsed      float64
	SessionUsage     int
	MonthlyUsage     int
	QueryLimit       int
	SessionLimit     int
	MonthlyLimit     int
	Type             string
	Message          string
}

// ExtractBudgetInfo extracts budget information from response headers
func ExtractBudgetInfo(c *gin.Context) *BudgetInfo {
	info := &BudgetInfo{}

	info.Exceeded = c.GetHeader("X-Budget-Exceeded") == "true"
	info.Warning = c.GetHeader("X-Budget-Warning") == "true"
	info.Type = c.GetHeader("X-Budget-Type")
	info.Message = c.GetHeader("X-Budget-Warning-Message")

	if remaining := c.GetHeader("X-Budget-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			info.RemainingTokens = val
		}
	}

	if remainingDollars := c.GetHeader("X-Budget-Remaining-Dollars"); remainingDollars != "" {
		if val, err := strconv.ParseFloat(remainingDollars, 64); err == nil {
			info.RemainingDollars = val
		}
	}

	if percentUsed := c.GetHeader("X-Budget-Percent-Used"); percentUsed != "" {
		if val, err := strconv.ParseFloat(percentUsed, 64); err == nil {
			info.PercentUsed = val
		}
	}

	if sessionUsage := c.GetHeader("X-Budget-Session-Usage"); sessionUsage != "" {
		if val, err := strconv.Atoi(sessionUsage); err == nil {
			info.SessionUsage = val
		}
	}

	if monthlyUsage := c.GetHeader("X-Budget-Monthly-Usage"); monthlyUsage != "" {
		if val, err := strconv.Atoi(monthlyUsage); err == nil {
			info.MonthlyUsage = val
		}
	}

	if queryLimit := c.GetHeader("X-Budget-Query-Limit"); queryLimit != "" {
		if val, err := strconv.Atoi(queryLimit); err == nil {
			info.QueryLimit = val
		}
	}

	if sessionLimit := c.GetHeader("X-Budget-Session-Limit"); sessionLimit != "" {
		if val, err := strconv.Atoi(sessionLimit); err == nil {
			info.SessionLimit = val
		}
	}

	if monthlyLimit := c.GetHeader("X-Budget-Monthly-Limit"); monthlyLimit != "" {
		if val, err := strconv.Atoi(monthlyLimit); err == nil {
			info.MonthlyLimit = val
		}
	}

	return info
}
