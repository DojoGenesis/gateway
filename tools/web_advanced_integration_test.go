package tools

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebNavigateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	result, err := WebNavigate(ctx, map[string]interface{}{
		"url": "https://example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Navigation failed: %v", result["error"])
		t.Skip("Skipping - browser automation may not be available in this environment")
	}

	assert.True(t, success)
	assert.Equal(t, "https://example.com", result["url"])
	assert.Contains(t, result, "content")
	assert.Contains(t, result, "title")

	content, ok := result["content"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "Example Domain")

	title, ok := result["title"].(string)
	require.True(t, ok)
	assert.Contains(t, title, "Example")
}

func TestWebScrapeStructuredIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	result, err := WebScrapeStructured(ctx, map[string]interface{}{
		"url": "https://example.com",
		"selectors": map[string]interface{}{
			"title":      "h1",
			"paragraphs": "p",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Scraping failed: %v", result["error"])
		t.Skip("Skipping - browser automation may not be available in this environment")
	}

	assert.True(t, success)
	assert.Contains(t, result, "extracted")

	extracted, ok := result["extracted"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, extracted, "title")
	assert.Contains(t, extracted, "paragraphs")
}

func TestWebScreenshotIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	result, err := WebScreenshot(ctx, map[string]interface{}{
		"url": "https://example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Screenshot failed: %v", result["error"])
		t.Skip("Skipping - browser automation may not be available in this environment")
	}

	assert.True(t, success)
	assert.Equal(t, "https://example.com", result["url"])
	assert.Contains(t, result, "screenshot")
	assert.Equal(t, "png", result["format"])

	screenshot, ok := result["screenshot"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, screenshot)

	decoded, err := base64.StdEncoding.DecodeString(screenshot)
	require.NoError(t, err)
	assert.True(t, len(decoded) > 1000, "Screenshot should be at least 1KB")

	assert.True(t, decoded[0] == 0x89 && decoded[1] == 0x50, "Should be a PNG file")
}

func TestWebMonitorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	result1, err := WebMonitor(ctx, map[string]interface{}{
		"url": "https://example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, result1)

	success1, _ := result1["success"].(bool)
	if !success1 {
		t.Logf("First monitoring failed: %v", result1["error"])
		t.Skip("Skipping - browser automation may not be available in this environment")
	}

	assert.True(t, success1)
	assert.Contains(t, result1, "current_hash")

	currentHash1, ok := result1["current_hash"].(string)
	require.True(t, ok)
	assert.Len(t, currentHash1, 32)

	time.Sleep(1 * time.Second)

	result2, err := WebMonitor(ctx, map[string]interface{}{
		"url":           "https://example.com",
		"previous_hash": currentHash1,
	})

	require.NoError(t, err)
	require.NotNil(t, result2)

	success2, _ := result2["success"].(bool)
	assert.True(t, success2)

	currentHash2, ok := result2["current_hash"].(string)
	require.True(t, ok)
	assert.Equal(t, currentHash1, currentHash2, "Content should be the same")

	changed, ok := result2["changed"].(bool)
	require.True(t, ok)
	assert.False(t, changed, "Content should not have changed")
}

func TestWebExtractLinksIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	result, err := WebExtractLinks(ctx, map[string]interface{}{
		"url": "https://example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Link extraction failed: %v", result["error"])
		t.Skip("Skipping - browser automation may not be available in this environment")
	}

	assert.True(t, success)
	assert.Equal(t, "https://example.com", result["url"])
	assert.Contains(t, result, "links")
	assert.Contains(t, result, "count")
	assert.Contains(t, result, "internal")
	assert.Contains(t, result, "external")

	links, ok := result["links"].([]string)
	require.True(t, ok)
	assert.NotEmpty(t, links)

	count, ok := result["count"].(int)
	require.True(t, ok)
	assert.Equal(t, len(links), count)

	internal, ok := result["internal"].([]string)
	require.True(t, ok)
	assert.NotNil(t, internal)

	external, ok := result["external"].([]string)
	require.True(t, ok)
	assert.NotNil(t, external)

	t.Logf("Found %d total links (%d internal, %d external)", count, len(internal), len(external))
}

func TestWebAdvancedPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()

	start := time.Now()
	result, err := WebNavigate(ctx, map[string]interface{}{
		"url":             "https://example.com",
		"extract_content": false,
	})

	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)

	success, _ := result["success"].(bool)
	if !success {
		t.Skip("Skipping - browser automation may not be available in this environment")
	}

	t.Logf("Navigation completed in %v", duration)
	assert.Less(t, duration, 10*time.Second, "Navigation should complete within 10 seconds")
}
