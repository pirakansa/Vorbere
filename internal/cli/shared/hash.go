package shared

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hex returns lowercase hex encoded digest for content.
func SHA256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
