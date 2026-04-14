package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/server/chatui"
)

// chatUIHandler returns a Gin handler that serves the Chat UI SPA from the
// embedded dist/ filesystem.
//
// Route prefix: /chat  (registered in router.go)
//
// Serving rules:
//  1. Strip the /chat path prefix from the request URL.
//  2. Attempt to read the resulting path from the embedded FS.
//  3. On a miss, serve index.html — enabling SvelteKit client-side routing.
//  4. If the SPA was not built (IsBuilt() == false), return 503 with
//     instructions so developers get a clear signal rather than a 404 maze.
func (s *Server) chatUIHandler() gin.HandlerFunc {
	if !chatui.IsBuilt() {
		slog.Warn("chat UI SPA not embedded — GET /chat will return 503. Run: make build-chat-spa")
		return func(c *gin.Context) {
			c.String(http.StatusServiceUnavailable,
				"Chat UI not built.\n\nRun: make build-chat-spa\n\nThis compiles the SvelteKit SPA and copies it into server/chatui/dist/.\n")
		}
	}

	uiFS, err := chatui.FS()
	if err != nil {
		slog.Error("chat UI embed FS unavailable", "error", err)
		return func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "chat UI FS error: %v", err)
		}
	}

	return func(c *gin.Context) {
		// Strip the /chat prefix to get the relative file path.
		urlPath := strings.TrimPrefix(c.Request.URL.Path, "/chat")
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

		// HTML files must not be cached so browsers always fetch the latest
		// index.html (which references content-hashed JS/CSS). Immutable assets
		// can be cached indefinitely since their filenames change on rebuild.
		if strings.HasSuffix(urlPath, ".html") || urlPath == "index.html" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		c.Data(http.StatusOK, contentType, data)
	}
}
