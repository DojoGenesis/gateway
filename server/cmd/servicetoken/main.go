// Command servicetoken mints a long-lived, revocable, least-privilege service
// token for a machine client of the gateway.
//
// It is an OFFLINE operator tool. Issuance is deliberately not exposed over
// HTTP: minting requires JWT_SECRET, so this must run on the gateway host (or
// anywhere the operator has loaded the same secret), and there is no network
// path — authenticated or otherwise — by which a caller can mint one.
//
//	# on the gateway host, with the gateway's own environment loaded
//	go run ./cmd/servicetoken -service pdi
//	go run ./cmd/servicetoken -service pdi -ttl 720h
//
// The token is written to stdout and nothing else; all diagnostics go to
// stderr. Redirect it straight into your secret store — do not paste it into a
// file that gets committed, and do not echo it into shell history.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/DojoGenesis/gateway/server/middleware"
)

func main() {
	var (
		service        = flag.String("service", "", "client name, e.g. \"pdi\" (required; becomes subject \"service:<name>\")")
		ttl            = flag.Duration("ttl", middleware.DefaultServiceTokenTTL, "token lifetime, e.g. 720h; must not exceed the hard maximum")
		allowDevSecret = flag.Bool("allow-dev-secret", false, "permit minting while signing with the built-in development secret (local testing only)")
	)
	flag.Parse()

	if *service == "" {
		fatal("-service is required, e.g. -service pdi")
	}

	// A token signed with the publicly known development fallback is forgeable
	// by anyone who has read this repository. Refuse by default.
	if middleware.UsingDefaultJWTSecret() && !*allowDevSecret {
		fatal("refusing to mint: this process is signing with the built-in development secret.\n"+
			"       %s is unset or empty in this shell (the deprecated %s is also honoured).\n"+
			"       Load the SAME secret the running gateway uses, then retry.\n"+
			"       (Pass -allow-dev-secret only for local testing.)",
			middleware.EnvJWTSecret, middleware.EnvJWTSecretLegacy)
	}

	token, jti, err := middleware.IssueServiceToken(*service, *ttl)
	if err != nil {
		fatal("failed to mint service token: %v", err)
	}

	expiry := time.Now().Add(*ttl).UTC().Format(time.RFC3339)

	fmt.Fprintf(os.Stderr, "subject : %s%s\n", middleware.ServiceSubjectPrefix, *service)
	fmt.Fprintf(os.Stderr, "role    : %s  (cannot satisfy AdminAuthMiddleware)\n", middleware.ServiceRole)
	fmt.Fprintf(os.Stderr, "jti     : %s\n", jti)
	fmt.Fprintf(os.Stderr, "expires : %s  (ttl %s)\n", expiry, *ttl)
	fmt.Fprintf(os.Stderr, "\nRECORD THE jti. To revoke this one token, add it to %s\n", middleware.EnvRevokedTokenIDs)
	fmt.Fprintf(os.Stderr, "in the gateway environment and restart:\n")
	fmt.Fprintf(os.Stderr, "    %s=%s\n", middleware.EnvRevokedTokenIDs, jti)
	fmt.Fprintf(os.Stderr, "To revoke every token issued to this service instead:\n")
	fmt.Fprintf(os.Stderr, "    %s=%s%s\n", middleware.EnvRevokedSubjects, middleware.ServiceSubjectPrefix, *service)
	fmt.Fprintf(os.Stderr, "\nThe token follows on stdout. It is a bearer credential — treat it as a\n")
	fmt.Fprintf(os.Stderr, "password, store it in a secret manager, and never commit it.\n\n")

	fmt.Println(token)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
