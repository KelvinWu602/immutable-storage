package blueprint

type ImmutableStorage interface {
	Store(key Key, message []byte) error
	Read(key Key) ([]byte, error)
	AvailableKeys() []Key
	IsDiscovered(key Key) bool
}

type Key [48]byte

const KeySize int = 48
