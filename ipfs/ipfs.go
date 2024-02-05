// An IPFS implementation of the ImmutableStorage interface.
package ipfs

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// The maximum size in byte for an entry of <key>,<cid>;.
const mappingEntryMaxSize uint64 = uint64(blueprint.KeySize) + 64

// A mapping page is a file under the /mappings MFS directory named with "<page_number>.txt".
const mappingPageMaxSize uint64 = 512 * mappingEntryMaxSize

// The /nodes.txt file contains an array of IPNS name. nodestxtMaxSize is its maximum size in byte.
const nodestxtMaxSize uint64 = 640000

// cidProfile stores info associated with a message stored on IPFS.
type cidProfile struct {
	cid    string // cid is the IPFS CID of the stored message.
	source string // source is the IPNS record name of the /mappings directory of the node that stored the message.
}

// mappingEntry represents an entry in the mapping pages of a node, with the form of <key>,<cid>;
type mappingEntry struct {
	key blueprint.Key // key is the logical key associated with the stored message.
	cid string        // cid is the IPFS CID of the stored message.
}

// discoverProgressProfile represents the progress for this node to catch up existing records of a particular remote node.
type discoverProgressProfile struct {
	nextReadPage        uint   // nextReadPage stores the number of page this node has read.
	nextReadEntryOffset uint   // nextReadEntryOffset stores the number of entries in the current page this node has read.
	lastCommitCID       string // lastCommitCID stores the CID of the /mappings of the remote node, used to detect changes since previous read.
}

// IPFS implements blueprint.ImmutableStorage.
type IPFS struct {
	daemon              *ipfsClient          // daemon object is used to talk to the IPFS daemon.
	nodeDiscoveryClient *nodeDiscoveryClient // used to talk to the localhost NodeDiscovery service
	clusterServer       *ClusterServer
	grpcServer          *grpc.Server
	keyToCid            map[blueprint.Key]cidProfile // keyToCid stores
	nodesIPNS           string
	mappingsIPNS        string
	discoverProgress    map[string]discoverProgressProfile
}

