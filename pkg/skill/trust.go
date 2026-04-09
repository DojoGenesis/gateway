package skill

import (
	"os"
	"strconv"
)

// getTrustMinimum reads the minimum trust tier for skill installation from
// the DOJO_TRUST_MINIMUM environment variable. Returns TierCommunity (0) if
// unset or invalid.
//
// Valid values:
//   - "0" or unset: TierCommunity (any skill allowed)
//   - "1": TierVerified (Cosign OIDC signature required)
//   - "2": TierOfficial (Dojo Platform org signature required)
func getTrustMinimum() int {
	val := os.Getenv("DOJO_TRUST_MINIMUM")
	if val == "" {
		return TierCommunity
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < TierCommunity || n > TierOfficial {
		return TierCommunity
	}
	return n
}
