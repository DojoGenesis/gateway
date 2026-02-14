package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/wcharczuk/go-chart/v2"
)

func validateDiagramSyntax(syntax string, diagramType string) []string {
	var errors []string

	if strings.TrimSpace(syntax) == "" {
		errors = append(errors, "syntax cannot be empty")
		return errors
	}

	switch diagramType {
	case "mermaid":
		errors = append(errors, validateMermaidSyntax(syntax)...)
	case "d2":
		errors = append(errors, validateD2Syntax(syntax)...)
	default:
		errors = append(errors, fmt.Sprintf("unsupported diagram type: %s", diagramType))
	}

	return errors
}

func validateMermaidSyntax(syntax string) []string {
	var errors []string
	lines := strings.Split(syntax, "\n")

	if len(lines) == 0 {
		errors = append(errors, "diagram must have at least one line")
		return errors
	}

	firstLine := strings.TrimSpace(lines[0])
	validStarters := []string{
		"graph", "flowchart", "sequenceDiagram", "classDiagram",
		"stateDiagram", "erDiagram", "journey", "gantt", "pie",
		"gitGraph", "mindmap", "timeline", "quadrantChart",
	}

	hasValidStarter := false
	for _, starter := range validStarters {
		if strings.HasPrefix(firstLine, starter) {
			hasValidStarter = true
			break
		}
	}

	if !hasValidStarter {
		errors = append(errors, fmt.Sprintf("diagram must start with a valid keyword (graph, sequenceDiagram, etc.), got: %s", firstLine))
	}

	openBrackets := strings.Count(syntax, "[") + strings.Count(syntax, "(") + strings.Count(syntax, "{")
	closeBrackets := strings.Count(syntax, "]") + strings.Count(syntax, ")") + strings.Count(syntax, "}")
	if openBrackets != closeBrackets {
		errors = append(errors, fmt.Sprintf("unbalanced brackets: %d open, %d close", openBrackets, closeBrackets))
	}

	return errors
}

func validateD2Syntax(syntax string) []string {
	var errors []string

	if !strings.Contains(syntax, "->") && !strings.Contains(syntax, "--") && !strings.Contains(syntax, ":") {
		errors = append(errors, "D2 diagram must contain at least one connection (->) or property (:)")
	}

	openBraces := strings.Count(syntax, "{")
	closeBraces := strings.Count(syntax, "}")
	if openBraces != closeBraces {
		errors = append(errors, fmt.Sprintf("unbalanced braces: %d open, %d close", openBraces, closeBraces))
	}

	return errors
}

func validateChartData(chartType string, data map[string]interface{}) []string {
	var errors []string

	switch chartType {
	case "line", "bar":
		if _, hasLabels := data["labels"]; !hasLabels {
			errors = append(errors, "line and bar charts require 'labels' array")
		}
		if _, hasDatasets := data["datasets"]; !hasDatasets {
			errors = append(errors, "chart requires 'datasets' array")
		} else {
			datasets, ok := data["datasets"].([]interface{})
			if !ok {
				errors = append(errors, "'datasets' must be an array")
			} else if len(datasets) == 0 {
				errors = append(errors, "'datasets' array cannot be empty")
			}
		}

	case "pie":
		if _, hasLabels := data["labels"]; !hasLabels {
			errors = append(errors, "pie chart requires 'labels' array")
		}
		if _, hasDatasets := data["datasets"]; !hasDatasets {
			errors = append(errors, "pie chart requires 'datasets' array")
		}

	case "scatter":
		if _, hasDatasets := data["datasets"]; !hasDatasets {
			errors = append(errors, "scatter chart requires 'datasets' array")
		}

	default:
		errors = append(errors, fmt.Sprintf("unsupported chart type: %s", chartType))
	}

	return errors
}

func PrepareDiagram(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	syntax := GetStringParam(params, "syntax", "")
	if syntax == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "syntax is required",
		}, nil
	}

	diagramType := GetStringParam(params, "type", "mermaid")
	if diagramType != "mermaid" && diagramType != "d2" {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported diagram type: %s (supported: mermaid, d2)", diagramType),
		}, nil
	}

	validationErrors := validateDiagramSyntax(syntax, diagramType)

	hash := sha256.Sum256([]byte(syntax))
	diagramID := fmt.Sprintf("%x", hash[:8])

	metadata := map[string]interface{}{
		"diagram_id":   diagramID,
		"type":         diagramType,
		"syntax_lines": len(strings.Split(syntax, "\n")),
		"generated_at": time.Now().Format(time.RFC3339),
	}

	format := GetStringParam(params, "format", "svg")

	return map[string]interface{}{
		"success":           len(validationErrors) == 0,
		"syntax":            syntax,
		"type":              diagramType,
		"format":            format,
		"validated":         len(validationErrors) == 0,
		"validation_errors": validationErrors,
		"metadata":          metadata,
		"message": func() string {
			if len(validationErrors) > 0 {
				return fmt.Sprintf("%s diagram validation failed with %d error(s)", diagramType, len(validationErrors))
			}
			return fmt.Sprintf("%s diagram validated and ready for rendering (ID: %s)", diagramType, diagramID)
		}(),
	}, nil
}

