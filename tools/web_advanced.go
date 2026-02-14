package tools

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func WebNavigate(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	timeout := GetDurationParam(params, "timeout", defaultTimeout)
	waitForSelector := GetStringParam(params, "wait_for_selector", "")
	extractContent := GetBoolParam(params, "extract_content", true)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewContext(ctx)
	defer allocCancel()

	var htmlContent string
	var pageTitle string

	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
	}

	if waitForSelector != "" {
		tasks = append(tasks, chromedp.WaitVisible(waitForSelector))
	} else {
		tasks = append(tasks, chromedp.Sleep(1*time.Second))
	}

	if extractContent {
		tasks = append(tasks, chromedp.OuterHTML("html", &htmlContent))
		tasks = append(tasks, chromedp.Title(&pageTitle))
	}

	if err := chromedp.Run(allocCtx, tasks); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("navigation failed: %v", err),
		}, nil
	}

	result := map[string]interface{}{
		"success": true,
		"url":     url,
	}

	if extractContent {
		result["content"] = htmlContent
		result["title"] = pageTitle
		result["size_bytes"] = len(htmlContent)
	}

	return result, nil
}

func WebScrapeStructured(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	selectors := GetMapParam(params, "selectors", map[string]interface{}{})
	if len(selectors) == 0 {
		return map[string]interface{}{
			"success": false,
			"error":   "selectors parameter is required and must not be empty",
		}, nil
	}

	timeout := GetDurationParam(params, "timeout", defaultTimeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewContext(ctx)
	defer allocCancel()

	var htmlContent string

	if err := chromedp.Run(allocCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(1*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("navigation failed: %v", err),
		}, nil
	}

	extracted := make(map[string]interface{})

	for key, selectorVal := range selectors {
		selector, ok := selectorVal.(string)
		if !ok {
			continue
		}

		var textContent string
		textTasks := chromedp.Tasks{
			chromedp.TextContent(selector, &textContent, chromedp.ByQuery),
		}

		allocCtx2, allocCancel2 := chromedp.NewContext(ctx)
		if err := chromedp.Run(allocCtx2, textTasks); err == nil {
			if textContent != "" {
				extracted[key] = textContent
			} else {
				extracted[key] = ""
			}
		} else {
			extracted[key] = ""
		}
		allocCancel2()
	}

	return map[string]interface{}{
		"success":   true,
		"url":       url,
		"extracted": extracted,
	}, nil
}

func WebScreenshot(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	timeout := GetDurationParam(params, "timeout", defaultTimeout)
	fullPage := GetBoolParam(params, "full_page", false)
	waitForSelector := GetStringParam(params, "wait_for_selector", "")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewContext(ctx)
	defer allocCancel()

	var buf []byte

	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
	}

	if waitForSelector != "" {
		tasks = append(tasks, chromedp.WaitVisible(waitForSelector))
	} else {
		tasks = append(tasks, chromedp.Sleep(1*time.Second))
	}

	if fullPage {
		tasks = append(tasks, chromedp.FullScreenshot(&buf, 90))
	} else {
		tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(allocCtx, tasks); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("screenshot failed: %v", err),
		}, nil
	}

	base64Image := base64.StdEncoding.EncodeToString(buf)

	return map[string]interface{}{
		"success":    true,
		"url":        url,
		"screenshot": base64Image,
		"size_bytes": len(buf),
		"format":     "png",
	}, nil
}

func WebMonitor(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	previousHash := GetStringParam(params, "previous_hash", "")
	selector := GetStringParam(params, "selector", "body")
	timeout := GetDurationParam(params, "timeout", defaultTimeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewContext(ctx)
	defer allocCancel()

	var content string

	if err := chromedp.Run(allocCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(1*time.Second),
		chromedp.TextContent(selector, &content, chromedp.ByQuery),
	); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("monitoring failed: %v", err),
		}, nil
	}

	currentHash := fmt.Sprintf("%x", md5.Sum([]byte(content)))

	changed := false
	if previousHash != "" && previousHash != currentHash {
		changed = true
	}

	return map[string]interface{}{
		"success":       true,
		"url":           url,
		"current_hash":  currentHash,
		"previous_hash": previousHash,
		"changed":       changed,
		"content":       content,
		"size_bytes":    len(content),
	}, nil
}

func WebExtractLinks(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	timeout := GetDurationParam(params, "timeout", defaultTimeout)
	filterPattern := GetStringParam(params, "filter_pattern", "")
	includeExternal := GetBoolParam(params, "include_external", true)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewContext(ctx)
	defer allocCancel()

	var htmlContent string

	if err := chromedp.Run(allocCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(1*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("navigation failed: %v", err),
		}, nil
	}

	linkRegex := regexp.MustCompile(`href=["']([^"']+)["']`)
	matches := linkRegex.FindAllStringSubmatch(htmlContent, -1)

	linksMap := make(map[string]bool)
	var links []string

	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]

			if !includeExternal {
				if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
					if !strings.Contains(link, extractDomain(url)) {
						continue
					}
				}
			}

			if filterPattern != "" {
				matched, _ := regexp.MatchString(filterPattern, link)
				if !matched {
					continue
				}
			}

			if !linksMap[link] {
				linksMap[link] = true
				links = append(links, link)
			}
		}
	}

	internal := []string{}
	external := []string{}

	baseDomain := extractDomain(url)
	for _, link := range links {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			if strings.Contains(link, baseDomain) {
				internal = append(internal, link)
			} else {
				external = append(external, link)
			}
		} else {
			internal = append(internal, link)
		}
	}

	return map[string]interface{}{
		"success":  true,
		"url":      url,
		"links":    links,
		"count":    len(links),
		"internal": internal,
		"external": external,
	}, nil
}

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return url
}

func init() {
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
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Navigation timeout in seconds (default: 30)",
					"default":     30,
				},
				"wait_for_selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector to wait for before extracting content (optional)",
				},
				"extract_content": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to extract page content (default: true)",
					"default":     true,
				},
			},
			"required": []string{"url"},
		},
		Function: WebNavigate,
	})

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
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
			},
			"required": []string{"url", "selectors"},
		},
		Function: WebScrapeStructured,
	})

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
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
				"full_page": map[string]interface{}{
					"type":        "boolean",
					"description": "Take full page screenshot (default: false)",
					"default":     false,
				},
				"wait_for_selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector to wait for before taking screenshot (optional)",
				},
			},
			"required": []string{"url"},
		},
		Function: WebScreenshot,
	})

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
				"previous_hash": map[string]interface{}{
					"type":        "string",
					"description": "Previous content hash for comparison (optional)",
				},
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector to monitor (default: body)",
					"default":     "body",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
			},
			"required": []string{"url"},
		},
		Function: WebMonitor,
	})

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
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
				"filter_pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regex pattern to filter links (optional)",
				},
				"include_external": map[string]interface{}{
					"type":        "boolean",
					"description": "Include external links (default: true)",
					"default":     true,
				},
			},
			"required": []string{"url"},
		},
		Function: WebExtractLinks,
	})
}
