package blueprint

import (
	"bytes"
	"crypto/sha256"
)

// ValidateKey returns a boolean value indicating whether a key is valid for a message.
// key: a 48 byte key that will be used to associate with the message
// message: an array of bytes begins with the key, followed by message content
func ValidateKey(key Key, message []byte) bool {
	// check message header = key
	if !bytes.Equal(key[:], message[:len(key)]) {
		return false
	}
	// compute the sha256 checksum of message
	correctChecksum := sha256.Sum256(message)
	keyChecksum := key[len(key)-32:]
	return bytes.Equal(correctChecksum[:], keyChecksum)
}
