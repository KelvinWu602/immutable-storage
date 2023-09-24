package ipfs

import (
	"github.com/KelvinWu602/immutable-storage/blueprint"
)

const mappingPageMaxSize uint = 512

type cidProfile struct {
	cid    string
	source string
}

type discoverProgressProfile struct {
	lastReadPage        uint
	lastReadEntryOffset uint
}

type IPFS struct {
	keyToCid         map[blueprint.Key]cidProfile
	indexDirectory   string
	indexPage        string
	discoverProgress map[string]discoverProgressProfile
}

func (ipfs IPFS) New(configFile string) error {
	// TODO
	return nil
}

func (ipfs IPFS) Store(key blueprint.Key, message []byte) error {
	// TODO
	return nil
}

func (ipfs IPFS) Read(key blueprint.Key) ([]byte, error) {
	//TODO
	return []byte{}, nil
}

func (ipfs IPFS) AvailableKeys() []blueprint.Key {
	//TODO
	return []blueprint.Key{}
}

func (ipfs IPFS) IsDiscovered(key blueprint.Key) bool {
	_, discovered := ipfs.keyToCid[key]
	return discovered
}

func (ipfs IPFS) propageWrite(updatedIndexIPNS string) error {
	//TODO
	return nil
}

func (ipfs IPFS) sync(key blueprint.Key) (string, string) {
	//TODO
	return "", ""
}

func (ipfs IPFS) updateIndexDirectoryWorker() error {
	//TODO
	return nil
}

func (ipfs IPFS) updateMappingWorker() error {
	//TODO
	return nil
}
