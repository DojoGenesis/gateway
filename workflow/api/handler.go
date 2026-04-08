// Package api implements HTTP handlers for the Workflow Gateway API.
// It provides CRUD endpoints for workflow definitions and canvas state,
// backed by content-addressable storage (CAS).
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/workflow"
)

// WorkflowHandler provides HTTP handlers for workflow CRUD operations.
type WorkflowHandler struct {
	cas cas.Store
}

// NewWorkflowHandler returns a WorkflowHandler backed by the given CAS store.
func NewWorkflowHandler(casStore cas.Store) *WorkflowHandler {
	return &WorkflowHandler{cas: casStore}
}

// RegisterRoutes registers all workflow API routes on the given mux.
func (h *WorkflowHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/workflows", h.handleWorkflows)
	mux.HandleFunc("/api/workflows/", h.handleWorkflowByName)
	mux.HandleFunc("/api/skills", h.handleSkills)
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// POST /api/workflows  — Create workflow
// GET  /api/workflows  — List workflows
// ---------------------------------------------------------------------------

func (h *WorkflowHandler) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createWorkflow(w, r)
	case http.MethodGet:
		h.listWorkflows(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// createWorkflow handles POST /api/workflows.
// Body: WorkflowDefinition JSON.
// Returns: {"ref": "sha256:...", "name": "...", "version": "..."} on 201.
func (h *WorkflowHandler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var def workflow.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate (includes cycle detection and empty-name check).
	if err := workflow.Validate(&def); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Serialize to stable JSON for CAS storage.
	data, err := workflow.Marshal(&def)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Store in CAS.
	ref, err := h.cas.Put(ctx, data, cas.ContentMeta{
		Type:      cas.ContentConfig,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"kind":    "workflow",
			"name":    def.Name,
			"version": def.Version,
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store error: "+err.Error())
		return
	}

	// Tag as workflow/{name}:{version}.
	tagName := "workflow/" + def.Name
	version := def.Version
	if version == "" {
		version = "latest"
	}
	if err := h.cas.Tag(ctx, tagName, version, ref); err != nil {
		writeError(w, http.StatusInternalServerError, "tag error: "+err.Error())
		return
	}

	// Always keep "latest" pointing to the most recently stored version.
	if version != "latest" {
		if err := h.cas.Tag(ctx, tagName, "latest", ref); err != nil {
			writeError(w, http.StatusInternalServerError, "tag latest error: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"ref":     "sha256:" + string(ref),
		"name":    def.Name,
		"version": version,
	})
}

// listWorkflows handles GET /api/workflows.
// Returns: {"workflows": [{"name": "...", "version": "...", "ref": "..."}]}
func (h *WorkflowHandler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entries, err := h.cas.List(ctx, "workflow/")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	type workflowEntry struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Ref     string `json:"ref"`
	}

	// Deduplicate: for each workflow name, prefer the explicit semver tag over the
	// "latest" alias we maintain internally. Skip "canvas" tags entirely.
	type dedupKey = string // plain workflow name
	best := make(map[dedupKey]workflowEntry)

	for _, e := range entries {
		if e.Version == "canvas" || e.Version == "latest" {
			continue
		}
		name := strings.TrimPrefix(e.Name, "workflow/")
		best[name] = workflowEntry{
			Name:    name,
			Version: e.Version,
			Ref:     "sha256:" + string(e.Ref),
		}
	}

	// For workflows that were stored without a version (only have a "latest" tag),
	// fall back to including the "latest" entry.
	for _, e := range entries {
		if e.Version != "latest" {
			continue
		}
		name := strings.TrimPrefix(e.Name, "workflow/")
		if _, exists := best[name]; !exists {
			best[name] = workflowEntry{
				Name:    name,
				Version: e.Version,
				Ref:     "sha256:" + string(e.Ref),
			}
		}
	}

	items := make([]workflowEntry, 0, len(best))
	for _, entry := range best {
		items = append(items, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{"workflows": items})
}

// ---------------------------------------------------------------------------
// /api/workflows/{name}
// /api/workflows/{name}/canvas
// /api/workflows/{name}/validate
// ---------------------------------------------------------------------------

func (h *WorkflowHandler) handleWorkflowByName(w http.ResponseWriter, r *http.Request) {
	// Strip leading "/api/workflows/" and trailing slash.
	rest := strings.TrimPrefix(r.URL.Path, "/api/workflows/")
	rest = strings.TrimSuffix(rest, "/")

	switch {
	case strings.HasSuffix(rest, "/canvas"):
		name := strings.TrimSuffix(rest, "/canvas")
		if name == "" {
			writeError(w, http.StatusBadRequest, "missing workflow name")
			return
		}
		switch r.Method {
		case http.MethodPut:
			h.saveCanvas(w, r, name)
		case http.MethodGet:
			h.getCanvas(w, r, name)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}

	case strings.HasSuffix(rest, "/validate"):
		name := strings.TrimSuffix(rest, "/validate")
		if name == "" {
			writeError(w, http.StatusBadRequest, "missing workflow name")
			return
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.validateWorkflow(w, r, name)

	default:
		// Plain /api/workflows/{name}
		name := rest
		if name == "" {
			writeError(w, http.StatusBadRequest, "missing workflow name")
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.getWorkflow(w, r, name)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// getWorkflow handles GET /api/workflows/{name}[?version=X].
// Returns the full WorkflowDefinition JSON.
func (h *WorkflowHandler) getWorkflow(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	version := r.URL.Query().Get("version")
	if version == "" {
		version = "latest"
	}

	ref, err := h.cas.Resolve(ctx, "workflow/"+name, version)
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workflow not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, "resolve error: "+err.Error())
		return
	}

	data, _, err := h.cas.Get(ctx, ref)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get error: "+err.Error())
		return
	}

	def, err := workflow.Unmarshal(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unmarshal error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, def)
}

// saveCanvas handles PUT /api/workflows/{name}/canvas.
// Body: CanvasState JSON.
// Returns: {"ref": "sha256:..."}
func (h *WorkflowHandler) saveCanvas(w http.ResponseWriter, r *http.Request, name string) {
	var state workflow.CanvasState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	data, err := workflow.MarshalCanvas(&state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal error: "+err.Error())
		return
	}

	ctx := r.Context()

	ref, err := h.cas.Put(ctx, data, cas.ContentMeta{
		Type:      cas.ContentConfig,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"kind": "canvas",
			"name": name,
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store error: "+err.Error())
		return
	}

	// Tag as workflow/{name}:canvas.
	if err := h.cas.Tag(ctx, "workflow/"+name, "canvas", ref); err != nil {
		writeError(w, http.StatusInternalServerError, "tag error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"ref": "sha256:" + string(ref),
	})
}

// getCanvas handles GET /api/workflows/{name}/canvas.
func (h *WorkflowHandler) getCanvas(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	ref, err := h.cas.Resolve(ctx, "workflow/"+name, "canvas")
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			writeError(w, http.StatusNotFound, "canvas not found for: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, "resolve error: "+err.Error())
		return
	}

	data, _, err := h.cas.Get(ctx, ref)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get error: "+err.Error())
		return
	}

	state, err := workflow.UnmarshalCanvas(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unmarshal error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, state)
}

// validateWorkflow handles POST /api/workflows/{name}/validate.
// Body: WorkflowDefinition JSON.
// Returns: {"valid": true} or {"valid": false, "error": "..."}.
// The name path parameter is accepted but not used for validation — the body is authoritative.
func (h *WorkflowHandler) validateWorkflow(w http.ResponseWriter, r *http.Request, _ string) {
	var def workflow.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid": false,
			"error": "invalid JSON: " + err.Error(),
		})
		return
	}

	if err := workflow.Validate(&def); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

// ---------------------------------------------------------------------------
// GET /api/skills — List available skills
// ---------------------------------------------------------------------------

// handleSkills handles GET /api/skills.
// Returns skill manifests from CAS. Optional ?q=search filter.
func (h *WorkflowHandler) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	query := strings.ToLower(r.URL.Query().Get("q"))

	entries, err := h.cas.List(ctx, "skill/")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	type skillEntry struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description,omitempty"`
		Ref         string `json:"ref"`
	}

	items := make([]skillEntry, 0)
	seen := make(map[string]bool)

	for _, e := range entries {
		// Only process config tags (e.g. "skill/foo:config" at version "1.0.0").
		if !strings.HasSuffix(e.Name, ":config") {
			continue
		}

		key := e.Name + "@" + e.Version
		if seen[key] {
			continue
		}
		seen[key] = true

		data, _, err := h.cas.Get(ctx, e.Ref)
		if err != nil {
			continue
		}

		var manifest struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		// Apply optional search filter.
		if query != "" {
			if !strings.Contains(strings.ToLower(manifest.Name), query) &&
				!strings.Contains(strings.ToLower(manifest.Description), query) {
				continue
			}
		}

		items = append(items, skillEntry{
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Ref:         "sha256:" + string(e.Ref),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"skills": items})
}
