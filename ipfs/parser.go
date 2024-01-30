package ipfs

import (
	"bufio"
	"io"
	"log"
	"os"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/go-yaml/yaml"
)

// config represents the content of a single config.yaml file
type config struct {
	// the location where the IPFS daemon is listening to
	Host string `yaml:"host"`
	// the maximum timeout in ms for each call to the IPFS daemon
	Timeout int `yaml:"timeout"`
}

// message represents a value stored via the ImmutableStorage interface
type message struct {
	// uuid is the first 16 bytes of the message key, which can be random values.
	uuid [16]byte
	// checksum is a SHA256 hash of concat(uuid,payload).
	checksum [32]byte
	// payload is the actual message content stored by upper application.
	payload []byte
}

// defaultConfig will be returned when the config.yaml file is unspecified or malformed
var defaultConfig config = config{
	Host:    "localhost:5001",
	Timeout: 2000,
}

// Functions for getting current network states. They should not mutate the network states.
// These functions are expected to be mocked during a unit test for the parse functions.

// openFile returns the io.ReadCloser of a file located at path
// returns a nil pointer if any error occured
func openFileWithPath(path string) io.ReadCloser {
	file, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return nil
	}
	return file
}

// openFile returns the io.ReadCloser of an IPFS object pointed by IPNS name.
// returns a nil pointer if any error occured
func openFileWithIPNS(daemon *ipfsClient, ipns string) io.ReadCloser {
	cid, err := daemon.resolveIPNSPointer(ipns)
	if err != nil {
		log.Println(err)
		return nil
	}
	file, err := daemon.readFileWithCID(cid)
	if err != nil {
		log.Println(err)
		return nil
	}
	return file
}

// openFile returns the io.ReadCloser of an IPFS object pointed by CID.
// returns a nil pointer if any error occured
func openFileWithCID(daemon *ipfsClient, cid string) io.ReadCloser {
	file, err := daemon.readFileWithCID(cid)
	if err != err {
		log.Println(err)
		return nil
	}
	return file
}

// parseConfig returns the pointer to a config struct which represents the file
// returns the defaultConfig when any error occured
func parseConfig(file io.ReadCloser) config {
	// Ignore file closing error as we only read file content
	// see https://www.joeshaw.org/dont-defer-close-on-writable-files/
	defer file.Close()
	// load all file content into a buffer, assuming no larger than 1024 bytes
	fileReader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	resultLength, err := fileReader.Read(buffer)
	if err != nil {
		log.Println(err)
		return defaultConfig
	}
	// parse the input data using yaml parser
	var config config
	err = yaml.Unmarshal(buffer[:resultLength], &config)
	if err != nil {
		log.Println(err)
		return defaultConfig
	}
	return config
}

// parseNodestxt returns a slice of mappingsIPNS stored in the nodes.txt file pointed by nodesIPNS
// returns empty slice when any error occured
//
// nodes.txt is expected to have the following format for each entry:
// <ipns>;
func parseNodestxt(file io.ReadCloser) []string {
	// Ignore file closing error as we only read file content
	defer file.Close()
	fileReader := bufio.NewReader(file)
	// Read until the delimiter ';'
	entry, err := fileReader.ReadSlice(';')
	mappingsIPNSs := make([]string, 0)
	// Check if an io.EOF error is returned
	for err == nil {
		mappingIPNS := string(entry[:len(entry)-1])
		mappingsIPNSs = append(mappingsIPNSs, mappingIPNS)
		entry, err = fileReader.ReadSlice(';')
	}
	if err != nil && err != io.EOF {
		return make([]string, 0)
	}
	return mappingsIPNSs
}

// parseMappingsCID returns a slice of mappingEntry struct with a maximum size of 512
// returns empty slice when any error occured
//
// mapping page is expected to have the following format for each entry:
// <key>,<cid>;
func parseMappings(file io.ReadCloser) []mappingEntry {
	// Ignore file closing error as we only read file content
	defer file.Close()
	fileReader := bufio.NewReader(file)
	// Read until the delimiter ';'
	entry, err := fileReader.ReadSlice(';')
	result := make([]mappingEntry, 0)
	for err == nil {
		// Read first blueprint.KeySize bytes as key
		key := blueprint.Key(entry[:blueprint.KeySize])
		// Read remaining bytes as cid (except last byte which is delimiter ';')
		cid := string(entry[blueprint.KeySize+1 : len(entry)-1])

		result = append(result, mappingEntry{key: key, cid: cid})
		entry, err = fileReader.ReadSlice(';')
	}
	if err != nil && err != io.EOF {
		return make([]mappingEntry, 0)
	}
	return result
}

// parseMessage returns a pointer to message struct stored as an IPFS object, without validation.
// return nil in case of any error
//
// message is expected to have the following format:
// <uuid><checksum><payload>
func parseMessage(file io.ReadCloser) (message, error) {
	// Ignore file closing error as we only read file content
	defer file.Close()
	fileReader := bufio.NewReader(file)

	var msg message
	var onErrorValue message
	// read the first 16 bytes as uuid
	_, err := io.ReadFull(fileReader, msg.uuid[:])
	if err != nil {
		return onErrorValue, err
	}
	// read the next 32 bytes as checksum
	_, err = io.ReadFull(fileReader, msg.checksum[:])
	if err != nil {
		return onErrorValue, err
	}
	// read the remaining bytes as payload
	msg.payload, err = io.ReadAll(fileReader)
	if err != nil {
		return onErrorValue, err
	}

	return msg, nil
}
