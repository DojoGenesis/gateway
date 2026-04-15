package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
		// Bundle holds the Rekor transparency log entry embedded by cosign.
		Bundle *cosignBundle `json:"Bundle"`
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

// cosignBundle is the Rekor bundle embedded in cosign verify output.
type cosignBundle struct {
	SignedEntryTimestamp string `json:"SignedEntryTimestamp"`
	Payload              struct {
		Body           string `json:"body"`
		IntegratedTime int64  `json:"integratedTime"`
		LogIndex       int64  `json:"logIndex"`
		LogID          string `json:"logID"`
	} `json:"Payload"`
}

const rekorSearchBaseURL = "https://search.sigstore.dev"

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

	result := &VerifyResult{
		Verified:  true,
		TrustTier: tier,
		Identity:  identity,
		Issuer:    issuer,
		Timestamp: time.Now().UTC(),
	}

	// Extract Rekor transparency log entry URL and timestamp from bundle.
	if b := p.Optional.Bundle; b != nil {
		if b.Payload.LogIndex > 0 {
			result.RekorEntry = fmt.Sprintf("%s/?logIndex=%d", rekorSearchBaseURL, b.Payload.LogIndex)
		}
		if b.Payload.IntegratedTime > 0 {
			result.Timestamp = time.Unix(b.Payload.IntegratedTime, 0).UTC()
		}
	}

	return result, nil
}

// VerifyCASBlob verifies the Cosign signature of a raw CAS blob.
//
// data is the blob content. bundleJSON is the cosign bundle JSON produced by
// `cosign sign-blob --bundle bundle.json <file>`. Both are written to temp
// files; cosign verify-blob is then invoked against them.
//
// Returns a VerifyResult with Verified=false (TierCommunity) when the
// signature is absent or invalid, without returning an error — identical
// to VerifySkill's behavior for unsigned artifacts.
func VerifyCASBlob(ctx context.Context, data []byte, bundleJSON []byte) (*VerifyResult, error) {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return nil, buildCosignNotFoundError()
	}

	dataFile, err := os.CreateTemp("", "cas-blob-*")
	if err != nil {
		return nil, fmt.Errorf("cosign verify-blob: create data temp: %w", err)
	}
	defer func() { _ = os.Remove(dataFile.Name()) }()
	if _, err := dataFile.Write(data); err != nil {
		_ = dataFile.Close()
		return nil, fmt.Errorf("cosign verify-blob: write data: %w", err)
	}
	_ = dataFile.Close()

	bundleFile, err := os.CreateTemp("", "cas-bundle-*.json")
	if err != nil {
		return nil, fmt.Errorf("cosign verify-blob: create bundle temp: %w", err)
	}
	defer func() { _ = os.Remove(bundleFile.Name()) }()
	if _, err := bundleFile.Write(bundleJSON); err != nil {
		_ = bundleFile.Close()
		return nil, fmt.Errorf("cosign verify-blob: write bundle: %w", err)
	}
	_ = bundleFile.Close()

	args := []string{
		"verify-blob",
		"--bundle", bundleFile.Name(),
		"--certificate-identity-regexp", ".*",
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		dataFile.Name(),
	}
	cmd := exec.CommandContext(ctx, cosignPath, args...)
	if err := cmd.Run(); err != nil {
		// Non-zero exit means verification failed (unsigned or invalid signature).
		return &VerifyResult{
			Verified:  false,
			TrustTier: TierCommunity,
		}, nil
	}

	// Signature verified. Extract identity from bundle JSON for tier determination.
	result := &VerifyResult{
		Verified:  true,
		TrustTier: TierVerified,
		Timestamp: time.Now().UTC(),
	}
	if tier, identity, rekorEntry, ts, ok := parseBundleJSON(bundleJSON); ok {
		result.TrustTier = tier
		result.Identity = identity
		result.RekorEntry = rekorEntry
		if !ts.IsZero() {
			result.Timestamp = ts
		}
	}
	return result, nil
}

// parseBundleJSON extracts trust tier, identity, Rekor URL and timestamp from
// a cosign sign-blob bundle JSON (Old "Payload" format).
//
// Returns ok=false when the bundle is not parseable; callers treat that as
// a successful verification at TierVerified with no identity detail.
func parseBundleJSON(bundleJSON []byte) (tier int, identity, rekorEntry string, ts time.Time, ok bool) {
	var b cosignBundle
	if err := json.Unmarshal(bundleJSON, &b); err != nil {
		return 0, "", "", time.Time{}, false
	}
	if b.Payload.IntegratedTime > 0 {
		ts = time.Unix(b.Payload.IntegratedTime, 0).UTC()
	}
	if b.Payload.LogIndex > 0 {
		rekorEntry = fmt.Sprintf("%s/?logIndex=%d", rekorSearchBaseURL, b.Payload.LogIndex)
	}
	// Bundle doesn't carry the OIDC subject; tier defaults to verified.
	tier = TierVerified
	return tier, "", rekorEntry, ts, true
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
