package blueprint

import (
	"bytes"
	"crypto/sha256"
	"log"
)

// ValidateKey returns a boolean value indicating whether a key is valid for a message.
// key: a 48 byte key that will be used to associate with the message
// message: an array of bytes begins with the key, followed by message content
func ValidateKey(key Key, message []byte) bool {
	// log.Println("DEBUG: called validate key")
	msgCopy := make([]byte, len(message))
	copy(msgCopy, message)
	// check message header = key
	if !bytes.Equal(key[:], message[:KeySize]) {
		log.Println("Key not match")
		return false
	}
	// compute the sha256 checksum of concat(message.key.uuid, message.content)
	msgUUID := msgCopy[:16]
	msgContent := msgCopy[KeySize:]
	correctChecksum := sha256.Sum256(append(msgUUID, msgContent...))
	keyChecksum := key[len(key)-32:]

	debug := false
	if debug {
		log.Println("msgUUID      :", msgUUID)
		log.Println("msgContent   :", msgContent)
		log.Println("correctChecksum:", correctChecksum)
		log.Println("keyChecksum    :", keyChecksum)
	}
	return bytes.Equal(correctChecksum[:], keyChecksum)
}
