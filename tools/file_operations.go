package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ReadFile(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "file_path parameter is required",
		}, nil
	}

	encoding := GetStringParam(params, "encoding", "utf-8")
	maxBytes := GetIntParam(params, "max_bytes", 1024*1024)

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid file path: %v", err),
		}, nil
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("File not found: %s", absPath),
			}, nil
		}
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("error accessing file: %v", err),
		}, nil
	}

	if info.IsDir() {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Path is not a file: %s", absPath),
		}, nil
	}

	sizeBytes := int(info.Size())
	if sizeBytes > maxBytes {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("File too large (%d bytes, max %d bytes)", sizeBytes, maxBytes),
		}, nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		if strings.Contains(err.Error(), "invalid UTF-8") || strings.Contains(err.Error(), "invalid argument") {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Encoding error: %v. Try a different encoding.", err),
			}, nil
		}
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	contentStr := string(content)
	lines := len(strings.Split(strings.TrimSuffix(contentStr, "\n"), "\n"))
	if contentStr == "" {
		lines = 0
	}

	return map[string]interface{}{
		"success":    true,
		"content":    contentStr,
		"file_path":  absPath,
		"size_bytes": sizeBytes,
		"encoding":   encoding,
		"lines":      lines,
	}, nil
}

func WriteFile(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "file_path parameter is required",
		}, nil
	}

	content, ok := params["content"]
	if !ok {
		return map[string]interface{}{
			"success": false,
			"error":   "content parameter is required",
		}, nil
	}

	contentStr, ok := content.(string)
	if !ok {
		contentStr = fmt.Sprintf("%v", content)
	}

	_ = GetStringParam(params, "encoding", "utf-8")
	createDirs := GetBoolParam(params, "create_dirs", true)
	overwrite := GetBoolParam(params, "overwrite", false)

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid file path: %v", err),
		}, nil
	}

	_, err = os.Stat(absPath)
	exists := err == nil

	if exists && !overwrite {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("File already exists and overwrite=False: %s", absPath),
		}, nil
	}

	if createDirs {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("error creating directories: %v", err),
			}, nil
		}
	}

	created := !exists
	if err := os.WriteFile(absPath, []byte(contentStr), 0644); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("error getting file info: %v", err),
		}, nil
	}

	sizeBytes := int(info.Size())
	lines := len(strings.Split(strings.TrimSuffix(contentStr, "\n"), "\n"))
	if contentStr == "" {
		lines = 0
	}

	return map[string]interface{}{
		"success":    true,
		"file_path":  absPath,
		"size_bytes": sizeBytes,
		"lines":      lines,
		"created":    created,
	}, nil
}

func ListDirectory(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	directoryPath := GetStringParam(params, "directory_path", ".")
	recursive := GetBoolParam(params, "recursive", false)
	includeHidden := GetBoolParam(params, "include_hidden", false)
	fileTypes := GetStringSliceParam(params, "file_types", []string{})

	absPath, err := filepath.Abs(directoryPath)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid directory path: %v", err),
		}, nil
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Directory not found: %s", absPath),
			}, nil
		}
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("error accessing directory: %v", err),
		}, nil
	}

	if !info.IsDir() {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Path is not a directory: %s", absPath),
		}, nil
	}

	files := []string{}
	directories := []string{}

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if path == absPath {
			return nil
		}

		name := filepath.Base(path)
		if !includeHidden && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !recursive && filepath.Dir(path) != absPath {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			directories = append(directories, path)
		} else {
			if len(fileTypes) > 0 {
				ext := filepath.Ext(path)
				matched := false
				for _, ft := range fileTypes {
					if ext == ft {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}
			files = append(files, path)
		}

		return nil
	}

	if err := filepath.WalkDir(absPath, walkFunc); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	sort.Strings(files)
	sort.Strings(directories)

	return map[string]interface{}{
		"success":     true,
		"directory":   absPath,
		"files":       files,
		"directories": directories,
		"total_files": len(files),
		"total_dirs":  len(directories),
	}, nil
}

