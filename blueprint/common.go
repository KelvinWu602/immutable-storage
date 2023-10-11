package blueprint

import (
	"bytes"
	"crypto/sha256"
)

func ValidateKey(key Key, message []byte) bool {
	// check if the header of the message = key
	if !bytes.Equal(key[:], message[:len(key)]) {
		return false
	}

	// compute the sha256 checksum of message
	correctChecksum := sha256.Sum256(message)
	keyChecksum := key[len(key)-32:]
	return bytes.Equal(correctChecksum[:], keyChecksum)
}
