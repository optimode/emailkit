package disposable

import "strings"

// IsDisposable returns whether the given domain is a known disposable domain.
func IsDisposable(domain string) bool {
	_, ok := disposableSet[strings.ToLower(domain)]
	return ok
}
