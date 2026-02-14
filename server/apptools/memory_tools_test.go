package apptools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

func setupMemoryToolsTest(t *testing.T) (*memory.MemoryManager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "memory_tools_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	mm, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create memory manager: %v", err)
	}

	InitializeMemoryTools(mm)

	// Reset the rate limiter so tests don't interfere with each other
	searchRateLimiter.mu.Lock()
	searchRateLimiter.requests = make(map[string][]time.Time)
	searchRateLimiter.mu.Unlock()

	cleanup := func() {
		mm.Close()
		os.RemoveAll(tmpDir)
		memoryManager = nil
	}

	return mm, cleanup
}

func TestMemorySearch_ToolRegistration(t *testing.T) {
	tool, err := tools.GetTool("memory_search")
	if err != nil {
		t.Fatalf("memory_search tool not registered: %v", err)
	}

	if tool.Name != "memory_search" {
		t.Errorf("expected tool name 'memory_search', got '%s'", tool.Name)
	}

	if tool.Function == nil {
		t.Error("tool function is nil")
	}

	params, ok := tool.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("parameters.properties not found")
	}

	if _, ok := params["query"]; !ok {
		t.Error("query parameter not defined")
	}

	if _, ok := params["tier"]; !ok {
		t.Error("tier parameter not defined")
	}

	if _, ok := params["max_results"]; !ok {
		t.Error("max_results parameter not defined")
	}

	if _, ok := params["context_type"]; !ok {
		t.Error("context_type parameter not defined")
	}

	required, ok := tool.Parameters["required"].([]string)
	if !ok {
		t.Fatal("required parameters not defined")
	}

	if len(required) != 1 || required[0] != "query" {
		t.Errorf("expected required parameters ['query'], got %v", required)
	}
}

func TestMemoryGet_ToolRegistration(t *testing.T) {
	tool, err := tools.GetTool("memory_get")
	if err != nil {
		t.Fatalf("memory_get tool not registered: %v", err)
	}

	if tool.Name != "memory_get" {
		t.Errorf("expected tool name 'memory_get', got '%s'", tool.Name)
	}

	if tool.Function == nil {
		t.Error("tool function is nil")
	}

	params, ok := tool.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("parameters.properties not found")
	}

	if _, ok := params["path"]; !ok {
		t.Error("path parameter not defined")
	}

	if _, ok := params["start_line"]; !ok {
		t.Error("start_line parameter not defined")
	}

	if _, ok := params["line_count"]; !ok {
		t.Error("line_count parameter not defined")
	}

	required, ok := tool.Parameters["required"].([]string)
	if !ok {
		t.Fatal("required parameters not defined")
	}

	if len(required) != 1 || required[0] != "path" {
		t.Errorf("expected required parameters ['path'], got %v", required)
	}
}

func TestMemorySearch_NotInitialized(t *testing.T) {
	originalMM := memoryManager
	memoryManager = nil
	defer func() {
		memoryManager = originalMM
	}()

	ctx := context.Background()
	params := map[string]interface{}{
		"query": "test query",
	}

	result, err := MemorySearch(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false when manager not initialized")
	}

	_, ok = result["error"].(string)
	if !ok {
		t.Error("expected error message when manager not initialized")
	}
}

func TestMemorySearch_MissingQuery(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{}

	result, err := MemorySearch(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false when query is missing")
	}

	_, ok = result["error"].(string)
	if !ok {
		t.Error("expected error message")
	}
}

func TestMemorySearch_EmptyQuery(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"query": "",
	}

	result, err := MemorySearch(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false when query is empty")
	}
}

func TestMemorySearch_InvalidTier(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testCases := []int{-1, 4, 5, 100}
	for _, tier := range testCases {
		params := map[string]interface{}{
			"query": "test",
			"tier":  tier,
		}

		result, err := MemorySearch(ctx, params)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || success {
			t.Errorf("expected success=false for tier %d", tier)
		}
	}
}

