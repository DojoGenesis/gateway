package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RegisterDIPResources registers DIP resources on the MCP server.
// Resources provide read-only views into DIP data via the MCP resource protocol.
func RegisterDIPResources(server Server, dipBaseURL string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	// dojo://dip/lenses — List all available DIP lenses.
	if err := server.RegisterResource(ResourceRegistration{
		URI:         "dojo://dip/lenses",
		Name:        "DIP Lenses",
		Description: "All DIP evaluation lenses with their dimensions and license types",
		MimeType:    "application/json",
		Handler: func(ctx context.Context, _ string) (interface{}, error) {
			return dipGet(ctx, client, dipBaseURL+"/api/v1/lenses")
		},
	}); err != nil {
		return fmt.Errorf("register dojo://dip/lenses: %w", err)
	}

	// dojo://dip/components — List all component nodes in the DIP graph.
	if err := server.RegisterResource(ResourceRegistration{
		URI:         "dojo://dip/components",
		Name:        "DIP Components",
		Description: "All component nodes tracked in the DIP semantic design graph",
		MimeType:    "application/json",
		Handler: func(ctx context.Context, _ string) (interface{}, error) {
			return dipGet(ctx, client, dipBaseURL+"/api/v1/nodes")
		},
	}); err != nil {
		return fmt.Errorf("register dojo://dip/components: %w", err)
	}

	return nil
}
