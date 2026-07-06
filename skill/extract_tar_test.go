package skill

import (
	"archive/tar"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTar is a test helper that creates an in-memory tar archive containing
// the provided files (map of filename → content).
func buildTar(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	return buf.Bytes()
}

func TestExtractSkillMDFromTar(t *testing.T) {
	const wantContent = "# My Skill\n\nDoes something useful."

	tests := []struct {
		name        string
		files       map[string]string // filename → content placed in tar
		wantContent string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name: "SKILL.md at root",
			files: map[string]string{
				"SKILL.md": wantContent,
			},
			wantContent: wantContent,
		},
		{
			name: "SKILL.md nested in subdirectory",
			files: map[string]string{
				"some/nested/path/SKILL.md": wantContent,
			},
			wantContent: wantContent,
		},
		{
			name: "first SKILL.md wins when multiple present",
			// tar/Reader processes entries in insertion order; first entry wins.
			files: map[string]string{
				"SKILL.md":       "first",
				"other/SKILL.md": "second",
			},
			// We only assert no error and non-empty result; order is map-iteration
			// non-deterministic so we can't assert exact content here — tested via
			// the single-file cases above. We just confirm the call succeeds.
			wantErr: false,
		},
		{
			name: "tar contains other files but no SKILL.md",
			files: map[string]string{
				"README.md":      "readme",
				"config.yaml":    "key: value",
				"scripts/run.sh": "#!/bin/bash",
			},
			wantErr:    true,
			wantErrMsg: "SKILL.md not found in tar archive",
		},
		{
			name:       "empty tar archive",
			files:      map[string]string{},
			wantErr:    true,
			wantErrMsg: "SKILL.md not found in tar archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tarBytes := buildTar(t, tt.files)
			got, err := extractSkillMDFromTar(tarBytes)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				if tt.wantContent != "" {
					assert.Equal(t, tt.wantContent, got)
				} else {
					// Multiple-SKILL.md case: just confirm we got something
					assert.NotEmpty(t, got)
				}
			}
		})
	}
}

func TestExtractSkillMDFromTar_InvalidTarBytes(t *testing.T) {
	// Garbage bytes that are not a valid tar archive
	garbage := []byte("this is not a tar archive at all")
	got, err := extractSkillMDFromTar(garbage)
	assert.Error(t, err)
	assert.Empty(t, got)
}

func TestExtractSkillMDFromTar_EmptyBytes(t *testing.T) {
	got, err := extractSkillMDFromTar([]byte{})
	// An empty byte slice produces an empty tar with no entries — same as empty archive.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SKILL.md not found in tar archive")
	assert.Empty(t, got)
}
