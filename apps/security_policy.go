package apps

import (
	"fmt"
	"net/http"
	"strings"
)

// SecurityPolicy enforces CSP and permissions for MCP Apps.
type SecurityPolicy struct {
	allowedOrigins       []string
	dangerousPermissions []string
}

// NewSecurityPolicy creates a security policy with default dangerous permissions.
func NewSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		dangerousPermissions: []string{"geolocation", "camera", "microphone"},
	}
}

// AddAllowedOrigin adds an origin to the CSP allowlist.
func (p *SecurityPolicy) AddAllowedOrigin(origin string) {
	p.allowedOrigins = append(p.allowedOrigins, origin)
}

// BuildCSPHeader constructs a Content-Security-Policy header value.
func (p *SecurityPolicy) BuildCSPHeader(meta *ResourceMeta) string {
	directives := []string{
		"default-src 'none'",
		"script-src 'self' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"font-src 'self'",
	}

	// Merge resource-specific CSP origins if they're in allowedOrigins
	if meta != nil && len(meta.CSP) > 0 {
		allowedSet := make(map[string]bool)
		for _, o := range p.allowedOrigins {
			allowedSet[o] = true
		}

		var extra []string
		for _, origin := range meta.CSP {
			if allowedSet[origin] {
				extra = append(extra, origin)
			}
		}

		if len(extra) > 0 {
			joined := strings.Join(extra, " ")
			directives = append(directives, "connect-src 'self' "+joined)
			// Replace the simpler connect-src with the one that includes origins
			for i, d := range directives {
				if d == "connect-src 'self'" {
					directives = append(directives[:i], directives[i+1:]...)
					break
				}
			}
		}
	}

	return strings.Join(directives, "; ")
}

// ValidatePermissions checks that no dangerous permissions are requested.
func (p *SecurityPolicy) ValidatePermissions(permissions []string) error {
	for _, perm := range permissions {
		for _, danger := range p.dangerousPermissions {
			if perm == danger {
				return fmt.Errorf("dangerous permission requested: %s", perm)
			}
		}
	}
	return nil
}

// InjectSecurityHeaders sets all security headers on the response.
func (p *SecurityPolicy) InjectSecurityHeaders(w http.ResponseWriter, meta *ResourceMeta) {
	w.Header().Set("Content-Security-Policy", p.BuildCSPHeader(meta))
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
}