func New(configFile string) (*IPFS, error) {
	// TODO: starting the workers
	e := errors.New("failed to initialize immutable-storage")

	config, err := parseConfigFile(configFile)
	if err != nil {
		log.Println("Error during parseConfigFile...")
		log.Println(err)
		return nil, e
	}
	//send HEAD request to <host>/api/v0/id
	err = checkDaemonAlive(config.Host)
	if err != nil {
		log.Println("Error during checkDaemonAlive...")
		log.Println(err)
		return nil, e
	}
	//create new IPFS struct
	var ipfs IPFS
	//set up ipfs daemon
	ipfs.daemon = newIPFSClient(config.Host, time.Duration(config.Timeout))
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
	newNodesIPNS, err := groupInitialization(&ipfs, oldNodesIPNS, mappingsIPNS, time.Duration(config.Timeout))
	if err != nil {
		log.Println("Error during groupInitialization...")
		log.Println(err)
		return nil, e
	}
	ipfs.nodesIPNS = newNodesIPNS

	// start grpc server
	// TODO: Check if this work...
	ipfs.grpcServer = grpc.NewServer()
	ipfs.clusterServer = NewClusterServer(&ipfs)
	protos.RegisterImmutableStorageClusterServer(ipfs.grpcServer, *ipfs.clusterServer)
	reflection.Register(ipfs.grpcServer)

	l, err := net.Listen("tcp", "127.0.0.1:3101")
	if err != nil {
		log.Println("failed to create gRPC server on port 3101")
		log.Println(err)
		return nil, e
	}
	if err := ipfs.grpcServer.Serve(l); err != nil {
		log.Println("failed to create gRPC server on port 3101")
		log.Println(err)
		return nil, e
	}
	log.Println("Successfully created gRPC server on port 3101")
	// start worker processes
	// TODO: create worker functions

	return &ipfs, nil
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
func lonelyInitialization(daemon *ipfsClient) (string, string, error) {
	// TODO: also consider the case when existing IPNS are provided for mapping IPNS and nodesIPNS
	// initialize public keys: nodes, mappings
	log.Println("Started lonely initialization...")
	keyNames, err := daemon.listKeys()
	if err != nil {
		return "", "", err
	}
	if !slices.Contains(keyNames, "mappings") {
		if err = daemon.generateKey("mappings"); err != nil {
			return "", "", err
		}
	}
	if !slices.Contains(keyNames, "nodes") {
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

	if err = daemon.appendStringToFile("/nodes.txt", mappingsIPNS+";", nodestxtMaxSize); err != nil {
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
	// TODO: error handlings

	log.Println("Start group initialization...")
	// On error retry every 10s, blocking
	memberIPs, err := ipfs.nodeDiscoveryClient.getNMembers(3)
	if err != nil {
		return myNodesIPNS, err
	}
	// On error no retry, non-blocking
	propagateWriteExternal(myMappingsIPNS, memberIPs)
	log.Println("Successfully propagated write")

	// On error retry every 10s, blocking
	validatedMappingsIPNSs, err := ipfs.initNodestxt()
	if err != nil {
		return myNodesIPNS, err
	}
	log.Println("Successfully validated all mappingsIPNSs: to be appended =", len(validatedMappingsIPNSs))

	// Append all mappingsIPNS to my own nodes.txt
	// See parser.go for expected format of nodes.txt. Append empty string at the end so that last element also has ending ';' after Join.
	formattedContent := strings.Join(append(validatedMappingsIPNSs, ""), ";")
	// On error retry 3 times, blocking
	ipfs.daemon.appendStringToFile("/nodes.txt", formattedContent, nodestxtMaxSize)

	// On error retry every 10s, blocking
	nodesCID, err := ipfs.daemon.getDirectoryCID("/nodes.txt")
	if err != nil {
		return "", err
	}
	// On error retry every 10s, blocking
	nodesIPNS, err := ipfs.daemon.publishIPNSPointer(nodesCID, "nodes")
	return nodesIPNS, err
}

func (ipfs *IPFS) Store(key blueprint.Key, message []byte) error {
	// TODO
	if !blueprint.ValidateKey(key, message) {
		return errors.New("invalid key")
	}
	// store the message on ipfs
	cid, err := ipfs.daemon.addFile(message)
	if err != nil {
		return err
	}
	// append the cid in the current mapping file
	// TODO handle offset, next page logic
	keyStr := string(key[:])
	ipfs.daemon.appendStringToFile("/mappings/???...", keyStr+","+cid+";", mappingPageMaxSize)

	// TODO propagate write
	// var peers []string = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	// TODO: 3 should be configurable
	for i := 0; i < 3; i++ {
		// addr := peers[rand.Intn(len(peers))]
		// grpcclient, ctx, cancel, err := createClusterClient(addr, ipfs.daemon.timeout) // TODO: daemon is not supposed to use in this way
		// if err != nil {
		// 	log.Println("error occured when Propagate Write.")
		// 	log.Println(err)
		// 	continue
		// }
		// _, err = grpcclient.PropagateWrite(*ctx, &protos.PropagateWriteRequest{})
		// defer (*cancel)()
		// if err != nil {
		// 	log.Println("error occured when Propagate Write.")
		// 	log.Println(err)
		// 	continue
		// }
	}
	return nil
}

func (ipfs *IPFS) Read(key blueprint.Key) ([]byte, error) {
	//TODO
	return []byte{}, nil
}

func (ipfs *IPFS) AvailableKeys() []blueprint.Key {
	result := make([]blueprint.Key, len(ipfs.keyToCid))
	for key := range ipfs.keyToCid {
		result = append(result, key)
	}
	return result
}

func (ipfs *IPFS) IsDiscovered(key blueprint.Key) bool {
	_, discovered := ipfs.keyToCid[key]
	return discovered
}

func (ipfs *IPFS) propagateWriteInternal(updatedMappingsIPNS string) error {
	// This function aims to speed up the latency for a remote node to discover a new local message stored on IPFS.
	// Elimination of this function should have no impact on the correctness of the algorithm.
	//
	// It performs the following actions:
	// 1) trigger the update IPNS routine, sync the local state regarding this IPNS with remote IPFS state.
	// 2) proceed if actual update is performed
	// 3) call NodeDiscovery.GetNMembers(fanout) to retrieve any N random members ip.
	// 4) call propagateWrite on each members in non-blocking manner
	//
	// Regarding step 4, non-blocking manner means that this function may return before it receives propagateWrite responses from the other members.
	// In other words, this function does not guarantee it will propagate the updatedMappingsIPNS to any node.

	// TODO: Step 1, 2

	// Step 3
	memberIPs, err := ipfs.nodeDiscoveryClient.getNMembers(3)
	if err != nil {
		return err
	}
	// Step 4
	propagateWriteExternal(updatedMappingsIPNS, memberIPs)

	return nil
}

func propagateWriteExternal(updatedMappingsIPNS string, memberIPs []string) {
	for _, memberIP := range memberIPs {
		go func(memberIP string) {
			// Propagate with best effort. Ignore errors.
			// Assume all peer Immutable Storage nodes run on port 3101.
			addr := memberIP + ":3101"
			cli, err := newClusterClient(addr, 3*time.Second)
			if err != nil {
				log.Println("[PropagateWrite]:Failed to establish connection with", addr, ". error:", err)
				return
			}
			// blocking for 3s in maximum
			err = cli.propagateWrite(updatedMappingsIPNS)
			if err != nil {
				log.Println("[PropagateWrite]:Failed to call propagateWrite endpoint on", addr, ". error:", err)
				return
			}
			log.Println("[PropagateWrite]:Success on", addr)
		}(memberIP)
	}
}

func (ipfs IPFS) initNodestxt() ([]string, error) {
	// This function allow a fresh Immutable Storage node instance to initialize its nodes.txt based on the current cluster state.
	// returns a slice of validated mappingsIPNS learnt from cluster peers.
	// returns nil on error
	//
	// It performs the following actions:
	// 1) query n nodes in the network to obtain their nodesIPNS
	// 2) resolve each nodesIPNS's contents as a slice of string, each element is a mappingIPNS
	// 3) remove duplicates among all n slices of string, combine them into one slice, and return.
	//
	// Error handling: error of any particular query node will be ignored.

	queryTargets, err := ipfs.nodeDiscoveryClient.getNMembers(3)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println("[initNodestxt]:Success call on getNMembers")

	var hashset sync.Map

	var wg sync.WaitGroup
	for _, memberIP := range queryTargets {
		wg.Add(1)
		go func(memberIP string) {
			addr := memberIP + ":3101"

			cli, err := newClusterClient(addr, 3*time.Second)
			if err != nil {
				log.Println("[GetNodestxt]:Failed to establish connection with", addr, ". error:", err)
				return
			}

			nodesIPNS, err := cli.getNodetxtIPNS()
			if err != nil {
				log.Println("[GetNodestxt]:Failed to call GetNodestxt endpoint on", addr, ". error:", err)
				return
			}

			externalNodestxt, err := openFileWithIPNS(ipfs.daemon, nodesIPNS)
			if err != nil {
				log.Println("[GetNodestxt]:Failed to open", externalNodestxt, "from", addr, ". error:", err)
				return
			}

			mappingsIPNSs, err := parseNodestxt(externalNodestxt)
			if err != nil {
				log.Println("[GetNodestxt]:Failed to parse", externalNodestxt, "from", addr, ". error:", err)
				return
			}

			for _, mappingsIPNS := range mappingsIPNSs {
				// Concurrent write operation, use thread-safe hashset
				hashset.Store(mappingsIPNS, true)
			}

			log.Println("[GetNodestxt]:Success on", addr)
			wg.Done()
		}(memberIP)
	}
	wg.Wait()
	log.Println("[initNodestxt]:Success on deduplicate all mappingsIPNS")

	// Validate each mappingsIPNS in hashset
	hashset.Range(func(key any, _ any) bool {
		// Type casting
		mappingsIPNS := key.(string)

		wg.Add(1)
		go func() {
			// Ignore the mappingsIPNS which causes error during validation
			ok, _ := ipfs.daemon.validateMappingsIPNS(mappingsIPNS, 10)
			// This may cause the hashset.Range method to revisit this key, but it is ok as this goroutine is idempotent
			hashset.Store(mappingsIPNS, ok)
			wg.Done()
		}()
		return true
	})
	wg.Wait()
	log.Println("[initNodestxt]:Success on validate all mappingsIPNS")

	validatedMappingsIPNSs := make([]string, 0)
	hashset.Range(func(key any, value any) bool {
		mappingsIPNS := key.(string)
		validness := value.(bool)
		if validness {
			validatedMappingsIPNSs = append(validatedMappingsIPNSs, mappingsIPNS)
		}
		return true
	})

	log.Println("[initNodestxt]:Success on composing the result")
	return validatedMappingsIPNSs, nil
}

func (ipfs IPFS) sync(key blueprint.Key) (string, string) {
	// return the corresponding CID and the IPNS name of the key if discovered
	if ipfs.IsDiscovered(key) {
		cidProfile := ipfs.keyToCid[key]
		return cidProfile.cid, cidProfile.source
	}
	return "", ""
}