func TestMemorySearch_ValidTiers(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testCases := []int{0, 1, 2, 3}
	for _, tier := range testCases {
		params := map[string]interface{}{
			"query": "test",
			"tier":  tier,
		}

		result, err := MemorySearch(ctx, params)
		if err != nil {
			t.Fatalf("expected no error for tier %d, got: %v", tier, err)
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			t.Errorf("expected success=true for tier %d, got: %v", tier, result)
		}
	}
}

func TestMemorySearch_MaxResultsValidation(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testCases := []struct {
		maxResults int
		expected   int
	}{
		{0, 10},
		{-5, 10},
		{50, 50},
		{150, 100},
	}

	for _, tc := range testCases {
		params := map[string]interface{}{
			"query":       "test",
			"max_results": tc.maxResults,
		}

		result, err := MemorySearch(ctx, params)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			t.Errorf("expected success=true, got: %v", result)
		}

		maxResults, ok := result["max_results"].(int)
		if !ok {
			t.Errorf("max_results not in result")
			continue
		}

		if maxResults != tc.expected {
			t.Errorf("for max_results=%d, expected %d, got %d", tc.maxResults, tc.expected, maxResults)
		}
	}
}

func TestMemorySearch_WithData(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testMemories := []memory.Memory{
		{
			ID:          "mem1",
			Type:        "tier_1",
			Content:     "This is about artificial intelligence and machine learning",
			ContextType: "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    map[string]interface{}{"session": "test1"},
		},
		{
			ID:          "mem2",
			Type:        "tier_2",
			Content:     "Data science is a multidisciplinary field",
			ContextType: "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    map[string]interface{}{"session": "test2"},
		},
		{
			ID:          "mem3",
			Type:        "tier_1",
			Content:     "Deep learning neural networks are powerful",
			ContextType: "group",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    map[string]interface{}{"session": "test3"},
		},
	}

	for _, mem := range testMemories {
		if err := mm.Store(ctx, mem); err != nil {
			t.Fatalf("failed to store memory: %v", err)
		}
	}

	params := map[string]interface{}{
		"query":       "machine learning",
		"max_results": 5,
	}

	result, err := MemorySearch(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got: %v", result)
	}

	count, ok := result["count"].(int)
	if !ok {
		t.Error("count not in result")
	} else if count < 0 || count > 3 {
		t.Errorf("expected count between 0 and 3, got %d", count)
	}

	results, ok := result["results"].([]map[string]interface{})
	if !ok {
		t.Error("results not in expected format")
	} else if len(results) != count {
		t.Errorf("results length %d doesn't match count %d", len(results), count)
	}

	if len(results) > 0 {
		firstResult := results[0]
		if _, ok := firstResult["id"]; !ok {
			t.Error("result missing 'id' field")
		}
		if _, ok := firstResult["content"]; !ok {
			t.Error("result missing 'content' field")
		}
		if _, ok := firstResult["similarity"]; !ok {
			t.Error("result missing 'similarity' field")
		}
		if _, ok := firstResult["snippet"]; !ok {
			t.Error("result missing 'snippet' field")
		}
	}
}

func TestMemorySearch_ContextTypeFilter(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testMemories := []memory.Memory{
		{
			ID:          "mem1",
			Type:        "tier_1",
			Content:     "Private memory content",
			ContextType: "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    map[string]interface{}{},
		},
		{
			ID:          "mem2",
			Type:        "tier_1",
			Content:     "Group memory content",
			ContextType: "group",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    map[string]interface{}{},
		},
	}

	for _, mem := range testMemories {
		if err := mm.Store(ctx, mem); err != nil {
			t.Fatalf("failed to store memory: %v", err)
		}
	}

	params := map[string]interface{}{
		"query":        "memory",
		"context_type": "private",
		"max_results":  10,
	}

	result, err := MemorySearch(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got: %v", result)
	}
}

func TestMemoryGet_NotInitialized(t *testing.T) {
	originalMM := memoryManager
	memoryManager = nil
	defer func() {
		memoryManager = originalMM
	}()

	ctx := context.Background()
	params := map[string]interface{}{
		"path": "test-id",
	}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false when manager not initialized")
	}

	_, ok = result["error"].(string)
	if !ok {
		t.Error("expected error message when manager not initialized")
	}
}

func TestMemoryGet_MissingPath(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false when path is missing")
	}
}

