package hash

import "hash/fnv"

// Evaluate deterministically decides whether a user is active for a feature
// flag. The decision is a pure function of key and user: it hashes the
// concatenation key+"\x00"+user with FNV-1a 64-bit and compares
// hash%100 < rolloutPercent, so the same key/user pair yields the same result
// across repeated calls and process restarts. A disabled flag is never active,
// regardless of rolloutPercent.
func Evaluate(key, user string, rolloutPercent int, enabled bool) bool {
	if !enabled {
		return false
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(user))
	return int(h.Sum64()%100) < rolloutPercent
}
