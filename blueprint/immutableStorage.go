// The blueprint package defines the interface for the forus Immutable Storage component.
// To create a new implementation of Immutable Storage, implement the ImmutableStorge interface.
package blueprint

// ImmutableStorage should be implemented by all concrete implementation.
type ImmutableStorage interface {
	// Store a (key, message) pair in an immutable storage.
	Store(key Key, message []byte) error
	// Read the message content associated with key from an immutable storage.
	Read(key Key) ([]byte, error)
	// AvailableKeys return a list of all keys available for read.
	AvailableKeys() []Key
	// IsDiscovered return true if the key is available for read.
	IsDiscovered(key Key) bool
}

// Key is 48 bytes. The first 16 bytes are random bits. The last 32 bytes is SHA256 checksum of message content.
type Key [KeySize]byte

// KeySize defines the length of Key.
const KeySize int = 48
