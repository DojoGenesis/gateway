package tools

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ensureWebAdvancedToolsRegistered() {
	tools := []string{
		"web_navigate",
		"web_scrape_structured",
		"web_screenshot",
		"web_monitor",
		"web_extract_links",
	}

	for _, toolName := range tools {
		if _, err := GetTool(toolName); err != nil {
			switch toolName {
			case "web_navigate":
				RegisterTool(&ToolDefinition{
					Name:        "web_navigate",
					Description: "Navigate to a URL using a headless browser and extract content",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "URL to navigate to (must start with http:// or https://)",
							},
						},
						"required": []string{"url"},
					},
					Function: WebNavigate,
				})
			case "web_scrape_structured":
				RegisterTool(&ToolDefinition{
					Name:        "web_scrape_structured",
					Description: "Extract structured data from a web page using CSS selectors",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "URL to scrape (must start with http:// or https://)",
							},
							"selectors": map[string]interface{}{
								"type":        "object",
								"description": "Map of field names to CSS selectors",
							},
						},
						"required": []string{"url", "selectors"},
					},
					Function: WebScrapeStructured,
				})
			case "web_screenshot":
				RegisterTool(&ToolDefinition{
					Name:        "web_screenshot",
					Description: "Take a screenshot of a web page",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "URL to screenshot (must start with http:// or https://)",
							},
						},
						"required": []string{"url"},
					},
					Function: WebScreenshot,
				})
			case "web_monitor":
				RegisterTool(&ToolDefinition{
					Name:        "web_monitor",
					Description: "Monitor a web page for changes using content hashing",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "URL to monitor (must start with http:// or https://)",
							},
						},
						"required": []string{"url"},
					},
					Function: WebMonitor,
				})
			case "web_extract_links":
				RegisterTool(&ToolDefinition{
					Name:        "web_extract_links",
					Description: "Extract all links from a web page",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "URL to extract links from (must start with http:// or https://)",
							},
						},
						"required": []string{"url"},
					},
					Function: WebExtractLinks,
				})
			}
		}
	}
}

