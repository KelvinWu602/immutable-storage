package ipfs

import (
	"bufio"
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
	"github.com/go-yaml/yaml"
	"google.golang.org/grpc"
)

// A mapping page is a file under the /mappings MFS directory named with "<page_number>.txt". Each mapping page contains 512 entries at maximum.
const mappingPageMaxSize uint64 = 512

// The /nodes.txt file contains an array of IPNS name. nodestxtMaxSize is its maximum size in byte.
const nodestxtMaxSize uint64 = 640000

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
	host    string `yaml:"host"`
	timeout int    `yaml:"timeout"`
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
	newNodesIPNS, err := groupInitialization(&ipfs, oldNodesIPNS, mappingsIPNS, time.Duration(config.timeout))
	if err != nil {
		log.Println("Error during groupInitialization...")
		log.Println(err)
		return nil, e
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

// lonelyInitialization returns mappingsIPNS, nodesIPNS, err
func lonelyInitialization(daemon *ipfsRequest) (string, string, error) {
	// TODO: also consider the case when existing IPNS are provided for mapping IPNS and nodesIPNS
	// initialize public keys: nodes, mappings
	log.Println("Started lonely initialization...")
	keyNames, err := daemon.listKeys()
	if err != nil {
		return "", "", err
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
			return "", "", err
		}
	}
	if initNodes {
		if err = daemon.generateKey("nodes"); err != nil {
			return "", "", err
		}
	}
	log.Println("Successfully created keys")
	// create MFS directory /mappings, ignore error for file already exists
	if err = daemon.createDirectory("/mappings"); err != nil && err.Error() != "files/mkdir: file already exists" {
		return "", "", err
	}
	// publish IPNS name for directory /mappings
	mappingsCID, err := daemon.getDirectoryCID("/mappings")
	if err != nil {
		return "", "", err
	}
	mappingsIPNS, err := daemon.publishIPNSPointer(mappingsCID, "mappings")
	if err != nil {
		return "", "", err
	}
	log.Println("Successfully published mappingsIPNS:", mappingsIPNS)
	// create and append mappingIPNS in file /nodes.txt with comma delimiter
	_, err = daemon.getDirectoryCID("/nodes.txt")
	if err == nil {
		// if file exists, remove it
		if err2 := daemon.removeFile("/nodes.txt", false); err2 != nil {
			return "", "", err2
		}
	} else if err.Error() != "files/stat: file does not exist" {
		// if any error except file does not exist, return it
		return "", "", err
	}
	log.Println("Successfully cleaned old /nodes.txt")

	if err = daemon.appendStringToFile("/nodes.txt", mappingsIPNS+",", nodestxtMaxSize); err != nil {
		return "", "", err
	}
	// publish IPNS name for file /nodes.txt
	nodesCID, err := daemon.getDirectoryCID("/nodes.txt")
	if err != nil {
		return "", "", err
	}
	nodesIPNS, err := daemon.publishIPNSPointer(nodesCID, "nodes")
	if err != nil {
		return "", "", err
	}
	log.Println("Successfully published nodesIPNS:", nodesIPNS)
	return nodesIPNS, mappingsIPNS, nil
}

