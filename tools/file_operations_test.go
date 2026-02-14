package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile(t *testing.T) {
	ctx := context.Background()

	t.Run("successful read", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!\nLine 2\nLine 3"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ReadFile(ctx, map[string]interface{}{
			"file_path": filePath,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["content"].(string) != content {
			t.Errorf("expected content=%q, got %q", content, result["content"])
		}

		if result["lines"].(int) != 3 {
			t.Errorf("expected lines=3, got %d", result["lines"])
		}

		if result["encoding"].(string) != "utf-8" {
			t.Errorf("expected encoding=utf-8, got %s", result["encoding"])
		}
	})

	t.Run("file not found", func(t *testing.T) {
		result, err := ReadFile(ctx, map[string]interface{}{
			"file_path": "/nonexistent/file.txt",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for nonexistent file")
		}

		if !strings.Contains(result["error"].(string), "not found") {
			t.Errorf("expected 'not found' error, got: %s", result["error"])
		}
	})

	t.Run("path is directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := ReadFile(ctx, map[string]interface{}{
			"file_path": tmpDir,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for directory")
		}

		if !strings.Contains(result["error"].(string), "not a file") {
			t.Errorf("expected 'not a file' error, got: %s", result["error"])
		}
	})

	t.Run("file too large", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "large.txt")
		content := strings.Repeat("a", 2000)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ReadFile(ctx, map[string]interface{}{
			"file_path": filePath,
			"max_bytes": 1000,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for large file")
		}

		if !strings.Contains(result["error"].(string), "too large") {
			t.Errorf("expected 'too large' error, got: %s", result["error"])
		}
	})

	t.Run("missing file_path parameter", func(t *testing.T) {
		result, err := ReadFile(ctx, map[string]interface{}{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for missing parameter")
		}

		if !strings.Contains(result["error"].(string), "required") {
			t.Errorf("expected 'required' error, got: %s", result["error"])
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "empty.txt")
		if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ReadFile(ctx, map[string]interface{}{
			"file_path": filePath,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["content"].(string) != "" {
			t.Errorf("expected empty content, got %q", result["content"])
		}

		if result["lines"].(int) != 0 {
			t.Errorf("expected lines=0, got %d", result["lines"])
		}
	})
}

func TestWriteFile(t *testing.T) {
	ctx := context.Background()

	t.Run("successful write new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!\nLine 2"

		result, err := WriteFile(ctx, map[string]interface{}{
			"file_path": filePath,
			"content":   content,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if !result["created"].(bool) {
			t.Errorf("expected created=true for new file")
		}

		if result["lines"].(int) != 2 {
			t.Errorf("expected lines=2, got %d", result["lines"])
		}

		actualContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}

		if string(actualContent) != content {
			t.Errorf("expected content=%q, got %q", content, string(actualContent))
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(filePath, []byte("old content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := WriteFile(ctx, map[string]interface{}{
			"file_path": filePath,
			"content":   "new content",
			"overwrite": true,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["created"].(bool) {
			t.Errorf("expected created=false for existing file")
		}

		actualContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}

		if string(actualContent) != "new content" {
			t.Errorf("expected content='new content', got %q", string(actualContent))
		}
	})

	t.Run("fail on existing file without overwrite", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(filePath, []byte("old content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := WriteFile(ctx, map[string]interface{}{
			"file_path": filePath,
			"content":   "new content",
			"overwrite": false,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for existing file without overwrite")
		}

		if !strings.Contains(result["error"].(string), "already exists") {
			t.Errorf("expected 'already exists' error, got: %s", result["error"])
		}
	})

	t.Run("create directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nested", "dir", "test.txt")

		result, err := WriteFile(ctx, map[string]interface{}{
			"file_path":   filePath,
			"content":     "test",
			"create_dirs": true,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("file should exist at %s", filePath)
		}
	})

	t.Run("missing file_path parameter", func(t *testing.T) {
		result, err := WriteFile(ctx, map[string]interface{}{
			"content": "test",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for missing parameter")
		}

		if !strings.Contains(result["error"].(string), "required") {
			t.Errorf("expected 'required' error, got: %s", result["error"])
		}
	})

	t.Run("missing content parameter", func(t *testing.T) {
		result, err := WriteFile(ctx, map[string]interface{}{
			"file_path": "/tmp/test.txt",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for missing parameter")
		}

		if !strings.Contains(result["error"].(string), "required") {
			t.Errorf("expected 'required' error, got: %s", result["error"])
		}
	})
}

func TestListDirectory(t *testing.T) {
	ctx := context.Background()

	t.Run("list non-recursive", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "file2.py"), []byte("test"), 0644)
		os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
		os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("test"), 0644)

		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": tmpDir,
			"recursive":      false,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		files := result["files"].([]string)
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}

		dirs := result["directories"].([]string)
		if len(dirs) != 1 {
			t.Errorf("expected 1 directory, got %d: %v", len(dirs), dirs)
		}
	})

	t.Run("list recursive", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
		os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
		os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("test"), 0644)

		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": tmpDir,
			"recursive":      true,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		files := result["files"].([]string)
		if len(files) != 2 {
			t.Errorf("expected 2 files in recursive mode, got %d: %v", len(files), files)
		}
	})

	t.Run("filter by file type", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "file2.py"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("test"), 0644)

		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": tmpDir,
			"file_types":     []interface{}{".txt"},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		files := result["files"].([]string)
		if len(files) != 2 {
			t.Errorf("expected 2 .txt files, got %d: %v", len(files), files)
		}
	})

	t.Run("exclude hidden files", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, ".hidden.txt"), []byte("test"), 0644)

		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": tmpDir,
			"include_hidden": false,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		files := result["files"].([]string)
		if len(files) != 1 {
			t.Errorf("expected 1 visible file, got %d: %v", len(files), files)
		}
	})

	t.Run("include hidden files", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, ".hidden.txt"), []byte("test"), 0644)

		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": tmpDir,
			"include_hidden": true,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		files := result["files"].([]string)
		if len(files) != 2 {
			t.Errorf("expected 2 files (including hidden), got %d: %v", len(files), files)
		}
	})

	t.Run("directory not found", func(t *testing.T) {
		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": "/nonexistent/directory",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for nonexistent directory")
		}

		if !strings.Contains(result["error"].(string), "not found") {
			t.Errorf("expected 'not found' error, got: %s", result["error"])
		}
	})

	t.Run("path is not a directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(filePath, []byte("test"), 0644)

		result, err := ListDirectory(ctx, map[string]interface{}{
			"directory_path": filePath,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for file path")
		}

		if !strings.Contains(result["error"].(string), "not a directory") {
			t.Errorf("expected 'not a directory' error, got: %s", result["error"])
		}
	})
}

