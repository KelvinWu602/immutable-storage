package ipfs

import (
	"io"
	"testing"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/stretchr/testify/assert"
)

func TestOpenFileWithPath(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/test-openFileWithPath.txt"
	file := openFileWithPath(filePath)
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("error when calling io.ReadAll(): %s", err)
	}
	assert.Equal(t, "hello, world!", string(content), "File content should be 'hello, world!'")
}

func TestParseConfigWithValidFormat(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-config-valid.yml"
	file := openFileWithPath(filePath)
	config := parseConfig(file)

	assert := assert.New(t)
	assert.Equal("localhost:5001", config.Host, "config.Host should be 'localhost:5001'.")
	assert.Equal(5000, config.Timeout, "config.Timeout should be 5000.")
}

func TestParseConfigWithInvalidFormat(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-config-invalid.yml"
	file := openFileWithPath(filePath)
	config := parseConfig(file)

	assert := assert.New(t)
	// expected to be equal defaultConfig
	assert.Equal("localhost:5001", config.Host, "config.Host should be 'localhost:5001'.")
	assert.Equal(2000, config.Timeout, "config.Timeout should be '2000'.")
}

func TestParseNodestxt(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-nodes.txt"
	file := openFileWithPath(filePath)
	nodesIPNS := parseNodestxt(file)

	assert := assert.New(t)
	assert.Equal(5, len(nodesIPNS), "length of nodesIPNS should be 5.")
	assert.Equal("IPNS1", nodesIPNS[0], "1st element should be 'IPNS1'")
	assert.Equal("IPNS2", nodesIPNS[1], "2nd element should be 'IPNS2'")
	assert.Equal("IPNS3", nodesIPNS[2], "3rd element should be 'IPNS3'")
	assert.Equal("IPNS4", nodesIPNS[3], "4th element should be 'IPNS4'")
	assert.Equal("IPNS5", nodesIPNS[4], "5th element should be 'IPNS5'")
}

func TestParseMappings(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-mappings.txt"
	file := openFileWithPath(filePath)
	mappings := parseMappings(file)

	assert := assert.New(t)
	assert.Equal(3, len(mappings), "size of mappings should be 3.")

	assert.Equal(blueprint.Key([]byte("012345678901234567890123456789012345678901234567")), mappings[0].key, "1st element key should starts with '0'")
	assert.Equal("cid01234", mappings[0].cid, "1st element cid should be 'cid01234'")

	assert.Equal(blueprint.Key([]byte("112345678901234567890123456789012345678901234567")), mappings[1].key, "2nd element key should starts with '1'")
	assert.Equal("cid012345678", mappings[1].cid, "2nd element cid should be 'cid012345678'")

	assert.Equal(blueprint.Key([]byte("212345678901234567890123456789012345678901234567")), mappings[2].key, "3rd element key should starts with '2'")
	assert.Equal("cid01234567890", mappings[2].cid, "3rd element cid should be 'cid01234567890'")
}

func TestParseMessage(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-message.txt"
	file := openFileWithPath(filePath)
	message, err := parseMessage(file)
	if err != nil {
		t.Fatalf("error when calling parseMessage(file): %s", err)
	}

	assert := assert.New(t)

	assert.Equal("uuid456789012345", string(message.uuid[:]), "uuid not match")

	assert.Equal("checksum890123456789012345678901", string(message.checksum[:]), "checksum not match")

	assert.Equal("payloadpayloadpayload", string(message.payload), "payload not match")
}