// groupInitialization returns nodesIPNS, err
func groupInitialization(ipfs *IPFS, myNodesIPNS string, myMappingsIPNS string, timeout time.Duration) (string, error) {
	log.Println("Start group initialization...")
	if err := ipfs.propagateWrite(myMappingsIPNS, timeout); err != nil {
		return myNodesIPNS, err
	}
	log.Println("Successfully propagated write")

	externalNodesIPNS, err := ipfs.initNodestxt(timeout)
	if err != nil {
		return myNodesIPNS, err
	}
	log.Println("Successfully requested for nodesIPNS:", externalNodesIPNS)

	// get the content for each nodes.txt
	union := make(map[string]bool)
	seenRecords, err := parseNodesIPNS(ipfs.daemon, myNodesIPNS)
	if err != nil {
		return myNodesIPNS, err
	}
	for _, record := range seenRecords {
		union[record] = true
	}

	for _, nodesIPNS := range externalNodesIPNS {
		if len(nodesIPNS) == 0 {
			continue
		}
		content, err := parseNodesIPNS(ipfs.daemon, nodesIPNS)
		if err != nil {
			return myNodesIPNS, err
		}
		// for each unseen record, union[record] = false
		for _, record := range content {
			if _, ok := union[record]; !ok {
				union[record] = false
			}
		}
	}

	// for each unseen record, validate, append to your own nodes.txt file if validated, discard if not
	var wg sync.WaitGroup
	for record, seen := range union {
		if !seen {
			wg.Add(1)
			go func() {
				// check any random 1 record in the mappings
				_, err := validateMappingsIPNS(ipfs.daemon, record, 1)
				if err != nil {
					wg.Done()
					return
				}
				//append the record to my own nodes.txt
				if err := ipfs.daemon.appendStringToFile("/nodes.txt", record, nodestxtMaxSize); err != nil {
					wg.Done()
					return
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()

	// update the IPNS name for your nodes.txt
	nodesCID, err := ipfs.daemon.getDirectoryCID("/nodes.txt")
	if err != nil {
		return "", err
	}
	nodesIPNS, err := ipfs.daemon.publishIPNSPointer(nodesCID, "nodes")
	return nodesIPNS, err
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

	r := bufio.NewReader(file)
	entry, err := r.ReadSlice(';')
	mappingsIPNSs := make([]string, 1)
	for err == nil {
		mappingIPNS := string(entry[:len(entry)-1])
		mappingsIPNSs = append(mappingsIPNSs, mappingIPNS)
		entry, err = r.ReadSlice(';')
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	return mappingsIPNSs, nil
}

// parseMappingsIPNS takes the CID of a mapping page with at most 512 <key>,<cid>; records and convert it into a map[Key]Cid
func parseMappingsIPNS(daemon *ipfsRequest, mappingCID string) (map[blueprint.Key]string, error) {
	file, err := daemon.readFileWithCID(mappingCID)
	if err != nil {
		return nil, err
	}

	r := bufio.NewReader(file)
	entry, err := r.ReadSlice(';')
	result := make(map[blueprint.Key]string)
	for err == nil {
		key := blueprint.Key(entry[:blueprint.KeySize])
		cid := string(entry[blueprint.KeySize+1:])
		result[key] = cid
		entry, err = r.ReadSlice(';')
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	return result, nil
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

		entries, err := parseMappingsIPNS(daemon, filesNameToCid[fileName])
		if err != nil {
			return false, err
		}

		target := rand.Intn(len(entries))
		for key, cid := range entries {
			if target == 0 {
				ok, err := validateMapping(daemon, key, cid)
				if err != nil {
					return false, err
				}
				return ok, nil
			}
			target--
		}
	}
	return true, nil
}

func validateMapping(daemon *ipfsRequest, key blueprint.Key, cid string) (bool, error) {
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

func (ipfs IPFS) propagateWrite(updatedMappingsIPNS string, timeout time.Duration) error {
	// gossip broadcast updatedMappingsIPNS to 3 random peers
	// TODO: using node-discovery endpoints instead
	var peers []string = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	port := ":3101"

	// grpc cluster server handler will spawn 3 goroutines to propagate call, but it will not wait for them to finish
	// current implementation does not guarantee any node to receive the call
	for i := 0; i < int(math.Min(3, float64(len(peers)))); i++ {
		go func() {
			// randomly choose one peer
			addr := peers[rand.Intn(len(peers))] + port
			//	make a grpc call to peer for the propagate_write grpc endpoint
			var opts []grpc.DialOption
			conn, err := grpc.Dial(addr, opts...)
			if err != nil {
				log.Println("failed to connect to", addr)
			}
			defer conn.Close()
			grpcclient := protos.NewImmutableStorageClusterClient(conn)
			// Contact the server and print out its response.
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			_, err = grpcclient.PropagateWrite(ctx, &protos.PropagateWriteRequest{MappingsIPNS: updatedMappingsIPNS})
			if err != nil {
				log.Println("failed to propagate_write to", addr)
			}
			log.Println("Successfully propagate write to", addr)
		}()
	}
	return nil
}

func (ipfs IPFS) initNodestxt(timeout time.Duration) ([]string, error) {
	// gossip broadcast updatedMappingsIPNS to 3 random peers
	// TODO: using node-discovery endpoints instead
	var peers []string = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	port := ":3101"
	gossipSpan := int(math.Min(3, float64(len(peers))))
	externalNodesIPNS := make([]string, gossipSpan)

	var wg sync.WaitGroup
	for i := 0; i < gossipSpan; i++ {
		wg.Add(1)
		go func(jobID int) {
			// randomly choose one peer
			addr := peers[rand.Intn(len(peers))] + port
			//	make a grpc call to peer for the propagate_write grpc endpoint
			var opts []grpc.DialOption
			conn, err := grpc.Dial(addr, opts...)
			if err != nil {
				log.Println("failed to connect to", addr)
			}
			defer conn.Close()
			grpcclient := protos.NewImmutableStorageClusterClient(conn)
			// Contact the server and print out its response.
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			res, err := grpcclient.GetNodetxtIPNS(ctx, &protos.GetNodetxtIPNSRequest{})
			if err != nil {
				log.Println("failed to propagate_write to", addr)
			} else {
				externalNodesIPNS[i] = res.NodetxtIPNS
				log.Println("Successfully propagate write to", addr)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	for _, nodesIPNS := range externalNodesIPNS {
		if len(nodesIPNS) > 0 {
			return externalNodesIPNS, nil
		}
	}
	return nil, errors.New("all nodes failed to response")
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
