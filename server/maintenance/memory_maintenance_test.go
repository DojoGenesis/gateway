package maintenance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPluginManager struct {
	prov provider.ModelProvider
}

func (m *mockPluginManager) GetProvider(name string) (interface{}, error) {
	if m.prov != nil {
		return m.prov, nil
	}
	return nil, fmt.Errorf("provider not found: %s", name)
}

func (m *mockPluginManager) GetProviders() map[string]interface{} {
	if m.prov != nil {
		return map[string]interface{}{"ollama": m.prov}
	}
	return map[string]interface{}{}
}

type mockProvider struct {
	completionResponse *provider.CompletionResponse
	embeddingResponse  []float32
	completionError    error
	embeddingError     error
}

func (m *mockProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if m.completionError != nil {
		return nil, m.completionError
	}
	if m.completionResponse != nil {
		return m.completionResponse, nil
	}
	return &provider.CompletionResponse{
		Content: "INSIGHT: Test insight\nWHY: This is important\n---\nINSIGHT: Another insight\nWHY: This matters too\n---",
		Model:   "ollama",
	}, nil
}

func (m *mockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.embeddingError != nil {
		return nil, m.embeddingError
	}
	if m.embeddingResponse != nil {
		return m.embeddingResponse, nil
	}
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

func (m *mockProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:    "Mock Provider",
		Version: "1.0.0",
	}, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:   "ollama",
			Name: "Qwen3 Embedded",
		},
	}, nil
}

func (m *mockProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	return nil, fmt.Errorf("streaming not supported in mock")
}

func (m *mockProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return nil, fmt.Errorf("tool calls not supported in mock")
}

func setupTestMaintenance(t *testing.T) (*MemoryMaintenance, string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "maintenance_test")
	require.NoError(t, err)

	memoryDir := filepath.Join(tempDir, "memory")
	err = os.MkdirAll(memoryDir, 0755)
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "test.db")
	memoryManager, err := memory.NewMemoryManager(dbPath)
	require.NoError(t, err)

	mockPM := &mockPluginManager{
		prov: &mockProvider{},
	}

	mm, err := NewMemoryMaintenance(memoryManager, mockPM, memoryDir)
	require.NoError(t, err)

	cleanup := func() {
		memoryManager.Close()
		os.RemoveAll(tempDir)
	}

	return mm, memoryDir, cleanup
}

func TestNewMemoryMaintenance(t *testing.T) {
	tests := []struct {
		name        string
		memoryMgr   *memory.MemoryManager
		pluginMgr   memory.PluginManagerInterface
		memoryDir   string
		expectError bool
	}{
		{
			name:        "nil memory manager",
			memoryMgr:   nil,
			pluginMgr:   &mockPluginManager{},
			memoryDir:   "memory",
			expectError: true,
		},
		{
			name:        "nil plugin manager",
			memoryMgr:   &memory.MemoryManager{},
			pluginMgr:   nil,
			memoryDir:   "memory",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm, err := NewMemoryMaintenance(tt.memoryMgr, tt.pluginMgr, tt.memoryDir)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, mm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mm)
			}
		})
	}
}

