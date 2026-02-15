package apps

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
)

// mockMCPConnection implements MCPConnection for testing.
type mockMCPConnection struct {
	contents []ResourceContent
	err      error
}

func (m *mockMCPConnection) ReadResource(_ context.Context, _ string) ([]ResourceContent, error) {
	return m.contents, m.err
}

func TestMCPResourceLoader_LoadResource_TextContent(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)

	text := "<html>hello from MCP</html>"
	conn := &mockMCPConnection{
		contents: []ResourceContent{
			{URI: "ui://test/app.html", MimeType: "text/html", Text: &text},
		},
	}

	if err := loader.LoadResource(context.Background(), conn, "ui://test/app.html"); err != nil {
		t.Fatalf("LoadResource failed: %v", err)
	}

	meta, err := reg.Get("ui://test/app.html")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(meta.Content) != text {
		t.Errorf("content = %q, want %q", meta.Content, text)
	}
	if meta.MimeType != "text/html" {
		t.Errorf("mime = %q, want text/html", meta.MimeType)
	}
	if meta.CacheKey == "" {
		t.Error("expected non-empty CacheKey")
	}
}

func TestMCPResourceLoader_LoadResource_BlobContent(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)

	raw := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	encoded := base64.StdEncoding.EncodeToString(raw)
	conn := &mockMCPConnection{
		contents: []ResourceContent{
			{URI: "ui://test/icon.png", MimeType: "image/png", Blob: &encoded},
		},
	}

	if err := loader.LoadResource(context.Background(), conn, "ui://test/icon.png"); err != nil {
		t.Fatalf("LoadResource failed: %v", err)
	}

	meta, err := reg.Get("ui://test/icon.png")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(meta.Content) != len(raw) {
		t.Errorf("content length = %d, want %d", len(meta.Content), len(raw))
	}
	if meta.MimeType != "image/png" {
		t.Errorf("mime = %q, want image/png", meta.MimeType)
	}
}

func TestMCPResourceLoader_LoadResource_InvalidScheme(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)
	conn := &mockMCPConnection{}

	err := loader.LoadResource(context.Background(), conn, "https://example.com/app.html")
	if err == nil {
		t.Fatal("expected error for non-ui:// scheme")
	}
}

func TestMCPResourceLoader_LoadResource_NoContent(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)
	conn := &mockMCPConnection{
		contents: []ResourceContent{},
	}

	err := loader.LoadResource(context.Background(), conn, "ui://test/empty.html")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestMCPResourceLoader_LoadResource_BadBase64(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)

	badBlob := "not-valid-base64!!!"
	conn := &mockMCPConnection{
		contents: []ResourceContent{
			{URI: "ui://test/bad.png", MimeType: "image/png", Blob: &badBlob},
		},
	}

	err := loader.LoadResource(context.Background(), conn, "ui://test/bad.png")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestMCPResourceLoader_LoadResource_MIMETypeDefault(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)

	text := "<html>no mime</html>"
	conn := &mockMCPConnection{
		contents: []ResourceContent{
			{URI: "ui://test/nomime.html", MimeType: "", Text: &text},
		},
	}

	if err := loader.LoadResource(context.Background(), conn, "ui://test/nomime.html"); err != nil {
		t.Fatalf("LoadResource failed: %v", err)
	}

	meta, err := reg.Get("ui://test/nomime.html")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.MimeType != "text/html" {
		t.Errorf("mime = %q, want text/html default", meta.MimeType)
	}
}

func TestMCPResourceLoader_LoadResource_ConnectionError(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)
	conn := &mockMCPConnection{
		err: fmt.Errorf("connection refused"),
	}

	err := loader.LoadResource(context.Background(), conn, "ui://test/fail.html")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestMCPResourceLoader_LoadResource_NoTextOrBlob(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)
	conn := &mockMCPConnection{
		contents: []ResourceContent{
			{URI: "ui://test/empty.html", MimeType: "text/html"},
		},
	}

	err := loader.LoadResource(context.Background(), conn, "ui://test/empty.html")
	if err == nil {
		t.Fatal("expected error when resource has neither text nor blob")
	}
}

func TestNewMCPResourceLoader(t *testing.T) {
	reg := NewResourceRegistry()
	loader := NewMCPResourceLoader(reg)
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}
}
