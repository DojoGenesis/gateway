package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Trust tier constants for clarity.
const (
	TierCommunity = 0 // unsigned
	TierVerified  = 1 // Cosign OIDC keyless (GitHub Actions)
	TierOfficial  = 2 // Dojo Platform org-signed
)

// TierBadge returns the display badge string for a trust tier.
func TierBadge(tier int) string {
	switch tier {
	case TierOfficial:
		return "[official]"
	case TierVerified:
		return "[verified]"
	case TierCommunity:
		return "[community]"
	default:
		return "[unsigned]"
	}
}

// VerifyResult contains the outcome of a Cosign verification check.
type VerifyResult struct {
	// Verified is true when a valid Cosign signature was found.
	Verified bool

	// TrustTier is the resolved trust level: 0=community, 1=verified, 2=official.
	TrustTier int

	// Identity is the OIDC subject identity (e.g. "https://github.com/trespies/...").
	Identity string

	// Issuer is the OIDC token issuer (e.g. "https://token.actions.githubusercontent.com").
	Issuer string

	// Timestamp is when the signature was created.
	Timestamp time.Time

	// RekorEntry is the Rekor transparency log entry URL (if present).
	RekorEntry string
}

// cosignPayload is the JSON structure returned by `cosign verify --output json`.
// The outer array wraps one object per signature.
type cosignPayload struct {
	Optional struct {
		Subject string `json:"subject"`
		Issuer  string `json:"Issuer"`
	} `json:"optional"`
	Critical struct {
		Identity struct {
			DockerReference string `json:"docker-reference"`
		} `json:"identity"`
		Image struct {
			DockerManifestDigest string `json:"docker-manifest-digest"`
		} `json:"image"`
		Type string `json:"type"`
	} `json:"critical"`
}

// VerifySkill checks the Cosign signature of an OCI artifact referenced by ref.
//
// It shells out to the `cosign` CLI binary (same pattern used by `dojo tunnel`
// wrapping `cloudflared`). If the binary is not installed, a descriptive error
// with install instructions is returned.
//
// Signature trust tier is determined by the OIDC identity:
//   - Identity contains "trespies" or "dojo-skills" → TierOfficial (2)
//   - Any other valid signature → TierVerified (1)
//   - No signature / unsigned                       → TierCommunity (0)
func VerifySkill(ctx context.Context, ref ResolvedRef) (*VerifyResult, error) {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return nil, buildCosignNotFoundError()
	}

	args := buildCosignArgs(ref)
	cmd := exec.CommandContext(ctx, cosignPath, args...)

	output, err := cmd.Output()
	if err != nil {
		// cosign exits non-zero when no signature found.
		return &VerifyResult{
			Verified:  false,
			TrustTier: TierCommunity,
		}, nil
	}

	result, parseErr := parseCosignOutput(output)
	if parseErr != nil {
		// Output was present but unparseable; treat as community.
		return &VerifyResult{
			Verified:  false,
			TrustTier: TierCommunity,
		}, nil
	}

	return result, nil
}

// buildCosignArgs constructs the argument slice for the cosign verify command.
func buildCosignArgs(ref ResolvedRef) []string {
	image := fmt.Sprintf("%s/%s:%s", ref.Registry, ref.Path, ref.Tag)
	return []string{
		"verify",
		image,
		"--certificate-identity-regexp", ".*",
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		"--output", "json",
	}
}

// parseCosignOutput parses the JSON array output from `cosign verify --output json`
// into a VerifyResult.
func parseCosignOutput(data []byte) (*VerifyResult, error) {
	// cosign emits a JSON array of signature payload objects.
	var payloads []cosignPayload
	if err := json.Unmarshal(data, &payloads); err != nil {
		return nil, fmt.Errorf("parse cosign output: %w", err)
	}

	if len(payloads) == 0 {
		return nil, fmt.Errorf("parse cosign output: no signatures in output")
	}

	p := payloads[0]
	identity := p.Optional.Subject
	issuer := p.Optional.Issuer

	tier := TierVerified
	lowerIdentity := strings.ToLower(identity)
	if strings.Contains(lowerIdentity, "trespies") || strings.Contains(lowerIdentity, "dojo-skills") {
		tier = TierOfficial
	}

	return &VerifyResult{
		Verified:  true,
		TrustTier: tier,
		Identity:  identity,
		Issuer:    issuer,
		Timestamp: time.Now().UTC(),
	}, nil
}

// buildCosignNotFoundError returns a descriptive error with install instructions.
func buildCosignNotFoundError() error {
	var installHint string
	switch runtime.GOOS {
	case "darwin":
		installHint = "  macOS:  brew install cosign"
	case "linux":
		installHint = "  Linux:  curl -O -L \"https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64\"\n" +
			"           mv cosign-linux-amd64 /usr/local/bin/cosign && chmod +x /usr/local/bin/cosign"
	default:
		installHint = "  See: https://docs.sigstore.dev/system_config/installation/"
	}
	return fmt.Errorf("cosign not found in PATH\n\nInstall it:\n%s", installHint)
}
