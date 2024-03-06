package blueprint

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

// go test -run TestValidateKey -v
func TestValidateKey(t *testing.T) {
	key, message := generateValidKeyMessagePair()
	t.Log("Raw:")
	t.Log(key)
	t.Log(message)

	if !ValidateKey(key, message) {
		t.Fatal("expected to be true but returned false")
	} else {
		t.Log("check mutable")
		t.Log(message)

		base64Key := base64.StdEncoding.EncodeToString(key[:])
		base64Msg := base64.StdEncoding.EncodeToString(message[:])
		t.Log("Base64")
		t.Log(base64Key)
		t.Log(base64Msg)

		t.Log("Decoded from Base64")
		t.Log(base64.StdEncoding.DecodeString(base64Key))
		decodedMsg, err := base64.StdEncoding.DecodeString(base64Msg)
		if err != nil {
			t.Log(err)
		}
		t.Log(decodedMsg[:48])
		t.Log(message[:48])
	}
}

func generateValidKeyMessagePair() (Key, []byte) {
	// random 16 bytes
	uuid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	// content
	content := []byte("helloworld")
	// hash
	hash := sha256.Sum256(append(uuid, content...))
	key := append(uuid, hash[:]...)
	message := append(key, content...)
	return Key(key), message
}
