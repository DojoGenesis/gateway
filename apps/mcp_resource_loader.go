package apps

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// ResourceContent holds content returned from an MCP resource read.
type ResourceContent struct {
	URI      string
	MimeType string
	Text     *string
	Blob     *string
}

// MCPConnection abstracts the MCP server connection for resource loading.
type MCPConnection interface {
	ReadResource(ctx context.Context, uri string) ([]ResourceContent, error)
}

// MCPResourceLoader loads resources from MCP servers into the ResourceRegistry.
type MCPResourceLoader struct {
	registry *ResourceRegistry
}

// NewMCPResourceLoader creates a new loader backed by the given registry.
func NewMCPResourceLoader(registry *ResourceRegistry) *MCPResourceLoader {
	return &MCPResourceLoader{registry: registry}
}

// LoadResource fetches a resource from an MCP server and registers it.
func (l *MCPResourceLoader) LoadResource(ctx context.Context, conn MCPConnection, resourceURI string) error {
	if !strings.HasPrefix(resourceURI, "ui://") {
		return fmt.Errorf("invalid resource URI scheme: %s (expected ui://)", resourceURI)
	}

	contents, err := conn.ReadResource(ctx, resourceURI)
	if err != nil {
		return fmt.Errorf("failed to load resource %s: %w", resourceURI, err)
	}

	if len(contents) == 0 {
		return fmt.Errorf("no content returned for resource %s", resourceURI)
	}

	rc := contents[0]

	var content []byte
	if rc.Text != nil {
		content = []byte(*rc.Text)
	} else if rc.Blob != nil {
		decoded, err := base64.StdEncoding.DecodeString(*rc.Blob)
		if err != nil {
			return fmt.Errorf("failed to decode blob for resource %s: %w", resourceURI, err)
		}
		content = decoded
	} else {
		return fmt.Errorf("resource %s has no text or blob content", resourceURI)
	}

	mimeType := rc.MimeType
	if mimeType == "" {
		mimeType = "text/html"
	}

	cacheKey := fmt.Sprintf("%s-%d", resourceURI, len(content))

	meta := &ResourceMeta{
		URI:      resourceURI,
		MimeType: mimeType,
		Content:  content,
		CacheKey: cacheKey,
	}

	return l.registry.Register(meta)
}
