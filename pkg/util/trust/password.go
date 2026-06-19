package trust

import (
	"crypto/sha256"
	"encoding/hex"
)

// DeriveTrustPassword generates a deterministic trust password from a given seed.
// The output is a hex string truncated to 32 characters for readability.
func DeriveTrustPassword(seed string) string {
	sum := sha256.Sum256([]byte("lxd-trust:" + seed))
	s := hex.EncodeToString(sum[:])
	if len(s) > 32 {
		return s[:32]
	}
	return s
}