func TestSearchFiles(t *testing.T) {
	ctx := context.Background()

	t.Run("simple pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "file2.py"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("test"), 0644)

		result, err := SearchFiles(ctx, map[string]interface{}{
			"pattern":        "*.txt",
			"root_directory": tmpDir,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		matches := result["matches"].([]string)
		if len(matches) != 2 {
			t.Errorf("expected 2 .txt matches, got %d: %v", len(matches), matches)
		}
	})

	t.Run("recursive pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("test"), 0644)
		os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
		os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("test"), 0644)
		os.Mkdir(filepath.Join(tmpDir, "subdir", "deep"), 0755)
		os.WriteFile(filepath.Join(tmpDir, "subdir", "deep", "deep.txt"), []byte("test"), 0644)

		result, err := SearchFiles(ctx, map[string]interface{}{
			"pattern":        "*.txt",
			"root_directory": tmpDir,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		matches := result["matches"].([]string)
		if len(matches) != 3 {
			t.Errorf("expected 3 .txt matches in all subdirectories, got %d: %v", len(matches), matches)
		}
	})

	t.Run("max results limit", func(t *testing.T) {
		tmpDir := t.TempDir()
		for i := 0; i < 10; i++ {
			filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
			os.WriteFile(filename, []byte("test"), 0644)
		}

		result, err := SearchFiles(ctx, map[string]interface{}{
			"pattern":        "*.txt",
			"root_directory": tmpDir,
			"max_results":    5,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		matches := result["matches"].([]string)
		if len(matches) > 5 {
			t.Errorf("expected at most 5 matches, got %d", len(matches))
		}

		if !result["truncated"].(bool) {
			t.Errorf("expected truncated=true when hitting max_results")
		}
	})

	t.Run("missing pattern parameter", func(t *testing.T) {
		result, err := SearchFiles(ctx, map[string]interface{}{
			"root_directory": "/tmp",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for missing parameter")
		}

		if !strings.Contains(result["error"].(string), "required") {
			t.Errorf("expected 'required' error, got: %s", result["error"])
		}
	})

	t.Run("root directory not found", func(t *testing.T) {
		result, err := SearchFiles(ctx, map[string]interface{}{
			"pattern":        "*.txt",
			"root_directory": "/nonexistent/directory",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for nonexistent directory")
		}

		if !strings.Contains(result["error"].(string), "not found") {
			t.Errorf("expected 'not found' error, got: %s", result["error"])
		}
	})

	t.Run("root path is not a directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(filePath, []byte("test"), 0644)

		result, err := SearchFiles(ctx, map[string]interface{}{
			"pattern":        "*.txt",
			"root_directory": filePath,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for file path")
		}

		if !strings.Contains(result["error"].(string), "not a directory") {
			t.Errorf("expected 'not a directory' error, got: %s", result["error"])
		}
	})
}

func TestFileOperationsRegistration(t *testing.T) {
	tools := []string{"read_file", "write_file", "list_directory", "search_files"}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			tool, err := GetTool(toolName)
			if err != nil {
				t.Fatalf("tool %s not registered: %v", toolName, err)
			}

			if tool.Name != toolName {
				t.Errorf("expected tool name=%s, got %s", toolName, tool.Name)
			}

			if tool.Description == "" {
				t.Errorf("tool %s has empty description", toolName)
			}

			if tool.Parameters == nil {
				t.Errorf("tool %s has nil parameters", toolName)
			}

			if tool.Function == nil {
				t.Errorf("tool %s has nil function", toolName)
			}
		})
	}
}
