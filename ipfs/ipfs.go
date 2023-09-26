package ipfs

import (
	"bufio"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-yaml/yaml"

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
	daemon           *ipfsRequest
	keyToCid         map[blueprint.Key]cidProfile
	indexDirectory   string
	indexPage        string
	discoverProgress map[string]discoverProgressProfile
}

type config struct {
	host    string `yaml:host`
	timeout int    `yaml:timeout`
}

func (ipfs IPFS) New(configFile string) error {
	// TODO
	/**
	  check connection to IPFS daemon
	  set up keys
	  load configFile
	  initialize directorys and files
	  start workers
	*/
	//error
	e := errors.New("Failed to create immutable-storage")

	//read configFile
	file, err := os.Open(configFile)
	if err != nil {
		log.Println(err)
		return e
	}
	buf := make([]byte, 1024)
	r := bufio.NewReader(file)
	n, err := r.Read(buf)
	if err != nil {
		log.Println(err)
		return e
	}

	//unmarshal configFile
	var config config
	yaml.Unmarshal(buf[:n], &config)

	//send HEAD request to <host>/api/v0/id
	cli := &http.Client{}
	res, err := cli.Head(config.host + "/api/v0/id")
	if err != nil {
		log.Println(err)
		return e
	}
	if res.StatusCode != 405 {
		log.Println("Abnormal response from host!")
		return e
	}
	ipfs.daemon = newIPFSClient(config.host, time.Duration(config.timeout))

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
