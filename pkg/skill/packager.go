package skill

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackSkill reads a skill directory and produces the manifest JSON and content tar.
//
// It reads SKILL.md from dirPath and parses YAML frontmatter (delimited by
// "---" markers) to populate manifest fields: name, description, version
// (default "1.0.0"), triggers, and dependencies.
//
// The returned configBlob is the JSON-encoded manifest and contentTar is a tar
// archive of the entire directory.
func PackSkill(dirPath string) (manifest SkillManifest, configBlob []byte, contentTar []byte, err error) {
	// Ensure directory exists.
	info, err := os.Stat(dirPath)
	if err != nil {
		return manifest, nil, nil, fmt.Errorf("skill: pack: stat dir: %w", err)
	}
	if !info.IsDir() {
		return manifest, nil, nil, fmt.Errorf("skill: pack: %s is not a directory", dirPath)
	}

	// Read and parse SKILL.md frontmatter.
	skillPath := filepath.Join(dirPath, "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		return manifest, nil, nil, fmt.Errorf("skill: pack: read SKILL.md: %w", err)
	}

	manifest, err = parseFrontmatter(string(skillData))
	if err != nil {
		return manifest, nil, nil, fmt.Errorf("skill: pack: parse frontmatter: %w", err)
	}

	// Default version if not specified.
	if manifest.Version == "" {
		manifest.Version = "1.0.0"
	}

	// If no name, derive from directory name.
	if manifest.Name == "" {
		manifest.Name = filepath.Base(dirPath)
	}

	// Serialize manifest to JSON.
	configBlob, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return manifest, nil, nil, fmt.Errorf("skill: pack: marshal manifest: %w", err)
	}

	// Create tar archive of the directory.
	contentTar, err = createTar(dirPath)
	if err != nil {
		return manifest, nil, nil, fmt.Errorf("skill: pack: create tar: %w", err)
	}

	return manifest, configBlob, contentTar, nil
}

// parseFrontmatter extracts YAML-like frontmatter from SKILL.md content.
// It handles a minimal subset of YAML: scalar key: value pairs and simple
// lists (- item lines). This avoids pulling in a full YAML dependency.
func parseFrontmatter(content string) (SkillManifest, error) {
	var m SkillManifest

	// Find frontmatter delimiters. Some SKILL.md files have a copyright
	// header (comment lines) before the opening "---", so we scan for the
	// first line that is exactly "---" (with optional surrounding whitespace).
	lines := strings.Split(content, "\n")
	openLine := -1
	closeLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if openLine < 0 {
				openLine = i
			} else {
				closeLine = i
				break
			}
		}
	}

	if openLine < 0 || closeLine < 0 {
		return m, fmt.Errorf("no frontmatter found (missing --- delimiters)")
	}

	fmBlock := strings.Join(lines[openLine+1:closeLine], "\n")

	return parseFrontmatterBlock(fmBlock)
}

// parseFrontmatterBlock parses a YAML-like frontmatter block (without the
// "---" delimiters) into a SkillManifest.
func parseFrontmatterBlock(fmBlock string) (SkillManifest, error) {
	var m SkillManifest

	lines := strings.Split(fmBlock, "\n")

	var currentKey string
	var currentList []string
	inMetadata := false // tracks whether we are inside a "metadata:" section

	flushList := func() {
		if currentKey != "" && len(currentList) > 0 {
			switch currentKey {
			case "triggers":
				m.Triggers = currentList
			case "dependencies", "tool_dependencies":
				m.Dependencies = currentList
			case "authors":
				m.Authors = currentList
			}
		}
		currentKey = ""
		currentList = nil
	}

	for _, line := range lines {
		trimLine := strings.TrimSpace(line)
		if trimLine == "" || strings.HasPrefix(trimLine, "#") {
			continue
		}

		// Detect indentation to track nested sections.
		isIndented := len(line) > 0 && (line[0] == ' ' || line[0] == '\t')

		// If we hit a non-indented key, we've left the metadata section.
		if !isIndented && inMetadata {
			inMetadata = false
		}

		// Check if this is a list item (starts with "- ").
		if strings.HasPrefix(trimLine, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(trimLine, "- "))
			item = stripQuotes(item)
			currentList = append(currentList, item)
			continue
		}

		// Must be a key: value pair. Flush any pending list.
		flushList()

		colonIdx := strings.Index(trimLine, ":")
		if colonIdx < 0 {
			continue
		}

		key := strings.TrimSpace(trimLine[:colonIdx])
		value := strings.TrimSpace(trimLine[colonIdx+1:])
		value = stripQuotes(value)

		// Enter metadata section when we see "metadata:" with no value.
		if key == "metadata" && value == "" {
			inMetadata = true
			continue
		}

		// Map keys — handle both top-level and metadata-nested forms.
		switch key {
		case "name":
			if !inMetadata {
				m.Name = value
			}
		case "version":
			m.Version = value
		case "description":
			if !inMetadata {
				m.Description = value
			}
		case "license":
			m.License = value
		case "author":
			if value != "" {
				m.Authors = []string{value}
			}
		case "triggers", "dependencies", "tool_dependencies", "authors":
			currentKey = key
			if value != "" && value != "|" && value != ">" {
				currentList = append(currentList, value)
			}
		}
	}

	flushList()
	return m, nil
}

// stripQuotes removes surrounding single or double quotes from a string.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// addFileToTar writes a single file into the tar archive. Using a separate
// function ensures the deferred f.Close fires after each file is written,
// preventing file descriptor accumulation during filepath.Walk.
func addFileToTar(tw *tar.Writer, path string, relPath string, info os.FileInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	header := &tar.Header{
		Name: filepath.ToSlash(relPath),
		Size: info.Size(),
		Mode: int64(info.Mode()),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	_, err = io.Copy(tw, f)
	return err
}

// createTar creates a tar archive from a directory.
func createTar(dirPath string) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Compute relative path for tar entry name.
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}

		// Skip the root directory entry itself.
		if relPath == "." {
			return nil
		}

		// Security: reject paths that contain ".." components to prevent
		// directory traversal attacks.
		if strings.Contains(relPath, "..") {
			return nil
		}

		// Security: if the entry is a symlink, resolve it and verify the
		// target is still within the base directory.
		if info.Mode()&os.ModeSymlink != 0 {
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				// Skip unresolvable symlinks.
				return nil
			}
			if !strings.HasPrefix(realPath, absDir+string(os.PathSeparator)) {
				// Symlink points outside the skill directory; skip it.
				return nil
			}
		}

		// Directories: write a header-only entry.
		if info.IsDir() {
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return fmt.Errorf("file info header: %w", err)
			}
			header.Name = filepath.ToSlash(relPath)
			return tw.WriteHeader(header)
		}

		// Regular files: delegate to helper so defer closes promptly.
		return addFileToTar(tw, path, relPath, info)
	})
	if err != nil {
		_ = tw.Close()
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close tar: %w", err)
	}

	return buf.Bytes(), nil
}
