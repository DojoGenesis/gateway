package tools

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareDiagram_Success(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantType    string
		wantSuccess bool
	}{
		{
			name: "mermaid flowchart",
			params: map[string]interface{}{
				"syntax": "graph TD\n    A[Start] --> B[End]",
				"type":   "mermaid",
			},
			wantType:    "mermaid",
			wantSuccess: true,
		},
		{
			name: "mermaid sequence diagram",
			params: map[string]interface{}{
				"syntax": "sequenceDiagram\n    Alice->>Bob: Hello\n    Bob->>Alice: Hi",
				"type":   "mermaid",
			},
			wantType:    "mermaid",
			wantSuccess: true,
		},
		{
			name: "d2 diagram",
			params: map[string]interface{}{
				"syntax": "x -> y: hello",
				"type":   "d2",
			},
			wantType:    "d2",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareDiagram(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantSuccess, result["success"])
			assert.Equal(t, tt.wantType, result["type"])
			assert.Equal(t, tt.params["syntax"], result["syntax"])
			assert.True(t, result["validated"].(bool))

			metadata, ok := result["metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, metadata["diagram_id"])
			assert.NotEmpty(t, metadata["generated_at"])
		})
	}
}

func TestPrepareDiagram_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   string
		expectValid bool
	}{
		{
			name:        "missing syntax",
			params:      map[string]interface{}{"type": "mermaid"},
			wantError:   "syntax is required",
			expectValid: false,
		},
		{
			name: "invalid diagram type",
			params: map[string]interface{}{
				"syntax": "graph TD",
				"type":   "invalid",
			},
			wantError:   "unsupported diagram type: invalid",
			expectValid: false,
		},
		{
			name: "empty syntax",
			params: map[string]interface{}{
				"syntax": "",
				"type":   "mermaid",
			},
			wantError:   "syntax is required",
			expectValid: false,
		},
		{
			name: "invalid mermaid syntax",
			params: map[string]interface{}{
				"syntax": "not a valid diagram",
				"type":   "mermaid",
			},
			wantError:   "diagram must start with a valid keyword",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareDiagram(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.False(t, result["success"].(bool))
			if tt.wantError != "" {
				if errorStr, ok := result["error"].(string); ok {
					assert.Contains(t, errorStr, tt.wantError)
				} else if validationErrors, ok := result["validation_errors"].([]string); ok {
					found := false
					for _, e := range validationErrors {
						if assert.ObjectsAreEqualValues(e, tt.wantError) {
							found = true
							break
						}
					}
					if !found {
						t.Logf("Validation errors: %v", validationErrors)
					}
				}
			}
		})
	}
}

func TestPrepareChart_Success(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantType    string
		wantSuccess bool
	}{
		{
			name: "line chart",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"Jan", "Feb", "Mar"},
					"datasets": []interface{}{
						map[string]interface{}{
							"label": "Sales",
							"data":  []interface{}{10, 20, 30},
						},
					},
				},
				"chart_type": "line",
			},
			wantType:    "line",
			wantSuccess: true,
		},
		{
			name: "bar chart with options",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"Q1", "Q2", "Q3", "Q4"},
					"datasets": []interface{}{
						map[string]interface{}{
							"label": "Revenue",
							"data":  []interface{}{100, 150, 200, 250},
						},
					},
				},
				"chart_type": "bar",
				"options": map[string]interface{}{
					"title": map[string]interface{}{
						"text": "Quarterly Revenue",
					},
				},
			},
			wantType:    "bar",
			wantSuccess: true,
		},
		{
			name: "pie chart",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"Red", "Blue", "Yellow"},
					"datasets": []interface{}{
						map[string]interface{}{
							"data": []interface{}{300, 50, 100},
						},
					},
				},
				"chart_type": "pie",
			},
			wantType:    "pie",
			wantSuccess: true,
		},
		{
			name: "scatter chart",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"datasets": []interface{}{
						map[string]interface{}{
							"label": "Points",
							"data": []interface{}{
								map[string]interface{}{"x": 1, "y": 2},
								map[string]interface{}{"x": 2, "y": 4},
							},
						},
					},
				},
				"chart_type": "scatter",
			},
			wantType:    "scatter",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareChart(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantSuccess, result["success"])
			assert.Equal(t, tt.wantType, result["chart_type"])
			assert.NotEmpty(t, result["chart_id"])

			data, ok := result["data"].(map[string]interface{})
			require.True(t, ok)
			assert.NotNil(t, data)

			metadata, ok := result["metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, metadata["chart_id"])
			assert.NotEmpty(t, metadata["generated_at"])
		})
	}
}