func TestWebNavigate(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		wantSuccess    bool
		wantError      string
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name:        "missing url parameter",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "empty url",
			params: map[string]interface{}{
				"url": "",
			},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "invalid url protocol",
			params: map[string]interface{}{
				"url": "ftp://example.com",
			},
			wantSuccess: false,
			wantError:   "url must start with http:// or https://",
		},
		{
			name: "valid navigation with default options",
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "content")
				assert.Contains(t, result, "title")
				assert.Contains(t, result, "size_bytes")

				content, ok := result["content"].(string)
				require.True(t, ok)
				assert.NotEmpty(t, content)
				assert.Contains(t, strings.ToLower(content), "example")
			},
		},
		{
			name: "navigation with extract_content false",
			params: map[string]interface{}{
				"url":             "https://example.com",
				"extract_content": false,
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.NotContains(t, result, "content")
				assert.NotContains(t, result, "title")
			},
		},
		{
			name: "navigation with custom timeout",
			params: map[string]interface{}{
				"url":     "https://example.com",
				"timeout": 10,
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "content")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid navigation with default options" ||
				tt.name == "navigation with extract_content false" ||
				tt.name == "navigation with custom timeout" {
				t.Skip("Skipping browser automation tests in CI - requires headless browser")
			}

			ctx := context.Background()
			result, err := WebNavigate(ctx, tt.params)

			require.NoError(t, err)
			require.NotNil(t, result)

			success, ok := result["success"].(bool)
			require.True(t, ok)
			assert.Equal(t, tt.wantSuccess, success)

			if !tt.wantSuccess {
				errorMsg, ok := result["error"].(string)
				require.True(t, ok)
				assert.Contains(t, errorMsg, tt.wantError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestWebScrapeStructured(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		wantSuccess    bool
		wantError      string
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name:        "missing url parameter",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "missing selectors parameter",
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantSuccess: false,
			wantError:   "selectors parameter is required",
		},
		{
			name: "empty selectors",
			params: map[string]interface{}{
				"url":       "https://example.com",
				"selectors": map[string]interface{}{},
			},
			wantSuccess: false,
			wantError:   "selectors parameter is required and must not be empty",
		},
		{
			name: "invalid url protocol",
			params: map[string]interface{}{
				"url": "ftp://example.com",
				"selectors": map[string]interface{}{
					"title": "h1",
				},
			},
			wantSuccess: false,
			wantError:   "url must start with http:// or https://",
		},
		{
			name: "valid scraping with selectors",
			params: map[string]interface{}{
				"url": "https://example.com",
				"selectors": map[string]interface{}{
					"title":      "h1",
					"paragraphs": "p",
					"links":      "a",
				},
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "extracted")

				extracted, ok := result["extracted"].(map[string]interface{})
				require.True(t, ok)
				assert.NotNil(t, extracted)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid scraping with selectors" {
				t.Skip("Skipping browser automation tests in CI - requires headless browser")
			}

			ctx := context.Background()
			result, err := WebScrapeStructured(ctx, tt.params)

			require.NoError(t, err)
			require.NotNil(t, result)

			success, ok := result["success"].(bool)
			require.True(t, ok)
			assert.Equal(t, tt.wantSuccess, success)

			if !tt.wantSuccess {
				errorMsg, ok := result["error"].(string)
				require.True(t, ok)
				assert.Contains(t, errorMsg, tt.wantError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestWebScreenshot(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		wantSuccess    bool
		wantError      string
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name:        "missing url parameter",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "empty url",
			params: map[string]interface{}{
				"url": "",
			},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "invalid url protocol",
			params: map[string]interface{}{
				"url": "ftp://example.com",
			},
			wantSuccess: false,
			wantError:   "url must start with http:// or https://",
		},
		{
			name: "valid screenshot with default options",
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "screenshot")
				assert.Contains(t, result, "size_bytes")
				assert.Equal(t, "png", result["format"])

				screenshot, ok := result["screenshot"].(string)
				require.True(t, ok)
				assert.NotEmpty(t, screenshot)

				decoded, err := base64.StdEncoding.DecodeString(screenshot)
				require.NoError(t, err)
				assert.NotEmpty(t, decoded)

				assert.True(t, len(decoded) > 1000, "Screenshot should be at least 1KB")
			},
		},
		{
			name: "full page screenshot",
			params: map[string]interface{}{
				"url":       "https://example.com",
				"full_page": true,
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "screenshot")

				screenshot, ok := result["screenshot"].(string)
				require.True(t, ok)
				assert.NotEmpty(t, screenshot)
			},
		},
		{
			name: "screenshot with custom timeout",
			params: map[string]interface{}{
				"url":     "https://example.com",
				"timeout": 15,
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "screenshot")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid screenshot with default options" ||
				tt.name == "full page screenshot" ||
				tt.name == "screenshot with custom timeout" {
				t.Skip("Skipping browser automation tests in CI - requires headless browser")
			}

			ctx := context.Background()
			result, err := WebScreenshot(ctx, tt.params)

			require.NoError(t, err)
			require.NotNil(t, result)

			success, ok := result["success"].(bool)
			require.True(t, ok)
			assert.Equal(t, tt.wantSuccess, success)

			if !tt.wantSuccess {
				errorMsg, ok := result["error"].(string)
				require.True(t, ok)
				assert.Contains(t, errorMsg, tt.wantError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestWebMonitor(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		wantSuccess    bool
		wantError      string
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name:        "missing url parameter",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "empty url",
			params: map[string]interface{}{
				"url": "",
			},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "invalid url protocol",
			params: map[string]interface{}{
				"url": "ftp://example.com",
			},
			wantSuccess: false,
			wantError:   "url must start with http:// or https://",
		},
		{
			name: "valid monitoring without previous hash",
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "current_hash")
				assert.Contains(t, result, "content")
				assert.Contains(t, result, "changed")
				assert.False(t, result["changed"].(bool))

				currentHash, ok := result["current_hash"].(string)
				require.True(t, ok)
				assert.NotEmpty(t, currentHash)
				assert.Len(t, currentHash, 32)
			},
		},
		{
			name: "valid monitoring with previous hash - no change",
			params: map[string]interface{}{
				"url":           "https://example.com",
				"previous_hash": "dummy_hash",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "current_hash")
				assert.Contains(t, result, "previous_hash")
				assert.Equal(t, "dummy_hash", result["previous_hash"])
				assert.Contains(t, result, "changed")

				currentHash, ok := result["current_hash"].(string)
				require.True(t, ok)
				assert.NotEmpty(t, currentHash)

				changed, ok := result["changed"].(bool)
				require.True(t, ok)
				assert.True(t, changed)
			},
		},
		{
			name: "monitoring with custom selector",
			params: map[string]interface{}{
				"url":      "https://example.com",
				"selector": "h1",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "current_hash")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid monitoring without previous hash" ||
				tt.name == "valid monitoring with previous hash - no change" ||
				tt.name == "monitoring with custom selector" {
				t.Skip("Skipping browser automation tests in CI - requires headless browser")
			}

			ctx := context.Background()
			result, err := WebMonitor(ctx, tt.params)

			require.NoError(t, err)
			require.NotNil(t, result)

			success, ok := result["success"].(bool)
			require.True(t, ok)
			assert.Equal(t, tt.wantSuccess, success)

			if !tt.wantSuccess {
				errorMsg, ok := result["error"].(string)
				require.True(t, ok)
				assert.Contains(t, errorMsg, tt.wantError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestWebExtractLinks(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		wantSuccess    bool
		wantError      string
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name:        "missing url parameter",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "empty url",
			params: map[string]interface{}{
				"url": "",
			},
			wantSuccess: false,
			wantError:   "url parameter is required",
		},
		{
			name: "invalid url protocol",
			params: map[string]interface{}{
				"url": "ftp://example.com",
			},
			wantSuccess: false,
			wantError:   "url must start with http:// or https://",
		},
		{
			name: "valid link extraction with default options",
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "links")
				assert.Contains(t, result, "count")
				assert.Contains(t, result, "internal")
				assert.Contains(t, result, "external")

				links, ok := result["links"].([]string)
				require.True(t, ok)
				assert.NotNil(t, links)

				count, ok := result["count"].(int)
				require.True(t, ok)
				assert.Equal(t, len(links), count)
			},
		},
		{
			name: "link extraction with external links excluded",
			params: map[string]interface{}{
				"url":              "https://example.com",
				"include_external": false,
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "links")

				external, ok := result["external"].([]string)
				require.True(t, ok)
				assert.Empty(t, external)
			},
		},
		{
			name: "link extraction with filter pattern",
			params: map[string]interface{}{
				"url":            "https://example.com",
				"filter_pattern": ".*\\.html$",
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "links")

				links, ok := result["links"].([]string)
				require.True(t, ok)

				for _, link := range links {
					assert.True(t, strings.HasSuffix(link, ".html"), "Link should match filter pattern")
				}
			},
		},
		{
			name: "link extraction with custom timeout",
			params: map[string]interface{}{
				"url":     "https://example.com",
				"timeout": 15,
			},
			wantSuccess: true,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Contains(t, result, "links")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid link extraction with default options" ||
				tt.name == "link extraction with external links excluded" ||
				tt.name == "link extraction with filter pattern" ||
				tt.name == "link extraction with custom timeout" {
				t.Skip("Skipping browser automation tests in CI - requires headless browser")
			}

			ctx := context.Background()
			result, err := WebExtractLinks(ctx, tt.params)

			require.NoError(t, err)
			require.NotNil(t, result)

			success, ok := result["success"].(bool)
			require.True(t, ok)
			assert.Equal(t, tt.wantSuccess, success)

			if !tt.wantSuccess {
				errorMsg, ok := result["error"].(string)
				require.True(t, ok)
				assert.Contains(t, errorMsg, tt.wantError)
			} else if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "http url with path",
			url:      "http://example.com/path/to/page",
			expected: "example.com",
		},
		{
			name:     "https url with path",
			url:      "https://example.com/path/to/page",
			expected: "example.com",
		},
		{
			name:     "url without path",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "url with subdomain",
			url:      "https://www.example.com/page",
			expected: "www.example.com",
		},
		{
			name:     "url with port",
			url:      "https://example.com:8080/page",
			expected: "example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDomain(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebAdvancedToolsRegistration(t *testing.T) {
	ensureWebAdvancedToolsRegistered()

	tools := []string{
		"web_navigate",
		"web_scrape_structured",
		"web_screenshot",
		"web_monitor",
		"web_extract_links",
	}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			tool, err := GetTool(toolName)
			require.NoError(t, err)
			require.NotNil(t, tool)
			assert.Equal(t, toolName, tool.Name)
			assert.NotEmpty(t, tool.Description)
			assert.NotNil(t, tool.Parameters)
			assert.NotNil(t, tool.Function)
		})
	}
}

func TestWebAdvancedToolsTimeout(t *testing.T) {
	t.Run("web_navigate respects context timeout", func(t *testing.T) {
		t.Skip("Skipping timeout test - requires headless browser")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		params := map[string]interface{}{
			"url": "https://example.com",
		}

		result, err := WebNavigate(ctx, params)
		require.NoError(t, err)

		success, _ := result["success"].(bool)
		assert.False(t, success)
	})

	t.Run("web_screenshot respects context timeout", func(t *testing.T) {
		t.Skip("Skipping timeout test - requires headless browser")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		params := map[string]interface{}{
			"url": "https://example.com",
		}

		result, err := WebScreenshot(ctx, params)
		require.NoError(t, err)

		success, _ := result["success"].(bool)
		assert.False(t, success)
	})
}
