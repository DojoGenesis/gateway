package skill

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed reserved-names.txt
var reservedNamesData string

var (
	reservedOnce  sync.Once
	reservedNames map[string]bool
)

// LoadReservedNames parses the embedded reserved-names.txt file and returns
// a set of reserved skill names. Lines beginning with '#' and empty lines
// are ignored.
//
// The result is computed once and cached.
func LoadReservedNames() map[string]bool {
	reservedOnce.Do(func() {
		names := make(map[string]bool)
		for _, line := range strings.Split(reservedNamesData, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			names[strings.ToLower(line)] = true
		}
		reservedNames = names
	})
	return reservedNames
}

// IsReservedName reports whether name is in the reserved names corpus.
// The check is case-insensitive.
func IsReservedName(name string) bool {
	names := LoadReservedNames()
	return names[strings.ToLower(name)]
}
