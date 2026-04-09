// Command dojo is the Dojo Platform CLI for managing skills, tools, and agents.
//
// Usage:
//
//	dojo skill list                        — List installed skills
//	dojo skill search <query>              — Search installed skills by name or description
//	dojo skill install <ref>               — Install a skill (warns if unsigned)
//	dojo skill install <ref> --verify      — Install with Cosign signature verification
//	dojo skill install <ref> --force       — Install even if verification fails
//	dojo skill publish <dir>               — Package and publish a skill from a directory
//	dojo skill info <name> [version]       — Show skill metadata
//	dojo skill package-all <plugins-dir>   — Batch-package all SKILL.md files under a directory
//	dojo tunnel [port]                     — Start a cloudflared tunnel to expose localhost:<port> (default 8080)
//	dojo tunnel stop                       — Stop the running tunnel
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DojoGenesis/gateway/pkg/skill"
	"github.com/DojoGenesis/gateway/runtime/cas"
)

const defaultCASPath = "dojo-skills.db"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	switch subcmd {
	case "skill":
		if len(os.Args) < 3 {
			printSkillUsage()
			os.Exit(1)
		}
		if err := runSkillCommand(os.Args[2], os.Args[3:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "tunnel":
		if err := runTunnelCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "bridge":
		if err := runBridgeCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Println("dojo v3.1.0-era3")
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcmd)
		printUsage()
		os.Exit(1)
	}
}

func runSkillCommand(action string, args []string) error {
	casPath := os.Getenv("DOJO_CAS_PATH")
	if casPath == "" {
		casPath = defaultCASPath
	}

	casStore, err := cas.NewSQLiteStore(casPath)
	if err != nil {
		return fmt.Errorf("open CAS store at %s: %w", casPath, err)
	}
	defer casStore.Close()

	store := skill.NewSkillStore(casStore)
	ctx := context.Background()

	switch action {
	case "list":
		return skill.ListSkills(ctx, store)

	case "search":
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		return skill.SearchSkillsCmd(ctx, store, query)

	case "install":
		if len(args) < 1 {
			return fmt.Errorf("usage: dojo skill install <ref> [--verify] [--force]\n  ref: local dir, skill://name@version, oci://registry/path:tag, github:org/repo//path")
		}
		ref := args[0]
		var verify, force bool
		for _, flag := range args[1:] {
			switch flag {
			case "--verify":
				verify = true
			case "--force":
				force = true
			}
		}
		return skill.InstallSkill(ctx, store, ref, verify, force)

	case "publish":
		if len(args) < 1 {
			return fmt.Errorf("usage: dojo skill publish <dir>")
		}
		return skill.PublishSkill(ctx, store, args[0])

	case "info":
		if len(args) < 1 {
			return fmt.Errorf("usage: dojo skill info <name> [version]")
		}
		name := args[0]
		version := "1.0.0"
		if len(args) > 1 {
			version = args[1]
		}
		_, err := skill.SkillInfo(ctx, store, name, version)
		return err

	case "package-all":
		if len(args) < 1 {
			return fmt.Errorf("usage: dojo skill package-all <plugins-dir>")
		}
		return packageAll(ctx, store, args[0])

	default:
		return fmt.Errorf("unknown skill action: %s", action)
	}
}

// packageAll walks a plugins directory, finds all SKILL.md files, and packages
// each skill directory into the CAS store.
func packageAll(ctx context.Context, store *skill.SkillStore, pluginsDir string) error {
	var skillDirs []string

	err := filepath.Walk(pluginsDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Name() == "SKILL.md" && !info.IsDir() {
			skillDirs = append(skillDirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", pluginsDir, err)
	}

	if len(skillDirs) == 0 {
		fmt.Printf("No SKILL.md files found under %s\n", pluginsDir)
		return nil
	}

	fmt.Printf("Found %d skills to package\n\n", len(skillDirs))

	var succeeded, failed int
	var errors []string

	for _, dir := range skillDirs {
		relDir, _ := filepath.Rel(pluginsDir, dir)
		err := skill.PublishSkill(ctx, store, dir)
		if err != nil {
			failed++
			errMsg := fmt.Sprintf("  FAIL  %s: %v", relDir, err)
			errors = append(errors, errMsg)
			fmt.Println(errMsg)
		} else {
			succeeded++
		}
	}

	fmt.Printf("\nPackaged %d/%d skills", succeeded, succeeded+failed)
	if failed > 0 {
		fmt.Printf(" (%d failed)", failed)
	}
	fmt.Println()

	if failed > 0 {
		fmt.Println("\nFailures:")
		fmt.Println(strings.Join(errors, "\n"))
		return fmt.Errorf("%d skills failed to package", failed)
	}

	return nil
}

func printUsage() {
	fmt.Println(`Dojo Platform CLI

Usage:
  dojo skill <action> [args]    Manage skills
  dojo tunnel [port]            Start a cloudflared tunnel (default port: 8080)
  dojo tunnel stop              Stop the running tunnel
  dojo bridge                   Start Channel Bridge with NATS bus (production)
  dojo version                  Print version
  dojo help                     Show this help

Skill Actions:
  list                          List installed skills
  search <query>                Search installed skills by name or description
  install <ref>                 Install a skill (warns if unsigned)
  install <ref> --verify        Install with Cosign signature verification
  install <ref> --force         Install even if verification fails
  publish <dir>                 Package and publish a skill
  info <name> [version]         Show skill metadata
  package-all <plugins-dir>     Batch-package all skills

Tunnel:
  dojo tunnel                   Expose http://localhost:8080 via cloudflared
  dojo tunnel 3000              Expose http://localhost:3000 via cloudflared
  dojo tunnel stop              Kill any running cloudflared tunnel

Bridge:
  dojo bridge                   Start Channel Bridge with NATS bus (Era 3)

Environment:
  DOJO_CAS_PATH                 Path to CAS database (default: dojo-skills.db)
  DOJO_DATA_DIR                 Data directory for event WAL (default: ./data)
  DOJO_CREDENTIAL_BACKEND       Credential backend: "env" (default), "infisical"`)
}

func printSkillUsage() {
	fmt.Println(`Usage: dojo skill <action> [args]

Actions:
  list                          List installed skills
  search <query>                Search skills by name or description
  install <ref>                 Install a skill (warns if unsigned)
  install <ref> --verify        Install with Cosign signature verification
  install <ref> --force         Install even if verification fails
  publish <dir>                 Package and publish a skill
  info <name> [version]         Show skill metadata
  package-all <plugins-dir>     Batch-package all skills

References:
  ./path/to/skill               Local directory
  skill://name@version          Default registry (ghcr.io/dojo-skills)
  oci://registry/path:tag       Direct OCI reference
  github:org/repo//path         GitHub-hosted skill`)
}
