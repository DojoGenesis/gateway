package skill

import (
	"testing"
)

// TestIsReservedName_OfficialSkill verifies that an official Dojo skill name is reserved.
func TestIsReservedName_OfficialSkill(t *testing.T) {
	if !IsReservedName("strategic-scout") {
		t.Error("expected 'strategic-scout' to be reserved (official skill)")
	}
}

// TestIsReservedName_PlatformName verifies that the platform namespace is reserved.
func TestIsReservedName_PlatformName(t *testing.T) {
	if !IsReservedName("dojo-platform") {
		t.Error("expected 'dojo-platform' to be reserved (platform name)")
	}
}

// TestIsReservedName_SlopsquatName verifies that a slopsquatting corpus name is reserved.
func TestIsReservedName_SlopsquatName(t *testing.T) {
	if !IsReservedName("code-reviewer") {
		t.Error("expected 'code-reviewer' to be reserved (slopsquat corpus)")
	}
}

// TestIsReservedName_AllowedName verifies that a custom name is NOT reserved.
func TestIsReservedName_AllowedName(t *testing.T) {
	if IsReservedName("my-custom-skill") {
		t.Error("expected 'my-custom-skill' to NOT be reserved")
	}
}

// TestLoadReservedNames_Count verifies that the corpus has at least 200 entries.
func TestLoadReservedNames_Count(t *testing.T) {
	names := LoadReservedNames()
	if len(names) < 200 {
		t.Errorf("expected at least 200 reserved names, got %d", len(names))
	}
}