func TestPrepareChart_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError string
	}{
		{
			name:      "missing data",
			params:    map[string]interface{}{"chart_type": "line"},
			wantError: "data is required",
		},
		{
			name: "missing chart_type",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"A", "B"},
				},
			},
			wantError: "chart_type is required",
		},
		{
			name: "missing labels for line chart",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"datasets": []interface{}{
						map[string]interface{}{
							"data": []interface{}{1, 2, 3},
						},
					},
				},
				"chart_type": "line",
			},
			wantError: "chart validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareChart(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.False(t, result["success"].(bool))
			if errorStr, ok := result["error"].(string); ok {
				assert.Contains(t, errorStr, tt.wantError)
			}
		})
	}
}

func TestExportDiagram_ValidationFailure(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError string
	}{
		{
			name:      "missing syntax",
			params:    map[string]interface{}{"type": "mermaid"},
			wantError: "syntax is required",
		},
		{
			name: "invalid diagram type",
			params: map[string]interface{}{
				"syntax": "graph TD",
				"type":   "invalid",
			},
			wantError: "unsupported diagram type",
		},
		{
			name: "invalid format",
			params: map[string]interface{}{
				"syntax": "graph TD",
				"type":   "mermaid",
				"format": "gif",
			},
			wantError: "unsupported format",
		},
		{
			name: "invalid mermaid syntax",
			params: map[string]interface{}{
				"syntax": "not a valid diagram",
				"type":   "mermaid",
				"format": "svg",
			},
			wantError: "diagram validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExportDiagram(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.False(t, result["success"].(bool))
			assert.Contains(t, result["error"].(string), tt.wantError)
		})
	}
}

func TestExportChart_ValidationFailure(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError string
	}{
		{
			name:      "missing data",
			params:    map[string]interface{}{"chart_type": "line"},
			wantError: "data is required",
		},
		{
			name: "missing chart_type",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"A", "B"},
				},
			},
			wantError: "chart_type is required",
		},
		{
			name: "invalid format",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"A", "B"},
					"datasets": []interface{}{
						map[string]interface{}{
							"data": []interface{}{1, 2},
						},
					},
				},
				"chart_type": "bar",
				"format":     "gif",
			},
			wantError: "unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExportChart(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.False(t, result["success"].(bool))
			assert.Contains(t, result["error"].(string), tt.wantError)
		})
	}
}

func TestExportChart_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		params     map[string]interface{}
		wantType   string
		wantFormat string
	}{
		{
			name: "bar chart png",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"A", "B", "C"},
					"datasets": []interface{}{
						map[string]interface{}{
							"data": []interface{}{10, 20, 30},
						},
					},
				},
				"chart_type": "bar",
				"format":     "png",
			},
			wantType:   "bar",
			wantFormat: "png",
		},
		{
			name: "line chart svg",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"X", "Y", "Z"},
					"datasets": []interface{}{
						map[string]interface{}{
							"data": []interface{}{5, 15, 25},
						},
					},
				},
				"chart_type": "line",
				"format":     "svg",
			},
			wantType:   "line",
			wantFormat: "svg",
		},
		{
			name: "pie chart png",
			params: map[string]interface{}{
				"data": map[string]interface{}{
					"labels": []interface{}{"Red", "Blue"},
					"datasets": []interface{}{
						map[string]interface{}{
							"data": []interface{}{60, 40},
						},
					},
				},
				"chart_type": "pie",
				"format":     "png",
			},
			wantType:   "pie",
			wantFormat: "png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExportChart(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.True(t, result["success"].(bool))
			assert.Equal(t, tt.wantType, result["chart_type"])
			assert.Equal(t, tt.wantFormat, result["format"])
			assert.NotEmpty(t, result["filename"])
			assert.NotEmpty(t, result["path"])
			assert.NotEmpty(t, result["content"])
			assert.Greater(t, result["size"].(int), 0)

			path := result["path"].(string)
			defer os.Remove(path)

			_, err = os.Stat(path)
			assert.NoError(t, err, "exported file should exist")
		})
	}
}

