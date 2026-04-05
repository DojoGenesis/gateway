package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// runTunnelCommand dispatches "dojo tunnel [port]" and "dojo tunnel stop".
func runTunnelCommand(args []string) error {
	if len(args) > 0 && args[0] == "stop" {
		return stopTunnel()
	}

	port := 8080
	if len(args) > 0 {
		p, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port %q: must be a number", args[0])
		}
		port = p
	}

	return startTunnel(port)
}

// startTunnel launches cloudflared and streams the tunnel URL to stdout.
func startTunnel(port int) error {
	cfPath, err := exec.LookPath("cloudflared")
	if err != nil {
		return buildCloudflaredNotFoundError()
	}

	localURL := fmt.Sprintf("http://localhost:%d", port)
	cmd := exec.Command(cfPath, "tunnel", "--url", localURL)

	// cloudflared writes the tunnel URL to stderr.
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start cloudflared: %w", err)
	}

	// Forward SIGINT / SIGTERM to the child process for clean shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			_ = cmd.Process.Signal(sig)
		}
	}()

	// Scan cloudflared's stderr until we find the tunnel URL, then print
	// the banner and keep draining so the process doesn't stall.
	tunnelURLPrinted := false
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if !tunnelURLPrinted {
			if url := parseTunnelURL(line); url != "" {
				printTunnelBanner(url, localURL)
				tunnelURLPrinted = true
			}
		}
		// Continue draining stderr silently.
	}

	// Drain any remaining bytes (scanner stops on first error/EOF).
	_, _ = io.Copy(io.Discard, stderrPipe)

	if err := cmd.Wait(); err != nil {
		// Exit code 0 after Ctrl+C is fine; surface genuine errors only.
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 0 {
				return nil
			}
		}
		return fmt.Errorf("cloudflared exited: %w", err)
	}
	return nil
}

// stopTunnel finds and terminates any cloudflared process started by dojo.
func stopTunnel() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", "cloudflared.exe")
	default:
		cmd = exec.Command("pkill", "-f", "cloudflared tunnel")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// pkill exits 1 when no processes were matched.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			fmt.Println("No running cloudflared tunnel found.")
			return nil
		}
		return fmt.Errorf("stop tunnel: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	fmt.Println("Tunnel stopped.")
	return nil
}

// parseTunnelURL extracts the trycloudflare.com URL from a cloudflared log line.
// cloudflared emits a line such as:
//
//	INF | Your quick tunnel has been created! Visit it at: https://xxx-yyy-zzz.trycloudflare.com
//
// It may also appear as a standalone URL line in some cloudflared versions.
func parseTunnelURL(line string) string {
	// Look for a https://...trycloudflare.com token anywhere on the line.
	fields := strings.Fields(line)
	for _, f := range fields {
		if strings.HasPrefix(f, "https://") && strings.Contains(f, "trycloudflare.com") {
			return strings.TrimRight(f, "/")
		}
	}
	return ""
}

// generateWebhookURLs returns a map of platform name to webhook URL.
func generateWebhookURLs(tunnelURL string) map[string]string {
	base := strings.TrimRight(tunnelURL, "/")
	return map[string]string{
		"Slack":    base + "/webhooks/slack",
		"Discord":  base + "/webhooks/discord",
		"Telegram": base + "/webhooks/telegram",
		"Email":    base + "/webhooks/email",
	}
}

// printTunnelBanner writes the prominent URL banner to stdout.
func printTunnelBanner(tunnelURL, localURL string) {
	webhooks := generateWebhookURLs(tunnelURL)
	fmt.Printf(`
Dojo Tunnel active

Public URL: %s
Local:      %s

Webhook URLs:
  Slack:    %s
  Discord:  %s
  Telegram: %s
  Email:    %s

Press Ctrl+C to stop
`, tunnelURL, localURL,
		webhooks["Slack"],
		webhooks["Discord"],
		webhooks["Telegram"],
		webhooks["Email"],
	)
}

// buildCloudflaredNotFoundError returns a descriptive error with install hints.
func buildCloudflaredNotFoundError() error {
	var installHint string
	switch runtime.GOOS {
	case "darwin":
		installHint = "  macOS:  brew install cloudflared"
	case "linux":
		installHint = "  Linux:  curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o /usr/local/bin/cloudflared && chmod +x /usr/local/bin/cloudflared"
	default:
		installHint = "  See: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"
	}
	return fmt.Errorf("cloudflared not found in PATH\n\nInstall it:\n%s", installHint)
}