func PrepareChart(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	data := GetMapParam(params, "data", nil)
	if data == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "data is required",
		}, nil
	}

	chartType := GetStringParam(params, "chart_type", "")
	if chartType == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "chart_type is required",
		}, nil
	}

	validationErrors := validateChartData(chartType, data)

	chartData, err := json.Marshal(data)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid chart data: %v", err),
		}, nil
	}

	hash := sha256.Sum256(chartData)
	chartID := fmt.Sprintf("%x", hash[:8])

	options := GetMapParam(params, "options", map[string]interface{}{})

	metadata := map[string]interface{}{
		"chart_id":     chartID,
		"chart_type":   chartType,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	return map[string]interface{}{
		"success":           len(validationErrors) == 0,
		"chart_id":          chartID,
		"chart_type":        chartType,
		"data":              data,
		"options":           options,
		"validated":         len(validationErrors) == 0,
		"validation_errors": validationErrors,
		"metadata":          metadata,
		"message": func() string {
			if len(validationErrors) > 0 {
				return fmt.Sprintf("%s chart validation failed with %d error(s)", chartType, len(validationErrors))
			}
			return fmt.Sprintf("%s chart validated and ready for rendering (ID: %s)", chartType, chartID)
		}(),
	}, nil
}

func ExportDiagram(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	syntax := GetStringParam(params, "syntax", "")
	if syntax == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "syntax is required",
		}, nil
	}

	diagramType := GetStringParam(params, "type", "mermaid")
	format := GetStringParam(params, "format", "svg")

	if diagramType != "mermaid" && diagramType != "d2" {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported diagram type: %s", diagramType),
		}, nil
	}

	if format != "svg" && format != "png" {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported format: %s (supported: svg, png)", format),
		}, nil
	}

	validationErrors := validateDiagramSyntax(syntax, diagramType)
	if len(validationErrors) > 0 {
		return map[string]interface{}{
			"success":           false,
			"error":             "diagram validation failed",
			"validation_errors": validationErrors,
		}, nil
	}

	rendered, err := renderDiagramWithChromedp(ctx, syntax, diagramType, format)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to render diagram: %v", err),
		}, nil
	}

	hash := sha256.Sum256([]byte(syntax))
	diagramID := fmt.Sprintf("%x", hash[:8])
	filename := fmt.Sprintf("diagram_%s.%s", diagramID, format)

	outputPath := filepath.Join(os.TempDir(), filename)
	if err := os.WriteFile(outputPath, rendered, 0644); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to save diagram: %v", err),
		}, nil
	}

	var content string
	if format == "svg" {
		content = string(rendered)
	} else {
		content = base64.StdEncoding.EncodeToString(rendered)
	}

	return map[string]interface{}{
		"success":  true,
		"type":     diagramType,
		"format":   format,
		"filename": filename,
		"path":     outputPath,
		"content":  content,
		"size":     len(rendered),
		"message":  fmt.Sprintf("%s diagram exported as %s (%d bytes)", diagramType, format, len(rendered)),
	}, nil
}

