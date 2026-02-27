package shared

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/zeebo/blake3"
)

// SHA256Hex returns lowercase hex encoded digest for content.
func SHA256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// BLAKE3Hex returns lowercase hex encoded digest for content.
func BLAKE3Hex(content []byte) string {
	sum := blake3.Sum256(content)
	return hex.EncodeToString(sum[:])
}