func SearchFiles(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "pattern parameter is required",
		}, nil
	}

	rootDirectory := GetStringParam(params, "root_directory", ".")
	maxResults := GetIntParam(params, "max_results", 1000)

	absPath, err := filepath.Abs(rootDirectory)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid root directory: %v", err),
		}, nil
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Root directory not found: %s", absPath),
			}, nil
		}
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("error accessing root directory: %v", err),
		}, nil
	}

	if !info.IsDir() {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Root path is not a directory: %s", absPath),
		}, nil
	}

	searchPattern := pattern
	if !strings.Contains(pattern, "/") && !strings.Contains(pattern, "\\") {
		searchPattern = filepath.Join("**", pattern)
	}

	matches := []string{}
	var walkErr error

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if len(matches) >= maxResults {
			return filepath.SkipAll
		}

		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return nil
		}

		matched, err := filepath.Match(strings.ReplaceAll(searchPattern, "**", "*"), relPath)
		if err != nil {
			walkErr = err
			return filepath.SkipAll
		}

		if !matched && strings.Contains(searchPattern, "**") {
			parts := strings.Split(relPath, string(filepath.Separator))
			for i := range parts {
				subPath := strings.Join(parts[i:], string(filepath.Separator))
				matched, _ = filepath.Match(strings.TrimPrefix(searchPattern, "**/"), subPath)
				if matched {
					break
				}
			}
		}

		if matched && !d.IsDir() {
			matches = append(matches, path)
		}

		return nil
	}

	if err := filepath.WalkDir(absPath, walkFunc); err != nil && walkErr == nil {
		walkErr = err
	}

	if walkErr != nil {
		return map[string]interface{}{
			"success": false,
			"error":   walkErr.Error(),
		}, nil
	}

	sort.Strings(matches)
	truncated := len(matches) >= maxResults

	return map[string]interface{}{
		"success":        true,
		"pattern":        pattern,
		"root_directory": absPath,
		"matches":        matches,
		"total_matches":  len(matches),
		"truncated":      truncated,
	}, nil
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "read_file",
		Description: "Read contents of a file from the local filesystem",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to read",
				},
				"encoding": map[string]interface{}{
					"type":        "string",
					"description": "File encoding (default: utf-8)",
					"default":     "utf-8",
				},
				"max_bytes": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum bytes to read (default: 1MB)",
					"default":     1048576,
				},
			},
			"required": []string{"file_path"},
		},
		Function: ReadFile,
	})

	RegisterTool(&ToolDefinition{
		Name:        "write_file",
		Description: "Write content to a file on the local filesystem",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to write",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write",
				},
				"encoding": map[string]interface{}{
					"type":        "string",
					"description": "File encoding (default: utf-8)",
					"default":     "utf-8",
				},
				"create_dirs": map[string]interface{}{
					"type":        "boolean",
					"description": "Create parent directories if needed (default: true)",
					"default":     true,
				},
				"overwrite": map[string]interface{}{
					"type":        "boolean",
					"description": "Allow overwriting existing files (default: false)",
					"default":     false,
				},
			},
			"required": []string{"file_path", "content"},
		},
		Function: WriteFile,
	})

	RegisterTool(&ToolDefinition{
		Name:        "list_directory",
		Description: "List files and directories in a directory",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to directory to list (default: current directory)",
					"default":     ".",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "List recursively (default: false)",
					"default":     false,
				},
				"include_hidden": map[string]interface{}{
					"type":        "boolean",
					"description": "Include hidden files (default: false)",
					"default":     false,
				},
				"file_types": map[string]interface{}{
					"type":        "array",
					"description": "Filter by file extensions (e.g., ['.py', '.txt'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		Function: ListDirectory,
	})

	RegisterTool(&ToolDefinition{
		Name:        "search_files",
		Description: "Search for files matching a glob pattern",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern (e.g., '*.py', '**/*.txt')",
				},
				"root_directory": map[string]interface{}{
					"type":        "string",
					"description": "Root directory to search (default: current directory)",
					"default":     ".",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results to return (default: 1000)",
					"default":     1000,
				},
			},
			"required": []string{"pattern"},
		},
		Function: SearchFiles,
	})
}
