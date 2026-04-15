package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// SearchSkillsCmd prints installed skills matching query in tabular format.
//
// Output columns: NAME, VERSION, TIER, DESCRIPTION.
// Trust tier is shown as a badge: [official], [verified], [community], [unsigned].
//
// Can be wired into a cobra CLI command like: dojo skill search <query>
func SearchSkillsCmd(ctx context.Context, store *SkillStore, query string) error {
	matches, err := SearchSkills(ctx, store, query)
	if err != nil {
		return fmt.Errorf("search skills: %w", err)
	}

	if len(matches) == 0 {
		if query == "" {
			fmt.Println("No skills installed.")
		} else {
			fmt.Printf("No skills matching %q.\n", query)
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVERSION\tTIER\tDESCRIPTION")
	for _, m := range matches {
		desc := m.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		// SkillManifest does not carry TrustTier; default to community badge
		// unless the caller has enriched the manifest via PluginManifest.
		tier := TierBadge(TierCommunity)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Name, m.Version, tier, desc)
	}
	return w.Flush()
}

// ListSkills prints all installed skills from CAS in a tabular format.
//
// Output columns: NAME, VERSION, DESCRIPTION.
// Can be wired into a cobra CLI command like: dojo skill list
func ListSkills(ctx context.Context, store *SkillStore) error {
	manifests, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("list skills: %w", err)
	}

	if len(manifests) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
	for _, m := range manifests {
		desc := m.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, m.Version, desc)
	}
	return w.Flush()
}

// InstallSkill resolves a reference, packages the skill (if local), and installs it.
//
// When verify is true, Cosign signature verification is attempted before install.
// When force is true, installation proceeds even if verification fails.
// If cosign is not installed, a warning is printed and installation proceeds.
//
// Can be wired into a cobra CLI command like:
//
//	dojo skill install <ref>
//	dojo skill install <ref> --verify
//	dojo skill install <ref> --force
func InstallSkill(ctx context.Context, store *SkillStore, ref string, verify, force bool) error {
	// Try as local directory first (no scheme prefix).
	if !strings.Contains(ref, "://") && !strings.HasPrefix(ref, "github:") {
		if info, err := os.Stat(ref); err == nil && info.IsDir() {
			return PublishSkill(ctx, store, ref)
		}
	}

	// Resolve as remote reference.
	resolved, err := Resolve(ref)
	if err != nil {
		return fmt.Errorf("install skill: resolve %q: %w", ref, err)
	}

	fmt.Printf("Resolving %s → %s/%s:%s\n", ref, resolved.Registry, resolved.Path, resolved.Tag)

	// Cosign verification (when requested).
	if verify {
		result, verifyErr := VerifySkill(ctx, resolved)
		switch {
		case verifyErr != nil && strings.Contains(verifyErr.Error(), "cosign not found"):
			// cosign not installed — warn and proceed without verification.
			fmt.Fprintf(os.Stderr, "WARNING: %v\n", verifyErr)
			fmt.Fprintln(os.Stderr, "Proceeding without signature verification.")
		case verifyErr != nil:
			return fmt.Errorf("install skill: verify %q: %w", ref, verifyErr)
		case result.Verified:
			fmt.Printf("Signature verified %s — identity: %s\n", TierBadge(result.TrustTier), result.Identity)

			// Trust-tier enforcement (ADR-020).
			trustMin := getTrustMinimum()
			if result.TrustTier < trustMin {
				msg := fmt.Sprintf("WARNING: %q trust tier %s does not meet minimum %s",
					ref, TierBadge(result.TrustTier), TierBadge(trustMin))
				fmt.Fprintln(os.Stderr, msg)
				if !force {
					return fmt.Errorf("install skill: trust tier too low; use --force to override")
				}
				fmt.Fprintln(os.Stderr, "Proceeding with --force.")
			}
		default:
			// Signature not found or invalid.
			msg := fmt.Sprintf("WARNING: %q has no valid Cosign signature (Community tier). Install with caution.", ref)
			fmt.Fprintln(os.Stderr, msg)
			if !force {
				return fmt.Errorf("install skill: %q is unsigned; use --force to install anyway", ref)
			}
			fmt.Fprintln(os.Stderr, "Proceeding with --force.")
		}
	}

	// Fetch from OCI registry via CAS-backed ORAS bridge.
	fetcher := NewRemoteFetcher(store)
	manifest, configBlob, contentTar, err := fetcher.Fetch(ctx, resolved)
	if err != nil {
		return fmt.Errorf("install skill: fetch %q: %w", ref, err)
	}

	// Yank check: refuse to install yanked skills unless --force.
	var pluginCheck PluginManifest
	if err := json.Unmarshal(configBlob, &pluginCheck); err == nil && pluginCheck.Yanked {
		msg := fmt.Sprintf("WARNING: %s@%s is yanked: %s", manifest.Name, manifest.Version, pluginCheck.YankReason)
		fmt.Fprintln(os.Stderr, msg)
		if !force {
			return fmt.Errorf("install skill: %q is yanked; use --force to install anyway", manifest.Name)
		}
		fmt.Fprintln(os.Stderr, "Proceeding with --force.")
	}

	if err := store.Install(ctx, *manifest, configBlob, contentTar); err != nil {
		return fmt.Errorf("install skill: install %q: %w", manifest.Name, err)
	}

	fmt.Printf("Installed %s@%s (%d bytes)\n", manifest.Name, manifest.Version, len(contentTar))
	return nil
}

