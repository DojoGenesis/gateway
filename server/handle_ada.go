package server

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// ─── ADA Validation Types ────────────────────────────────────────────────────

// adaValidateRequest is the JSON body for POST /api/ada/validate.
type adaValidateRequest struct {
	Identity    *adaIdentity    `json:"identity"`
	Disposition *adaDisposition `json:"disposition"`
}

type adaIdentity struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	Version string `json:"version"`
}

type adaDisposition struct {
	Pacing     string `json:"pacing"`
	Depth      string `json:"depth"`
	Tone       string `json:"tone"`
	Initiative string `json:"initiative"`
}

type adaValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type adaValidateResponse struct {
	Valid  bool                 `json:"valid"`
	Errors []adaValidationError `json:"errors,omitempty"`
}

// ─── Allowed Enum Values ─────────────────────────────────────────────────────

var (
	validPacing     = map[string]bool{"deliberate": true, "measured": true, "responsive": true, "rapid": true}
	validDepth      = map[string]bool{"surface": true, "functional": true, "thorough": true, "exhaustive": true}
	validTone       = map[string]bool{"formal": true, "professional": true, "conversational": true, "casual": true}
	validInitiative = map[string]bool{"reactive": true, "responsive": true, "proactive": true, "autonomous": true}

	// Semver pattern: major.minor.patch with optional pre-release
	semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`)
)

// ─── Handler ─────────────────────────────────────────────────────────────────

// handleADAValidate validates an agent identity and disposition against
// structural rules. This is a lightweight check the frontend uses before
// agent creation — NOT a full JSON Schema validator.
//
// POST /api/ada/validate
func (s *Server) handleADAValidate(c *gin.Context) {
	var req adaValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid JSON body: "+err.Error())
		return
	}

	var errs []adaValidationError

	// ── Validate identity ──────────────────────────────────────────────
	if req.Identity == nil {
		errs = append(errs, adaValidationError{
			Field:   "identity",
			Message: "identity object is required",
		})
	} else {
		if req.Identity.Name == "" {
			errs = append(errs, adaValidationError{
				Field:   "identity.name",
				Message: "name is required and must be non-empty",
			})
		}
		if req.Identity.Purpose == "" {
			errs = append(errs, adaValidationError{
				Field:   "identity.purpose",
				Message: "purpose is required and must be non-empty",
			})
		}
		if req.Identity.Version == "" {
			errs = append(errs, adaValidationError{
				Field:   "identity.version",
				Message: "version is required and must be non-empty",
			})
		} else if !semverPattern.MatchString(req.Identity.Version) {
			errs = append(errs, adaValidationError{
				Field:   "identity.version",
				Message: "version must follow semantic versioning (e.g., 1.0.0)",
			})
		}
	}

	// ── Validate disposition (if provided) ─────────────────────────────
	if req.Disposition != nil {
		if req.Disposition.Pacing != "" && !validPacing[req.Disposition.Pacing] {
			errs = append(errs, adaValidationError{
				Field:   "disposition.pacing",
				Message: "pacing must be one of: deliberate, measured, responsive, rapid",
			})
		}
		if req.Disposition.Depth != "" && !validDepth[req.Disposition.Depth] {
			errs = append(errs, adaValidationError{
				Field:   "disposition.depth",
				Message: "depth must be one of: surface, functional, thorough, exhaustive",
			})
		}
		if req.Disposition.Tone != "" && !validTone[req.Disposition.Tone] {
			errs = append(errs, adaValidationError{
				Field:   "disposition.tone",
				Message: "tone must be one of: formal, professional, conversational, casual",
			})
		}
		if req.Disposition.Initiative != "" && !validInitiative[req.Disposition.Initiative] {
			errs = append(errs, adaValidationError{
				Field:   "disposition.initiative",
				Message: "initiative must be one of: reactive, responsive, proactive, autonomous",
			})
		}
	}

	if len(errs) > 0 {
		c.JSON(http.StatusOK, adaValidateResponse{
			Valid:  false,
			Errors: errs,
		})
		return
	}

	c.JSON(http.StatusOK, adaValidateResponse{
		Valid: true,
	})
}