func TestGenerateImage_Success(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantPrompt  string
		wantStyle   string
		wantWidth   int
		wantHeight  int
		wantSuccess bool
	}{
		{
			name: "simple prompt with defaults",
			params: map[string]interface{}{
				"prompt": "A beautiful sunset over mountains",
			},
			wantPrompt:  "A beautiful sunset over mountains",
			wantStyle:   "realistic",
			wantWidth:   1024,
			wantHeight:  1024,
			wantSuccess: true,
		},
		{
			name: "with custom style and dimensions",
			params: map[string]interface{}{
				"prompt": "Abstract geometric shapes",
				"style":  "abstract",
				"width":  512,
				"height": 768,
			},
			wantPrompt:  "Abstract geometric shapes",
			wantStyle:   "abstract",
			wantWidth:   512,
			wantHeight:  768,
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateImage(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantSuccess, result["success"])
			assert.Equal(t, tt.wantPrompt, result["prompt"])
			assert.Equal(t, tt.wantStyle, result["style"])
			assert.Equal(t, tt.wantWidth, result["width"])
			assert.Equal(t, tt.wantHeight, result["height"])
			assert.NotEmpty(t, result["image_id"])

			metadata, ok := result["metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, metadata["image_id"])
			assert.Equal(t, "pending", metadata["status"])
		})
	}
}

func TestGenerateImage_Validation(t *testing.T) {
	ctx := context.Background()

	result, err := GenerateImage(ctx, map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "prompt is required")
}

func TestVisualTools_ToolRegistration(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
	}{
		{"prepare_diagram registered", "prepare_diagram"},
		{"prepare_chart registered", "prepare_chart"},
		{"export_diagram registered", "export_diagram"},
		{"export_chart registered", "export_chart"},
		{"generate_image registered", "generate_image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := GetTool(tt.toolName)
			require.NoError(t, err)
			assert.NotNil(t, tool)
			assert.Equal(t, tt.toolName, tool.Name)
			assert.NotEmpty(t, tool.Description)
			assert.NotNil(t, tool.Parameters)
			assert.NotNil(t, tool.Function)
		})
	}
}

func TestPrepareDiagram_Integration(t *testing.T) {
	ctx := context.Background()

	mermaidSyntax := `graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Failure]
    C --> E[End]
    D --> E`

	params := map[string]interface{}{
		"syntax": mermaidSyntax,
		"type":   "mermaid",
	}

	result, err := InvokeTool(ctx, "prepare_diagram", params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result["success"].(bool))
	assert.Equal(t, mermaidSyntax, result["syntax"])
	assert.Equal(t, "mermaid", result["type"])
	assert.True(t, result["validated"].(bool))
}

func TestPrepareChart_Integration(t *testing.T) {
	ctx := context.Background()

	params := map[string]interface{}{
		"data": map[string]interface{}{
			"labels": []interface{}{"January", "February", "March", "April", "May"},
			"datasets": []interface{}{
				map[string]interface{}{
					"label":           "Sales",
					"data":            []interface{}{65, 59, 80, 81, 56},
					"backgroundColor": "rgba(75, 192, 192, 0.2)",
					"borderColor":     "rgba(75, 192, 192, 1)",
				},
			},
		},
		"chart_type": "line",
		"options": map[string]interface{}{
			"responsive": true,
			"title": map[string]interface{}{
				"display": true,
				"text":    "Monthly Sales Data",
			},
		},
	}

	result, err := InvokeTool(ctx, "prepare_chart", params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result["success"].(bool))
	assert.Equal(t, "line", result["chart_type"])
	assert.NotEmpty(t, result["chart_id"])

	data, ok := result["data"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, data["labels"])
	assert.NotNil(t, data["datasets"])
}

func TestGenerateImage_Integration(t *testing.T) {
	ctx := context.Background()

	params := map[string]interface{}{
		"prompt": "A serene landscape with mountains and a lake at sunset",
		"style":  "realistic",
		"width":  1024,
		"height": 1024,
	}

	result, err := InvokeTool(ctx, "generate_image", params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result["success"].(bool))
	assert.Equal(t, params["prompt"], result["prompt"])
	assert.Equal(t, params["style"], result["style"])
	assert.NotEmpty(t, result["image_id"])
	assert.NotEmpty(t, result["note"])
}