func renderDiagramWithChromedp(ctx context.Context, syntax, diagramType, format string) ([]byte, error) {
	var htmlTemplate string
	if diagramType == "mermaid" {
		htmlTemplate = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
    <style>
        body { margin: 0; padding: 20px; background: white; }
        #diagram { display: inline-block; }
    </style>
</head>
<body>
    <div id="diagram" class="mermaid">
%s
    </div>
    <script>
        mermaid.initialize({ startOnLoad: true, theme: 'default' });
    </script>
</body>
</html>
`, syntax)
	} else {
		return nil, fmt.Errorf("d2 rendering via chromedp not yet implemented (requires d2 CLI)")
	}

	allocCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var buf []byte
	var err error

	if format == "svg" {
		var svgContent string
		err = chromedp.Run(allocCtx,
			chromedp.Navigate("data:text/html,"+htmlTemplate),
			chromedp.Sleep(2*time.Second),
			chromedp.OuterHTML("#diagram svg", &svgContent),
		)
		if err == nil {
			buf = []byte(svgContent)
		}
	} else {
		err = chromedp.Run(allocCtx,
			chromedp.Navigate("data:text/html,"+htmlTemplate),
			chromedp.Sleep(2*time.Second),
			chromedp.FullScreenshot(&buf, 90),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("chromedp rendering failed: %w", err)
	}

	return buf, nil
}

func ExportChart(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	data := GetMapParam(params, "data", nil)
	if data == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "data is required",
		}, nil
	}

	chartType := GetStringParam(params, "chart_type", "")
	if chartType == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "chart_type is required",
		}, nil
	}

	format := GetStringParam(params, "format", "png")
	if format != "png" && format != "svg" {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported format: %s (supported: png, svg)", format),
		}, nil
	}

	validationErrors := validateChartData(chartType, data)
	if len(validationErrors) > 0 {
		return map[string]interface{}{
			"success":           false,
			"error":             "chart validation failed",
			"validation_errors": validationErrors,
		}, nil
	}

	rendered, err := renderChartWithGoChart(data, chartType, format)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to render chart: %v", err),
		}, nil
	}

	chartData, _ := json.Marshal(data)
	hash := sha256.Sum256(chartData)
	chartID := fmt.Sprintf("%x", hash[:8])
	filename := fmt.Sprintf("chart_%s.%s", chartID, format)

	outputPath := filepath.Join(os.TempDir(), filename)
	if err := os.WriteFile(outputPath, rendered, 0644); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to save chart: %v", err),
		}, nil
	}

	var content string
	if format == "svg" {
		content = string(rendered)
	} else {
		content = base64.StdEncoding.EncodeToString(rendered)
	}

	return map[string]interface{}{
		"success":    true,
		"chart_type": chartType,
		"format":     format,
		"filename":   filename,
		"path":       outputPath,
		"content":    content,
		"size":       len(rendered),
		"message":    fmt.Sprintf("%s chart exported as %s (%d bytes)", chartType, format, len(rendered)),
	}, nil
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	default:
		return 0
	}
}

func renderChartWithGoChart(data map[string]interface{}, chartType, format string) ([]byte, error) {
	switch chartType {
	case "bar":
		return renderBarChart(data, format)
	case "line":
		return renderLineChart(data, format)
	case "pie":
		return renderPieChart(data, format)
	default:
		return nil, fmt.Errorf("unsupported chart type for go-chart: %s", chartType)
	}
}

func renderBarChart(data map[string]interface{}, format string) ([]byte, error) {
	labels, ok := data["labels"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid labels format")
	}

	datasets, ok := data["datasets"].([]interface{})
	if !ok || len(datasets) == 0 {
		return nil, fmt.Errorf("invalid datasets format")
	}

	dataset := datasets[0].(map[string]interface{})
	dataValues, ok := dataset["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data values")
	}

	var bars []chart.Value
	for i, label := range labels {
		if i < len(dataValues) {
			val := toFloat64(dataValues[i])
			bars = append(bars, chart.Value{
				Label: fmt.Sprintf("%v", label),
				Value: val,
			})
		}
	}

	barChart := chart.BarChart{
		Title:  "Bar Chart",
		Width:  800,
		Height: 600,
		Bars:   bars,
	}

	buffer := &bytes.Buffer{}
	var err error
	if format == "svg" {
		err = barChart.Render(chart.SVG, buffer)
	} else {
		err = barChart.Render(chart.PNG, buffer)
	}

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func renderLineChart(data map[string]interface{}, format string) ([]byte, error) {
	labels, ok := data["labels"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid labels format")
	}

	datasets, ok := data["datasets"].([]interface{})
	if !ok || len(datasets) == 0 {
		return nil, fmt.Errorf("invalid datasets format")
	}

	dataset := datasets[0].(map[string]interface{})
	dataValues, ok := dataset["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data values")
	}

	var xValues []float64
	var yValues []float64
	for i := 0; i < len(labels) && i < len(dataValues); i++ {
		xValues = append(xValues, float64(i))
		val := toFloat64(dataValues[i])
		yValues = append(yValues, val)
	}

	graph := chart.Chart{
		Width:  800,
		Height: 600,
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: xValues,
				YValues: yValues,
			},
		},
	}

	buffer := &bytes.Buffer{}
	var err error
	if format == "svg" {
		err = graph.Render(chart.SVG, buffer)
	} else {
		err = graph.Render(chart.PNG, buffer)
	}

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func renderPieChart(data map[string]interface{}, format string) ([]byte, error) {
	labels, ok := data["labels"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid labels format")
	}

	datasets, ok := data["datasets"].([]interface{})
	if !ok || len(datasets) == 0 {
		return nil, fmt.Errorf("invalid datasets format")
	}

	dataset := datasets[0].(map[string]interface{})
	dataValues, ok := dataset["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data values")
	}

	var values []chart.Value
	for i, label := range labels {
		if i < len(dataValues) {
			val := toFloat64(dataValues[i])
			values = append(values, chart.Value{
				Label: fmt.Sprintf("%v", label),
				Value: val,
			})
		}
	}

	pie := chart.PieChart{
		Title:  "Pie Chart",
		Width:  800,
		Height: 600,
		Values: values,
	}

	buffer := &bytes.Buffer{}
	var err error
	if format == "svg" {
		err = pie.Render(chart.SVG, buffer)
	} else {
		err = pie.Render(chart.PNG, buffer)
	}

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func GenerateImage(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	prompt := GetStringParam(params, "prompt", "")
	if prompt == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "prompt is required",
		}, nil
	}

	style := GetStringParam(params, "style", "realistic")
	width := GetIntParam(params, "width", 1024)
	height := GetIntParam(params, "height", 1024)

	if width < 256 || width > 2048 {
		width = 1024
	}
	if height < 256 || height > 2048 {
		height = 1024
	}

	hash := sha256.Sum256([]byte(prompt))
	imageID := fmt.Sprintf("%x", hash[:8])

	metadata := map[string]interface{}{
		"image_id":   imageID,
		"prompt":     prompt,
		"style":      style,
		"dimensions": fmt.Sprintf("%dx%d", width, height),
		"status":     "pending",
		"created_at": time.Now().Format(time.RFC3339),
	}

	return map[string]interface{}{
		"success":  true,
		"image_id": imageID,
		"prompt":   prompt,
		"style":    style,
		"width":    width,
		"height":   height,
		"metadata": metadata,
		"message":  "Image generation API not yet integrated. This is a placeholder for future implementation.",
		"note":     "To enable image generation, integrate with an API like Stability AI, DALL-E, or a local diffusion model.",
	}, nil
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "prepare_diagram",
		Description: "Validate and prepare diagram syntax (Mermaid or D2) for client-side rendering. Performs syntax validation and returns the diagram specification ready for frontend visualization engines like Mermaid.js.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"syntax": map[string]interface{}{
					"type":        "string",
					"description": "Diagram syntax (Mermaid or D2 code)",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Type of diagram syntax",
					"enum":        []string{"mermaid", "d2"},
					"default":     "mermaid",
				},
			},
			"required": []string{"syntax", "type"},
		},
		Function: PrepareDiagram,
	})

	RegisterTool(&ToolDefinition{
		Name:        "prepare_chart",
		Description: "Validate and prepare chart data specification for client-side rendering. Validates data format and chart type, returning a Chart.js-compatible configuration ready for frontend visualization.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type":        "object",
					"description": "Chart data in JSON format with labels and datasets",
				},
				"chart_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of chart to create",
					"enum":        []string{"line", "bar", "pie", "scatter"},
				},
				"options": map[string]interface{}{
					"type":        "object",
					"description": "Optional chart configuration (titles, colors, axes, etc.)",
				},
			},
			"required": []string{"data", "chart_type"},
		},
		Function: PrepareChart,
	})

	RegisterTool(&ToolDefinition{
		Name:        "export_diagram",
		Description: "Render and export a diagram to SVG or PNG format. Uses headless browser rendering (chromedp) to generate high-quality diagram images from Mermaid syntax. Saves the rendered output to a file and returns the content.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"syntax": map[string]interface{}{
					"type":        "string",
					"description": "Diagram syntax (Mermaid or D2 code)",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Type of diagram syntax",
					"enum":        []string{"mermaid", "d2"},
					"default":     "mermaid",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for the diagram",
					"enum":        []string{"svg", "png"},
					"default":     "svg",
				},
			},
			"required": []string{"syntax", "type"},
		},
		Function: ExportDiagram,
	})

	RegisterTool(&ToolDefinition{
		Name:        "export_chart",
		Description: "Render and export a chart to PNG or SVG format. Uses go-chart library to generate publication-quality chart images from data. Supports bar charts, line charts, and pie charts. Saves the rendered output to a file and returns the content.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type":        "object",
					"description": "Chart data in JSON format with labels and datasets",
				},
				"chart_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of chart to create",
					"enum":        []string{"line", "bar", "pie"},
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for the chart",
					"enum":        []string{"png", "svg"},
					"default":     "png",
				},
			},
			"required": []string{"data", "chart_type"},
		},
		Function: ExportChart,
	})

	RegisterTool(&ToolDefinition{
		Name:        "generate_image",
		Description: "Generate an AI image from a text prompt. This is a placeholder for future integration with image generation APIs like Stability AI, DALL-E, or local diffusion models.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Text description of the image to generate",
				},
				"style": map[string]interface{}{
					"type":        "string",
					"description": "Art style for the image (e.g., realistic, anime, abstract, oil-painting)",
					"default":     "realistic",
				},
				"width": map[string]interface{}{
					"type":        "integer",
					"description": "Image width in pixels (256-2048)",
					"default":     1024,
				},
				"height": map[string]interface{}{
					"type":        "integer",
					"description": "Image height in pixels (256-2048)",
					"default":     1024,
				},
			},
			"required": []string{"prompt"},
		},
		Function: GenerateImage,
	})
}