func TestMemoryGet_EmptyPath(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"path": "",
	}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false when path is empty")
	}
}

func TestMemoryGet_ValidPath(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	mem := memory.Memory{
		ID:          "test-mem",
		Type:        "tier_1",
		Content:     testContent,
		ContextType: "private",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{},
	}

	if err := mm.Store(ctx, mem); err != nil {
		t.Fatalf("failed to store memory: %v", err)
	}

	params := map[string]interface{}{
		"path": "test-mem",
	}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got: %v", result)
	}

	content, ok := result["content"].(string)
	if !ok {
		t.Error("content not in result")
	} else if content != testContent {
		t.Errorf("expected content '%s', got '%s'", testContent, content)
	}

	// metadata is the raw metadata map stored with the memory (may be empty)
	_, ok = result["metadata"].(map[string]interface{})
	if !ok {
		// metadata might be nil if an empty map was stored
		if result["metadata"] != nil {
			t.Error("metadata not in result")
		}
	}
}

func TestMemoryGet_LineRange(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"
	mem := memory.Memory{
		ID:          "test-mem",
		Type:        "tier_1",
		Content:     testContent,
		ContextType: "private",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{},
	}

	if err := mm.Store(ctx, mem); err != nil {
		t.Fatalf("failed to store memory: %v", err)
	}

	params := map[string]interface{}{
		"path":       "test-mem",
		"start_line": 2,
		"line_count": 3,
	}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got: %v", result)
	}

	content, ok := result["content"].(string)
	if !ok {
		t.Error("content not in result")
	} else {
		expected := "Line 3\nLine 4\nLine 5"
		if content != expected {
			t.Errorf("expected content '%s', got '%s'", expected, content)
		}
	}
}

func TestMemoryGet_InvalidLineParams(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()

	testContent := "Line 1\nLine 2\nLine 3"
	mem := memory.Memory{
		ID:          "test-mem",
		Type:        "tier_1",
		Content:     testContent,
		ContextType: "private",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{},
	}

	if err := mm.Store(ctx, mem); err != nil {
		t.Fatalf("failed to store memory: %v", err)
	}

	testCases := []struct {
		name       string
		startLine  int
		lineCount  int
		shouldPass bool
	}{
		{"negative start", -5, 10, true},
		{"zero line count", 0, 0, true},
		{"negative line count", 0, -5, true},
		{"large line count", 0, 2000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]interface{}{
				"path":       "test-mem",
				"start_line": tc.startLine,
				"line_count": tc.lineCount,
			}

			result, err := MemoryGet(ctx, params)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Error("success not in result")
			}

			if tc.shouldPass && !success {
				t.Errorf("expected success for %s", tc.name)
			}
		})
	}
}

func TestMemoryGet_NonexistentPath(t *testing.T) {
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"path": "nonexistent-id",
	}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || success {
		t.Error("expected success=false for nonexistent path")
	}

	errorMsg, ok := result["error"].(string)
	if !ok {
		t.Error("expected error message for nonexistent path")
	} else if errorMsg == "" {
		t.Error("error message is empty")
	}
}

func TestMemoryGet_FileBasedPath(t *testing.T) {
	// NOTE: GetLines currently only supports memory IDs (database lookup),
	// not file-system paths. This test verifies that a file path that doesn't
	// correspond to a stored memory ID returns success=false.
	mm, cleanup := setupMemoryToolsTest(t)
	defer cleanup()

	if mm == nil {
		t.Fatal("memory manager is nil")
	}

	tmpDir, err := os.MkdirTemp("", "memory_get_file_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "File Line 1\nFile Line 2\nFile Line 3\nFile Line 4"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"path":       testFile,
		"start_line": 1,
		"line_count": 2,
	}

	result, err := MemoryGet(ctx, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// File-based paths are not supported by GetLines; it only queries by memory ID.
	// The call should return success=false with an error message.
	success, ok := result["success"].(bool)
	if !ok || success {
		t.Errorf("expected success=false for file path (not a stored memory ID), got: %v", result)
	}

	errorMsg, ok := result["error"].(string)
	if !ok || errorMsg == "" {
		t.Error("expected error message for file path lookup")
	}
}
