package server

import (
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/workflowui"
)

// workflowBuilderHandler returns a Gin handler that serves the Workflow Builder
// SPA from the embedded dist/ filesystem.
//
// Route prefix: /workflow  (registered in router.go)
//
// Serving rules:
//  1. Strip the /workflow path prefix from the request URL.
//  2. Attempt to read the resulting path from the embedded FS.
//  3. On a miss, serve index.html — enabling SvelteKit client-side routing.
//  4. If the SPA was not built (IsBuilt() == false), return 503 with
//     instructions so developers get a clear signal rather than a 404 maze.
func (s *Server) workflowBuilderHandler() gin.HandlerFunc {
	if !workflowui.IsBuilt() {
		slog.Warn("workflow builder SPA not embedded — GET /workflow will return 503. Run: make build-spa")
		return func(c *gin.Context) {
			c.String(http.StatusServiceUnavailable,
				"Workflow Builder not built.\n\nRun: make build-spa\n\nThis compiles the SvelteKit SPA and copies it into server/workflowui/dist/.\n")
		}
	}

	uiFS, err := workflowui.FS()
	if err != nil {
		slog.Error("workflow builder embed FS unavailable", "error", err)
		return func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "workflow builder FS error: %v", err)
		}
	}

	return func(c *gin.Context) {
		// Strip the /workflow prefix to get the relative file path.
		urlPath := strings.TrimPrefix(c.Request.URL.Path, "/workflow")
		urlPath = strings.TrimPrefix(urlPath, "/")
		if urlPath == "" {
			urlPath = "index.html"
		}

		data, contentType := readFile(uiFS, urlPath)
		if data == nil {
			// SPA fallback: unknown paths are handled by client-side routing.
			data, contentType = readFile(uiFS, "index.html")
		}
		if data == nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.Data(http.StatusOK, contentType, data)
	}
}

// readFile reads a file from the embedded FS and detects its content type.
// Returns (nil, "") if the file does not exist.
func readFile(uiFS fs.FS, path string) ([]byte, string) {
	data, err := fs.ReadFile(uiFS, path)
	if err != nil {
		return nil, ""
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return data, contentType
}