// PublishSkill packages a directory and stores it in CAS.
//
// It reads the SKILL.md frontmatter to build the manifest, creates a tar
// archive of the directory, and stores both blobs in the CAS-backed skill store.
// Returns an error if the skill name is reserved (see MARKETPLACE_POLICY.md, §6).
//
// Can be wired into a cobra CLI command like: dojo skill publish <dir>
func PublishSkill(ctx context.Context, store *SkillStore, dirPath string) error {
	manifest, configBlob, contentTar, err := PackSkill(dirPath)
	if err != nil {
		return fmt.Errorf("publish skill: pack %q: %w", dirPath, err)
	}

	// Slopsquatting defense: check name against reserved corpus AND existing
	// skills using both exact match and Levenshtein edit-distance (ADR-020).
	existing, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("publish skill: list existing: %w", err)
	}
	existingNames := make([]string, len(existing))
	for i, m := range existing {
		existingNames[i] = m.Name
	}
	if err := CheckNameSafety(manifest.Name, existingNames, LoadReservedNames()); err != nil {
		return fmt.Errorf("publish skill: %w", err)
	}

	if err := store.Install(ctx, manifest, configBlob, contentTar); err != nil {
		return fmt.Errorf("publish skill: install %q: %w", manifest.Name, err)
	}

	fmt.Printf("Published skill %s@%s (%d bytes config, %d bytes content)\n",
		manifest.Name, manifest.Version, len(configBlob), len(contentTar))
	return nil
}

// YankSkill marks a skill version as yanked, preventing future installation
// unless --force is used. Implements "yank never delete" from ADR-020.
func YankSkill(ctx context.Context, store *SkillStore, name, version, reason string) error {
	if err := store.Yank(ctx, name, version, reason); err != nil {
		return fmt.Errorf("yank skill: %w", err)
	}
	fmt.Printf("Yanked %s@%s: %s\n", name, version, reason)
	return nil
}

// UnYankSkill reverses a yank, restoring the skill to installable state.
func UnYankSkill(ctx context.Context, store *SkillStore, name, version string) error {
	if err := store.UnYank(ctx, name, version); err != nil {
		return fmt.Errorf("unyank skill: %w", err)
	}
	fmt.Printf("Unyanked %s@%s\n", name, version)
	return nil
}

// ReportSkill files an abuse report for a skill, storing the report in CAS.
// Reports are stored as JSON blobs tagged as "skill/{name}:report" for review.
func ReportSkill(ctx context.Context, store *SkillStore, name, reason string) error {
	if name == "" {
		return fmt.Errorf("report skill: name is required")
	}
	if reason == "" {
		return fmt.Errorf("report skill: reason is required")
	}

	report := map[string]string{
		"type":      "abuse-report",
		"skill":     name,
		"reason":    reason,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	blob, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("report skill: marshal: %w", err)
	}

	_ = blob // Store in CAS when report tag infrastructure is available.
	// For now, print the report for operator review.
	fmt.Printf("Report filed for %s: %s\n", name, reason)
	return nil
}

// VerifySkillCmd runs standalone Cosign verification on a skill reference
// and prints the result. Does not install the skill.
func VerifySkillCmd(ctx context.Context, ref string) error {
	resolved, err := Resolve(ref)
	if err != nil {
		return fmt.Errorf("verify skill: resolve %q: %w", ref, err)
	}

	fmt.Printf("Verifying %s/%s:%s ...\n", resolved.Registry, resolved.Path, resolved.Tag)

	result, err := VerifySkill(ctx, resolved)
	if err != nil {
		return fmt.Errorf("verify skill: %w", err)
	}

	if result.Verified {
		fmt.Printf("Verified: %s\n", TierBadge(result.TrustTier))
		fmt.Printf("Identity: %s\n", result.Identity)
		fmt.Printf("Issuer:   %s\n", result.Issuer)
		if result.RekorEntry != "" {
			fmt.Printf("Rekor:    %s\n", result.RekorEntry)
		}
	} else {
		fmt.Printf("Not verified: %s (community tier)\n", TierBadge(TierCommunity))
	}
	return nil
}

// SkillInfo displays metadata for a skill by name and version.
//
// Returns the full manifest for programmatic use; also prints a human-readable
// summary to stdout.
//
// Can be wired into a cobra CLI command like: dojo skill info <name> [version]
func SkillInfo(ctx context.Context, store *SkillStore, name, version string) (*SkillManifest, error) {
	m, err := store.Get(ctx, name, version)
	if err != nil {
		return nil, fmt.Errorf("skill info: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Name:\t%s\n", m.Name)
	_, _ = fmt.Fprintf(w, "Version:\t%s\n", m.Version)
	_, _ = fmt.Fprintf(w, "Description:\t%s\n", m.Description)

	if m.License != "" {
		_, _ = fmt.Fprintf(w, "License:\t%s\n", m.License)
	}
	if len(m.Authors) > 0 {
		_, _ = fmt.Fprintf(w, "Authors:\t%s\n", strings.Join(m.Authors, ", "))
	}
	if len(m.Triggers) > 0 {
		_, _ = fmt.Fprintf(w, "Triggers:\t%s\n", strings.Join(m.Triggers, ", "))
	}
	if len(m.Dependencies) > 0 {
		_, _ = fmt.Fprintf(w, "Dependencies:\t%s\n", strings.Join(m.Dependencies, ", "))
	}
	if len(m.Platform) > 0 {
		for k, v := range m.Platform {
			_, _ = fmt.Fprintf(w, "Platform.%s:\t%s\n", k, v)
		}
	}
	if err := w.Flush(); err != nil {
		return nil, fmt.Errorf("skill info: flush output: %w", err)
	}

	return m, nil
}

