package skill

import (
	"context"
	"os/exec"
	"testing"
)

// TestVerifySkill_NoCosign verifies that when cosign is not installed, VerifySkill
// returns a descriptive error with install instructions rather than a nil result.
func TestVerifySkill_NoCosign(t *testing.T) {
	// Only run this test when cosign is actually absent so we get deterministic
	// results in both CI (no cosign) and developer machines that may have it.
	_, err := exec.LookPath("cosign")
	if err == nil {
		t.Skip("cosign is installed; skipping NoCosign test")
	}

	ref := ResolvedRef{
		Registry: "ghcr.io",
		Path:     "dojo-skills/strategic-scout",
		Tag:      "1.0.0",
		Scheme:   "skill",
	}

	_, verifyErr := VerifySkill(context.Background(), ref)
	if verifyErr == nil {
		t.Fatal("expected error when cosign is not installed, got nil")
	}
	errMsg := verifyErr.Error()
	if !containsAny(errMsg, "cosign", "PATH", "Install") {
		t.Errorf("expected install hint in error, got: %q", errMsg)
	}
}

// TestVerifyResult_TrustTierDisplay verifies TierBadge for all three tier values.
func TestVerifyResult_TrustTierDisplay(t *testing.T) {
	cases := []struct {
		tier     int
		expected string
	}{
		{TierCommunity, "[community]"},
		{TierVerified, "[verified]"},
		{TierOfficial, "[official]"},
		{-1, "[unsigned]"}, // unknown tier falls back to unsigned
	}

	for _, tc := range cases {
		got := TierBadge(tc.tier)
		if got != tc.expected {
			t.Errorf("TierBadge(%d) = %q, want %q", tc.tier, got, tc.expected)
		}
	}
}

// TestParseCosignOutput verifies parsing of sample cosign JSON output.
func TestParseCosignOutput(t *testing.T) {
	// Sample output structure produced by `cosign verify --output json`
	sampleJSON := `[
  {
    "critical": {
      "identity": {"docker-reference": "ghcr.io/dojo-skills/strategic-scout"},
      "image": {"docker-manifest-digest": "sha256:abc123"},
      "type": "cosign container image signature"
    },
    "optional": {
      "subject": "https://github.com/trespies/dojo-skills/.github/workflows/publish.yml@refs/heads/main",
      "Issuer": "https://token.actions.githubusercontent.com"
    }
  }
]`

	result, err := parseCosignOutput([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("parseCosignOutput failed: %v", err)
	}

	if !result.Verified {
		t.Error("expected Verified=true")
	}
	if result.TrustTier != TierOfficial {
		t.Errorf("expected TrustTier=%d (official), got %d", TierOfficial, result.TrustTier)
	}
	if !containsAny(result.Identity, "trespies") {
		t.Errorf("expected identity to contain 'trespies', got %q", result.Identity)
	}
	if result.Issuer != "https://token.actions.githubusercontent.com" {
		t.Errorf("unexpected issuer: %q", result.Issuer)
	}
}

// TestParseCosignOutput_VerifiedTier checks that non-official identities resolve
// to TierVerified rather than TierOfficial.
func TestParseCosignOutput_VerifiedTier(t *testing.T) {
	sampleJSON := `[
  {
    "critical": {
      "identity": {"docker-reference": "ghcr.io/some-publisher/my-skill"},
      "image": {"docker-manifest-digest": "sha256:def456"},
      "type": "cosign container image signature"
    },
    "optional": {
      "subject": "https://github.com/some-publisher/my-skill/.github/workflows/publish.yml@refs/heads/main",
      "Issuer": "https://token.actions.githubusercontent.com"
    }
  }
]`

	result, err := parseCosignOutput([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("parseCosignOutput failed: %v", err)
	}

	if !result.Verified {
		t.Error("expected Verified=true")
	}
	if result.TrustTier != TierVerified {
		t.Errorf("expected TrustTier=%d (verified), got %d", TierVerified, result.TrustTier)
	}
}

// TestBuildCosignArgs verifies the correct argument construction for different refs.
func TestBuildCosignArgs(t *testing.T) {
	cases := []struct {
		name         string
		ref          ResolvedRef
		wantImage    string
		wantIssuer   string
		wantIdentity string
		wantOutput   string
	}{
		{
			name: "skill scheme ref",
			ref: ResolvedRef{
				Registry: "ghcr.io",
				Path:     "dojo-skills/strategic-scout",
				Tag:      "1.0.0",
				Scheme:   "skill",
			},
			wantImage:    "ghcr.io/dojo-skills/strategic-scout:1.0.0",
			wantIssuer:   "https://token.actions.githubusercontent.com",
			wantIdentity: ".*",
			wantOutput:   "json",
		},
		{
			name: "oci scheme ref with latest",
			ref: ResolvedRef{
				Registry: "registry.example.com",
				Path:     "myorg/my-skill",
				Tag:      "latest",
				Scheme:   "oci",
			},
			wantImage:    "registry.example.com/myorg/my-skill:latest",
			wantIssuer:   "https://token.actions.githubusercontent.com",
			wantIdentity: ".*",
			wantOutput:   "json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := buildCosignArgs(tc.ref)

			// args: ["verify", image, "--certificate-identity-regexp", id, "--certificate-oidc-issuer", issuer, "--output", "json"]
			if len(args) < 8 {
				t.Fatalf("expected at least 8 args, got %d: %v", len(args), args)
			}
			if args[0] != "verify" {
				t.Errorf("args[0] = %q, want %q", args[0], "verify")
			}
			if args[1] != tc.wantImage {
				t.Errorf("args[1] (image) = %q, want %q", args[1], tc.wantImage)
			}
			// Find --certificate-oidc-issuer value
			issuerVal := findArgValue(args, "--certificate-oidc-issuer")
			if issuerVal != tc.wantIssuer {
				t.Errorf("--certificate-oidc-issuer = %q, want %q", issuerVal, tc.wantIssuer)
			}
			// Find --certificate-identity-regexp value
			identVal := findArgValue(args, "--certificate-identity-regexp")
			if identVal != tc.wantIdentity {
				t.Errorf("--certificate-identity-regexp = %q, want %q", identVal, tc.wantIdentity)
			}
			// Find --output value
			outputVal := findArgValue(args, "--output")
			if outputVal != tc.wantOutput {
				t.Errorf("--output = %q, want %q", outputVal, tc.wantOutput)
			}
		})
	}
}

// --- helpers ---

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(sub) > 0 {
			found := false
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					found = true
					break
				}
			}
			if found {
				return true
			}
		}
	}
	return false
}

func findArgValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
