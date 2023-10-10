package ipfs

import (
	"bufio"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-yaml/yaml"

	"github.com/KelvinWu602/immutable-storage/blueprint"
)

const mappingPageMaxSize uint64 = 512
const nodestxtMaxSize uint64 = 640000

// TODO: find CID size
const entrySizeInByte int = 32 + 32 + 16 // CID size + key size

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
	nodesIPNS        string
	mappingsIPNS     string
	discoverProgress map[string]discoverProgressProfile
}

type config struct {
	host    string `yaml:host`
	timeout int    `yaml:timeout`
}

func New(configFile string) (*IPFS, error) {
	// TODO: grpc server and starting the workers
	e := errors.New("failed to initialize immutable-storage")

	config, err := parseConfigFile(configFile)
	if err != nil {
		log.Println("Error during parseConfigFile...")
		log.Println(err)
		return nil, e
	}
	//send HEAD request to <host>/api/v0/id
	err = checkDaemonAlive(config.host)
	if err != nil {
		log.Println("Error during checkDaemonAlive...")
		log.Println(err)
		return nil, e
	}
	//create new IPFS struct
	var ipfs IPFS
	//set up ipfs daemon
	ipfs.daemon = newIPFSClient(config.host, time.Duration(config.timeout))
	//initialize node state (lonely init + group init)
	//lonely init
	mappingsIPNS, oldNodesIPNS, err := lonelyInitialization(ipfs.daemon)
	if err != nil {
		log.Println("Error during lonelyInitialization...")
		log.Println(err)
		return nil, e
	}
	ipfs.mappingsIPNS = mappingsIPNS

	//group init
	newNodesIPNS, err = groupInitialization(daemon,oldNodesIPNS)
	if err!=nil {
		log.Println("Error during groupInitialization...")
		log.Println(err)
		return nil,e 
	}
	ipfs.nodesIPNS = newNodesIPNS

	// start grpc server
	// TODO: create inter-node grpc server

	// start worker processes
	// TODO: create worker functions

	return &ipfs, nil
}

func parseConfigFile(configFile string) (*config, error) {
	// create the file descriptor
	e := errors.New("failed to parse configFile")
	file, err := os.Open(configFile)
	if err != nil {
		log.Println(err)
		return nil, e
	}
	// load file content into a buffer
	buf := make([]byte, 1024)
	r := bufio.NewReader(file)
	n, err := r.Read(buf)
	if err != nil {
		log.Println(err)
		return nil, e
	}
	// parse the input data
	var config config
	yaml.Unmarshal(buf[:n], &config)
	return &config, nil
}

func checkDaemonAlive(host string) error {
	cli := &http.Client{}
	res, err := cli.Head(host + "/api/v0/id")
	if err != nil {
		log.Println(err)
		return errors.New("failed to send health check request")
	}
	switch res.StatusCode {
	case 405:
		return nil
	default:
		return errors.New("abnormal response from daemon")
	}
}

func lonelyInitialization(daemon *ipfsRequest) (mappingsIPNS string, nodesIPNS string, err error) {
	// TODO: also consider the case when existing IPNS are provided for mapping IPNS and nodesIPNS
	// initialize public keys: nodes, mappings
	keyNames, err := daemon.listKeys()
	if err != nil {
		return
	}
	initNodes := true
	initMappings := true
	for _, keyName := range keyNames {
		if keyName == "mappings" {
			initMappings = false
		}
		if keyName == "nodes" {
			initNodes = false
		}
	}

	if initMappings {
		if err = daemon.generateKey("mappings"); err != nil {
			return
		}
	}

	if initNodes {
		if err = daemon.generateKey("nodes"); err != nil {
			return
		}
	}

	// create MFS directory /mappings
	if err = daemon.createDirectory("/mappings"); err != nil {
		return
	}
	// publish IPNS name for directory /mappings
	mappingsCID, err := daemon.getDirectoryCID("/mappings")
	if err != nil {
		return
	}
	mappingsIPNS, err = daemon.publishIPNSPointer(mappingsCID, "mappings")
	if err != nil {
		return
	}
	// create and append mappingIPNS in file /nodes.txt
	// assumed each IPNS name has equal fixed length
	if err = daemon.appendStringToFile("/nodes.txt", mappingsIPNS, nodestxtMaxSize); err != nil {
		return
	}
	// publish IPNS name for file /nodes.txt
	nodesCID, err := daemon.getDirectoryCID("/nodes.txt")
	if err != nil {
		return
	}
	nodesIPNS, err = daemon.publishIPNSPointer(nodesCID, "nodes")
	if err != nil {
		return
	}
	return
}

