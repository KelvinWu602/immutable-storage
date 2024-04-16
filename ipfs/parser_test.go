package ipfs

import (
	"io"
	"testing"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/stretchr/testify/assert"
)

func TestOpenFileWithPath(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/test-openFileWithPath.txt"
	file, err := openFileWithPath(filePath)
	if err != nil {
		t.Fatalf("error when calling openFileWithPath(): %s", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("error when calling io.ReadAll(): %s", err)
	}
	assert.Equal(t, "hello, world!", string(content), "File content should be 'hello, world!'")
}

func TestParseConfigWithValidFormat(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-config-valid.yml"
	file, err := openFileWithPath(filePath)
	if err != nil {
		t.Fatalf("error when calling openFileWithPath(): %s", err)
	}
	defer file.Close()
	config, _ := parseConfig(file)

	assert := assert.New(t)
	assert.Equal("localhost:5001", config.Host, "config.Host should be 'localhost:5001'.")
	assert.Equal(5000, config.Timeout, "config.Timeout should be 5000.")
}

func TestParseConfigWithInvalidFormat(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-config-invalid.yml"
	file, err := openFileWithPath(filePath)
	if err != nil {
		t.Fatalf("error when calling openFileWithPath(): %s", err)
	}
	defer file.Close()
	config, _ := parseConfig(file)

	assert := assert.New(t)
	// expected to be equal defaultConfig
	assert.Equal("localhost:5001", config.Host, "config.Host should be 'localhost:5001'.")
	assert.Equal(2000, config.Timeout, "config.Timeout should be '2000'.")
}

func TestParseNodestxt(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-nodes.txt"
	file, err := openFileWithPath(filePath)
	if err != nil {
		t.Fatalf("error when calling openFileWithPath(): %s", err)
	}
	defer file.Close()
	nodesIPNS, _ := parseNodestxt(file)

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
	file, err := openFileWithPath(filePath)
	if err != nil {
		t.Fatalf("error when calling openFileWithPath(): %s", err)
	}
	defer file.Close()
	mappings, _ := parseMappings(file)

	assert := assert.New(t)
	assert.Equal(3, len(mappings), "size of mappings should be 3.")

	assert.Equal(blueprint.Key([]byte{225, 83, 140, 12, 141, 73, 87, 141, 233, 244, 199, 5, 31, 98, 182, 66, 171, 13, 122, 204, 241, 157, 80, 30, 74, 251, 9, 147, 155, 21, 14, 235, 91, 130, 46, 231, 36, 222, 30, 215, 207, 205, 100, 100, 46, 170, 44, 202}), mappings[0].key, "1st element key should starts with '225'")
	assert.Equal("cid01234", mappings[0].cid, "1st element cid should be 'cid01234'")

	assert.Equal(blueprint.Key([]byte{153, 161, 209, 132, 114, 210, 73, 116, 114, 164, 95, 228, 11, 196, 90, 145, 195, 174, 221, 5, 140, 62, 32, 251, 31, 32, 81, 75, 111, 230, 113, 77, 22, 69, 112, 139, 12, 214, 70, 135, 103, 58, 240, 151, 193, 244, 109, 109}), mappings[1].key, "2nd element key should starts with '153'")
	assert.Equal("cid012345678", mappings[1].cid, "2nd element cid should be 'cid012345678'")

	assert.Equal(blueprint.Key([]byte{57, 170, 52, 13, 247, 210, 0, 18, 171, 165, 12, 69, 163, 131, 250, 89, 123, 145, 69, 181, 7, 97, 174, 175, 223, 27, 160, 40, 89, 233, 126, 180, 200, 133, 95, 78, 230, 182, 122, 186, 203, 249, 189, 13, 50, 252, 0, 209}), mappings[2].key, "3rd element key should starts with '57'")
	assert.Equal("cid01234567890", mappings[2].cid, "3rd element cid should be 'cid01234567890'")
}

func TestParseMessage(t *testing.T) {
	const filePath string = "../testing-assets/ipfs/parser/mock-message.txt"
	file, err := openFileWithPath(filePath)
	if err != nil {
		t.Fatalf("error when calling openFileWithPath(): %s", err)
	}
	defer file.Close()
	message, err := parseMessage(file)
	if err != nil {
		t.Fatalf("error when calling parseMessage(file): %s", err)
	}

	assert := assert.New(t)

	assert.Equal("uuid456789012345", string(message.uuid[:]), "uuid not match")

	assert.Equal("checksum890123456789012345678901", string(message.checksum[:]), "checksum not match")

	assert.Equal("payloadpayloadpayload", string(message.payload), "payload not match")
}