func TestScanDailyFiles(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	now := time.Now()
	oldDate := now.AddDate(0, 0, -5)
	recentDate := now.AddDate(0, 0, -1)

	files := map[string]string{
		oldDate.Format("2006-01-02") + ".md":    "Old daily note",
		recentDate.Format("2006-01-02") + ".md": "Recent daily note",
		"MEMORY.md":                             "Main memory file",
		"not-a-date.md":                         "Invalid name",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(memoryDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	ctx := context.Background()
	dailyFiles, err := mm.scanDailyFiles(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(dailyFiles), "Should only find old daily files (>3 days)")

	expectedOldFile := filepath.Join(memoryDir, oldDate.Format("2006-01-02")+".md")
	assert.Contains(t, dailyFiles, expectedOldFile)
}

func TestIdentifyInsights(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	testFile := filepath.Join(memoryDir, "2026-01-01.md")
	testContent := `# Daily Note 2026-01-01

## Session 1
User: How do I implement authentication?
Agent: You should use JWT tokens with refresh token rotation...

Key decision: Using JWT with refresh tokens for security.
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	insights, err := mm.IdentifyInsights(ctx, testFile)

	assert.NoError(t, err)
	assert.Greater(t, len(insights), 0, "Should extract at least one insight")

	for _, insight := range insights {
		assert.NotEmpty(t, insight.Summary)
		assert.NotEmpty(t, insight.Source)
		assert.Equal(t, "2026-01-01.md", insight.Source)
	}
}

func TestIdentifyInsights_EmptyFile(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	testFile := filepath.Join(memoryDir, "empty.md")
	err := os.WriteFile(testFile, []byte(""), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	insights, err := mm.IdentifyInsights(ctx, testFile)

	assert.NoError(t, err)
	assert.Empty(t, insights, "Empty file should produce no insights")
}

func TestAppendToMemory(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	insights := []Insight{
		{
			Theme:       "authentication",
			Summary:     "Implement JWT with refresh tokens",
			Source:      "2026-01-01.md",
			Importance:  0.8,
			ExtractedAt: time.Now(),
		},
		{
			Theme:       "database",
			Summary:     "Use connection pooling for better performance",
			Source:      "2026-01-02.md",
			Importance:  0.7,
			ExtractedAt: time.Now(),
		},
	}

	ctx := context.Background()
	err := mm.AppendToMemory(ctx, insights)

	assert.NoError(t, err)

	content, err := os.ReadFile(mm.memoryFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "Memory Maintenance")
	assert.Contains(t, contentStr, "Implement JWT with refresh tokens")
	assert.Contains(t, contentStr, "Use connection pooling")
	assert.Contains(t, contentStr, "(2026-01-01.md)")
	assert.Contains(t, contentStr, "(2026-01-02.md)")
}

func TestAppendToMemory_ExistingFile(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	existingContent := "# Memory\n\nExisting content here.\n"
	err := os.WriteFile(mm.memoryFile, []byte(existingContent), 0644)
	require.NoError(t, err)

	insights := []Insight{
		{
			Theme:   "test",
			Summary: "New insight",
			Source:  "test.md",
		},
	}

	ctx := context.Background()
	err = mm.AppendToMemory(ctx, insights)

	assert.NoError(t, err)

	content, err := os.ReadFile(mm.memoryFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "Existing content here")
	assert.Contains(t, contentStr, "New insight")
}

func TestArchiveFile(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	testFile := filepath.Join(memoryDir, "2026-01-01.md")
	testContent := "# Daily Note\n\nThis is test content that should be archived."
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = mm.ArchiveFile(ctx, testFile)

	assert.NoError(t, err)

	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err), "Original file should be deleted")

	archivePath := filepath.Join(mm.archiveDir, "2026-01-01.md")
	_, err = os.Stat(archivePath)
	assert.NoError(t, err, "Archive file should exist")

	archivedContent, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.NotEmpty(t, string(archivedContent), "Archived content should not be empty")
}

func TestCompostMemory(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	oldDate := time.Now().AddDate(0, -7, 0).Format("2006-01-02")
	recentDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")

	memoryContent := fmt.Sprintf(`# Memory

## Memory Maintenance - %s

- Old insight that should be composted

## Memory Maintenance - %s

- Recent insight that should be kept
`, oldDate, recentDate)

	err := os.WriteFile(mm.memoryFile, []byte(memoryContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	composted, err := mm.CompostMemory(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, composted, "Should compost one old entry")

	content, err := os.ReadFile(mm.memoryFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.NotContains(t, contentStr, "Old insight")
	assert.Contains(t, contentStr, "Recent insight")
}

func TestCompostMemory_NoFile(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	ctx := context.Background()
	composted, err := mm.CompostMemory(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0, composted, "Should compost nothing when no file exists")
}

func TestRunMaintenance_Integration(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	oldDate := time.Now().AddDate(0, 0, -5)
	oldFile := filepath.Join(memoryDir, oldDate.Format("2006-01-02")+".md")
	oldContent := `# Daily Note

User: How do I optimize my database queries?
Agent: Use indexes and connection pooling for better performance.

Key insight: Database optimization is critical for performance.
`
	err := os.WriteFile(oldFile, []byte(oldContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.True(t, report.Success || len(report.Errors) > 0)
	assert.Equal(t, 1, report.FilesProcessed)

	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err), "Old file should be archived and removed")

	archivePath := filepath.Join(mm.archiveDir, oldDate.Format("2006-01-02")+".md")
	_, err = os.Stat(archivePath)
	assert.NoError(t, err, "Archive should exist")
}

func TestRunMaintenance_EmptyDirectory(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.True(t, report.Success)
	assert.Equal(t, 0, report.FilesProcessed)
	assert.Equal(t, 0, report.FilesArchived)
	assert.Equal(t, 0, report.InsightsExtracted)
}

func TestParseInsights(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	response := `INSIGHT: Use JWT tokens for authentication
WHY: Provides stateless auth and better scalability
---
INSIGHT: Implement rate limiting
WHY: Protects against abuse and DoS attacks
---`

	themes := []string{"authentication", "security"}
	insights := mm.parseInsights(response, themes, "/path/to/2026-01-01.md")

	assert.Len(t, insights, 2)
	assert.Equal(t, "Use JWT tokens for authentication", insights[0].Summary)
	assert.Equal(t, "Implement rate limiting", insights[1].Summary)
	assert.Equal(t, "authentication", insights[0].Theme)
	assert.Equal(t, "2026-01-01.md", insights[0].Source)
}

func TestParseInsights_InvalidFormat(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	response := `Some random text without proper format
More random text
---
INSIGHT: Only summary, no WHY
---`

	themes := []string{"test"}
	insights := mm.parseInsights(response, themes, "test.md")

	assert.Empty(t, insights, "Should not parse insights with invalid format")
}

func TestBuildInsightExtractionPrompt(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	content := "Test daily note content"
	themes := []string{"authentication", "database"}

	prompt := mm.buildInsightExtractionPrompt(content, themes)

	assert.Contains(t, prompt, "authentication, database")
	assert.Contains(t, prompt, "Test daily note content")
	assert.Contains(t, prompt, "INSIGHT:")
	assert.Contains(t, prompt, "WHY:")
	assert.Contains(t, prompt, "3+ months")
}

func TestGetMemorySize(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	files := map[string]string{
		"test1.md": strings.Repeat("a", 1000),
		"test2.md": strings.Repeat("b", 2000),
		"test.txt": "should not count",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(memoryDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	size, err := mm.getMemorySize()

	assert.NoError(t, err)
	assert.Equal(t, int64(3000), size, "Should only count .md files")
}

func TestNewMemoryMaintenance_EmptyDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	memoryManager, err := memory.NewMemoryManager(dbPath)
	require.NoError(t, err)
	defer memoryManager.Close()

	mockPM := &mockPluginManager{
		prov: &mockProvider{},
	}

	mm, err := NewMemoryMaintenance(memoryManager, mockPM, "")

	assert.NoError(t, err)
	assert.NotNil(t, mm)
	assert.Equal(t, "memory", mm.memoryDir)
}

func TestScanDailyFiles_ErrorReadingDir(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	mm.memoryDir = "/nonexistent/path"

	ctx := context.Background()
	_, err := mm.scanDailyFiles(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read memory directory")
}

func TestIdentifyInsights_ErrorReadingFile(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	ctx := context.Background()
	_, err := mm.IdentifyInsights(ctx, "/nonexistent/file.md")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestIdentifyInsights_ErrorGettingProvider(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	mm.pluginManager = &mockPluginManager{}

	testFile := filepath.Join(memoryDir, "test.md")
	testContent := strings.Repeat("Test content for insights extraction. ", 10)
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = mm.IdentifyInsights(ctx, testFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get provider")
}

func TestAppendToMemory_ErrorReadingExisting(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	err := os.WriteFile(mm.memoryFile, []byte("test"), 0000)
	require.NoError(t, err)

	insights := []Insight{
		{Summary: "test", Source: "test.md"},
	}

	ctx := context.Background()
	err = mm.AppendToMemory(ctx, insights)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read existing MEMORY.md")
}

func TestAppendToMemory_ErrorWriting(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	err := os.WriteFile(mm.memoryFile, []byte("existing"), 0644)
	require.NoError(t, err)
	err = os.Chmod(filepath.Dir(mm.memoryFile), 0444)
	require.NoError(t, err)
	defer os.Chmod(filepath.Dir(mm.memoryFile), 0755)

	insights := []Insight{
		{Summary: "test", Source: "test.md"},
	}

	ctx := context.Background()
	err = mm.AppendToMemory(ctx, insights)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write MEMORY.md")
}

func TestArchiveFile_ErrorReadingFile(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	ctx := context.Background()
	err := mm.ArchiveFile(ctx, "/nonexistent/file.md")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestArchiveFile_ErrorWritingArchive(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	testFile := filepath.Join(memoryDir, "test.md")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = os.RemoveAll(mm.archiveDir)
	require.NoError(t, err)
	err = os.WriteFile(mm.archiveDir, []byte("file instead of dir"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = mm.ArchiveFile(ctx, testFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write archive file")
}

func TestCompostMemory_ErrorReading(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	err := os.WriteFile(mm.memoryFile, []byte("test"), 0000)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = mm.CompostMemory(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read MEMORY.md")
}

func TestRunMaintenance_ErrorScanningFiles(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	mm.memoryDir = "/nonexistent/path"

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.Error(t, err)
	assert.NotNil(t, report)
	assert.False(t, report.Success)
	assert.Greater(t, len(report.Errors), 0)
}

func TestRunMaintenance_MultipleErrors(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	oldDate := time.Now().AddDate(0, 0, -5)
	oldFile := filepath.Join(memoryDir, oldDate.Format("2006-01-02")+".md")
	err := os.WriteFile(oldFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = os.RemoveAll(mm.archiveDir)
	require.NoError(t, err)
	err = os.WriteFile(mm.archiveDir, []byte("block archive"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, len(report.Errors), 0)
}

func TestFinalizeReport_WithErrors(t *testing.T) {
	report := &MaintenanceReport{
		StartTime: time.Now(),
		Errors:    []string{"error 1", "error 2"},
		Success:   true,
	}

	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	mm.finalizeReport(report)

	assert.False(t, report.Success)
	assert.Greater(t, report.Duration, time.Duration(0))
}

func TestRunMaintenance_ConcurrentSafety(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	ctx := context.Background()
	results := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			_, err := mm.RunMaintenance(ctx)
			results <- err
		}()
	}

	for i := 0; i < 3; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

func TestIdentifyInsights_ErrorGeneratingCompletion(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	mm.pluginManager = &mockPluginManager{
		prov: &mockProvider{
			completionError: fmt.Errorf("completion error"),
		},
	}

	testFile := filepath.Join(memoryDir, "test.md")
	testContent := strings.Repeat("Test content for insights extraction. ", 10)
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = mm.IdentifyInsights(ctx, testFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate insights")
}

func TestArchiveFile_ErrorRemovingOriginal(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	testFile := filepath.Join(memoryDir, "test.md")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = os.Chmod(testFile, 0444)
	require.NoError(t, err)

	err = os.Chmod(memoryDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(memoryDir, 0755)

	ctx := context.Background()
	err = mm.ArchiveFile(ctx, testFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove original file")
}

func TestRunMaintenance_ErrorGettingInitialSize(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	testSubDir := filepath.Join(memoryDir, "subdir")
	err := os.MkdirAll(testSubDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(testSubDir, "test.md")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	err = os.Chmod(testSubDir, 0000)
	require.NoError(t, err)
	defer os.Chmod(testSubDir, 0755)

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, len(report.Errors), 0)
}

func TestRunMaintenance_SuccessfulComposting(t *testing.T) {
	mm, _, cleanup := setupTestMaintenance(t)
	defer cleanup()

	oldDate := time.Now().AddDate(0, -7, 0)
	memoryContent := fmt.Sprintf(`# Memory

## Memory Maintenance - %s

- Old insight that should be composted
`, oldDate.Format("2006-01-02"))

	err := os.WriteFile(mm.memoryFile, []byte(memoryContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 1, report.EntriesComposted)
}

func TestRunMaintenance_WithInsights(t *testing.T) {
	mm, memoryDir, cleanup := setupTestMaintenance(t)
	defer cleanup()

	oldDate := time.Now().AddDate(0, 0, -5)
	oldFile := filepath.Join(memoryDir, oldDate.Format("2006-01-02")+".md")
	content := strings.Repeat("Important content about database optimization strategies. ", 10)
	err := os.WriteFile(oldFile, []byte(content), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	report, err := mm.RunMaintenance(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, report.InsightsExtracted, 0)
	assert.Equal(t, 1, report.FilesArchived)
}