func groupInitialization(daemon *ipfsRequest, myNodesIPNS string) (string, error) {
	// gossip broadcast mappingsIPNS to 3 random peers
	var peers [3]string
	// make a grpc call to node-discovery module for the get_member endpoint

	for _, peer := range peers {
		//	make a grpc call to peer for the propagate_write grpc endpoint
	}

	var externalNodesIPNS [3]string
	for _, peer := range peers {
		// make a grpc call to peer to request for its nodes.txt IPNS name
	}

	// get the content for each nodes.txt
	union := make(map[string]bool)
	seenRecords , err := parseNodesIPNS(daemon, myNodesIPNS)
	for _, record := range seenRecords {
		union[record] = true
	}

	for i, nodesIPNS := range externalNodesIPNS {
		content, err := parseNodesIPNS(daemon, nodesIPNS)
		if err != nil {
			return "",err
		}
		// for each unseen record, add seen with bool = false
		for _, record := range content {
			if _, ok := union[record]; !ok {
				union[record] = false
			}
		}
	}
	
	// for each unseen record, validate, append to your own nodes.txt file if validated, discard if not
	for record, seen := range union {
		if !seen {
			// check any random 1 record in the mappings
			ok, err:= validateMappingsIPNS(daemon,record,1)
			if err != nil{
				return "",err
			}
			//append the record to my own nodes.txt
			if err:= daemon.appendStringToFile("/nodes.txt",record,nodestxtMaxSize); err != nil{
				return "",err
			}
		}
	}

	// update the IPNS name for your nodes.txt
	nodesCID, err := daemon.getDirectoryCID("/nodes.txt")
	if err !=nil {
		return "",err
	}
	nodesIPNS, err := daemon.publishIPNSPointer(nodesCID,"nodes")
	return nodesIPFS, err
}

// parseNodesIPNS returns a slice of mappingsIPNS stored in the nodes.txt file pointed by nodesIPNS
func parseNodesIPNS(daemon *ipfsRequest, nodesIPNS string) ([]string, error) {
	nodesCID, err := daemon.resolveIPNSPointer(nodesIPNS)
	if err != nil {
		return nil, err
	}
	file, err := daemon.readFileWithCID(nodesCID)
	if err != nil {
		return nil, err
	}
	// TODO: change to more memory safe approach
	buf := make([]byte, nodestxtMaxSize)
	r := bufio.NewReader(file)
	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	mappingsIPNSs := make([]string, n/len(nodesIPNS))
	for i := 0; i < n/len(nodesIPNS); i++ {
		mappingsIPNS := string(buf[i*len(nodesIPNS) : (i+1)*len(nodesIPNS)])
		// validate the mappingsIPNS
		mappingsIPNSs[i] = mappingsIPNS
	}
	return mappingsIPNSs, nil
}

func validateMappingsIPNS(daemon *ipfsRequest, mappingsIPNS string, n int) (bool, error) {
	// randomly pick n entries and check their validness

	mappingsCID, err := daemon.resolveIPNSPointer(mappingsIPNS)
	if err != nil {
		return false, err
	}
	filesNameToCid, err := daemon.getDAGLinks(mappingsCID)
	if err != nil {
		return false, err
	}

	for i := 0; i < n; i++ {
		pageNumber := rand.Intn(len(filesNameToCid))

		fileName := string(pageNumber) + ".txt"
		fileSizeInByte, err := daemon.getDirectorySize("/mappings/" + fileName)
		if err != nil {
			return false, err
		}

		entryNumber := rand.Intn(int(fileSizeInByte) / entrySizeInByte)
		// TODO: change to use the read with byte offset, limit instead
		file, err := daemon.readFileWithCID(filesNameToCid[fileName])
		if err != nil {
			return false, err
		}
		key, cid := //somehow read the entryNumber th entry

		ok, err := validateMapping(daemon, key, cid)
		if err!=nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func validateMapping(daemon *ipfsRequest, key string, cid string) (bool, error) {
	// load the content of IPFS object with CID = cid
	// compare the content header with the provided key
	// if equal, check the checksum in the key, return true if everything is correct
	return true, nil
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

func (ipfs IPFS) propageWrite(updatedMappingsIPNS string) error {
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
