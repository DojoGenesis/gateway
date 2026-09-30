package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleSearchTools(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/tools/search", HandleSearchTools)

	tests := []struct {
		name           string
		requestBody    SearchToolsRequest
		expectedStatus int
		minCount       int
	}{
		{
			name: "search by query - file",
			requestBody: SearchToolsRequest{
				Query: "file",
			},
			expectedStatus: http.StatusOK,
			minCount:       2,
		},
		{
			name: "search by category - planning",
			requestBody: SearchToolsRequest{
				Category: "planning",
			},
			expectedStatus: http.StatusOK,
			minCount:       5,
		},
		{
			name: "search with max results",
			requestBody: SearchToolsRequest{
				MaxResults: 5,
			},
			expectedStatus: http.StatusOK,
			minCount:       5,
		},
		{
			name:           "search all tools",
			requestBody:    SearchToolsRequest{},
			expectedStatus: http.StatusOK,
			minCount:       25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/tools/search", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.Code)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			count, _ := result["count"].(float64)
			if int(count) < tt.minCount {
				t.Errorf("Expected at least %d tools, got %v", tt.minCount, count)
			}
		})
	}
}

func TestHandleInvokeTool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/tools/invoke", HandleInvokeTool)

	tests := []struct {
		name           string
		requestBody    InvokeToolRequest
		expectedStatus int
		wantSuccess    bool
		checkResult    func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "invoke calculate tool",
			requestBody: InvokeToolRequest{
				ToolName: "calculate",
				Params: map[string]interface{}{
					"expression": "2 + 2",
				},
			},
			expectedStatus: http.StatusOK,
			wantSuccess:    true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				if result["result"] == nil {
					t.Error("Expected result field")
				}
			},
		},
		{
			name: "invoke search_tools",
			requestBody: InvokeToolRequest{
				ToolName: "search_tools",
				Params: map[string]interface{}{
					"query": "file",
				},
			},
			expectedStatus: http.StatusOK,
			wantSuccess:    true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				if result["result"] == nil {
					t.Error("Expected result field")
				}
			},
		},
		{
			name: "invoke non-existent tool",
			requestBody: InvokeToolRequest{
				ToolName: "non_existent_tool",
				Params:   map[string]interface{}{},
			},
			expectedStatus: http.StatusInternalServerError,
			wantSuccess:    false,
			checkResult:    nil,
		},
		{
			name: "missing tool_name",
			requestBody: InvokeToolRequest{
				Params: map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
			wantSuccess:    false,
			checkResult:    nil,
		},
		{
			name: "invoke with session_id",
			requestBody: InvokeToolRequest{
				ToolName:  "calculate",
				Params:    map[string]interface{}{"expression": "10 * 5"},
				SessionID: "test-session-123",
			},
			expectedStatus: http.StatusOK,
			wantSuccess:    true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				if result["result"] == nil {
					t.Error("Expected result field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/tools/invoke", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.Code, resp.Body.String())
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("Expected success: %v, got %v. Error: %v", tt.wantSuccess, success, result["error"])
			}

			if tt.checkResult != nil && tt.wantSuccess {
				tt.checkResult(t, result)
			}
		})
	}
}
