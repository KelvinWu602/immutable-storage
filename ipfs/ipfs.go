// An IPFS implementation of the ImmutableStorage interface.
package ipfs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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
	cid string        // cid is the IPFS CID of the stored mes sage.
}

// discoverProgressProfile represents the progress for this node to catch up existing records of a particular remote node.
type discoverProgressProfile struct {
	nextReadPage  uint   // nextReadPage stores the number of page this node has read.
	lastCommitCID string // lastCommitCID stores the CID of the /mappings of the remote node, used to detect changes since previous read.
}

// IPFS implements blueprint.ImmutableStorage.
type IPFS struct {
	ipfsClient          *ipfsClient                  // used to talk to the IPFS daemon.
	nodeDiscoveryClient *nodeDiscoveryClient         // used to talk to the localhost NodeDiscovery service
	clusterServer       *grpc.Server                 // gRPC server serving other ImmutableStorage-IPFS nodes
	keyToCid            map[blueprint.Key]cidProfile // keyToCid stores
	nodesIPNS           string                       // the IPNS name pointing to the /nodes.txt
	mappingsIPNS        string                       // the IPNS name pointing to the /mappings/
	numOfFullyUsedPages int                          // number of fully used files under /mappings/.
	discoverProgress    map[string]discoverProgressProfile
}

func New(configFilePath string) (*IPFS, error) {
	// Use default config on any error
	// config := loadConfig(configFilePath)
	// Create new IPFS struct
	var ipfs IPFS
	// Connect to all dependencies
	ipfs.initDependencies()

	// Lonely Initialization. On error retry every 10s.
	mappingsIPNS, localNodesIPNS, err := ipfs.lonelyInit()
	for err != nil {
		log.Println("Error during lonelyInit. Retry after 10s.")
		log.Println(err)
		time.Sleep(10 * time.Second)
		mappingsIPNS, localNodesIPNS, err = ipfs.lonelyInit()
	}
	ipfs.mappingsIPNS = mappingsIPNS

	// Group Initialization. On error ignore and proceed.
	newNodesIPNS, err := ipfs.groupInit(localNodesIPNS, mappingsIPNS)
	if err != nil {
		log.Println("Error during groupInit. groupInit has no effect.")
	}
	ipfs.nodesIPNS = newNodesIPNS

	// Create gRPC server serving other ImmutableStorage-IPFS nodes. Blocking.
	ipfs.initClusterServer()

	// TODO: create worker functions

	return &ipfs, nil
}

// Load config file if specified. On error, proceed with default config.
func loadConfig(configFilePath string) config {
	// Open the file
	file, _ := openFileWithPath(configFilePath)
	// Parse the file
	config, _ := parseConfig(file)
	return config
}

// initDependencies creates connections with the dependencies of ImmutableStorage-IPFS, including
// 1) NodeDiscovery grpc server:  localhost:3200
// 2) IPFS daemon http server  :  localhost:5001
//
// This is a blocking procedure and it will retry infinitely on error
// TODO: add support for configurable ports
func (ipfs *IPFS) initDependencies() {
	// Establish connection with IPFS daemon
	ipfs.ipfsClient = newIPFSClient("localhost:5001", 10*time.Second)
	log.Println("Connected to localhost:5001 -- IPFS daemon")
	// Establish connection with NodeDiscovery grpc server
	var err error
	ipfs.nodeDiscoveryClient, err = newNodeDiscoveryClient("localhost:3200", 3*time.Second)
	for err != nil {
		// On error, retry every 1s
		time.Sleep(time.Second)
		ipfs.nodeDiscoveryClient, err = newNodeDiscoveryClient("localhost:3200", 3*time.Second)
	}
	log.Println("Connected to localhost:3200 -- NodeDiscovery")
}

// lonelyInit returns mappingsIPNS, nodesIPNS, err
func (ipfs *IPFS) lonelyInit() (string, string, error) {
	// TODO: also consider the case when existing IPNS are provided for mapping IPNS and nodesIPNS
	// initialize public keys: nodes, mappings
	log.Println("Started lonely initialization...")
	keyNames, err := ipfs.ipfsClient.listKeys()
	if err != nil {
		return "", "", err
	}
	if !slices.Contains(keyNames, "mappings") {
		if err = ipfs.ipfsClient.generateKey("mappings"); err != nil {
			return "", "", err
		}
	}
	if !slices.Contains(keyNames, "nodes") {
		if err = ipfs.ipfsClient.generateKey("nodes"); err != nil {
			return "", "", err
		}
	}
	log.Println("Successfully created keys")
	// create MFS directory /mappings, ignore error for file already exists
	if err = ipfs.ipfsClient.createDirectory("/mappings"); err != nil && err.Error() != "files/mkdir: file already exists" {
		return "", "", err
	}
	// publish IPNS name for directory /mappings
	mappingsCID, err := ipfs.ipfsClient.getDirectoryCID("/mappings")
	if err != nil {
		return "", "", err
	}
	mappingsIPNS, err := ipfs.ipfsClient.publishIPNSPointer(mappingsCID, "mappings")
	if err != nil {
		return "", "", err
	}
	log.Println("Successfully published mappingsIPNS:", mappingsIPNS)
	// create and append mappingIPNS in file /nodes.txt with comma delimiter
	_, err = ipfs.ipfsClient.getDirectoryCID("/nodes.txt")
	if err == nil {
		// if file exists, remove it
		if err2 := ipfs.ipfsClient.removeFile("/nodes.txt", false); err2 != nil {
			return "", "", err2
		}
	} else if err.Error() != "files/stat: file does not exist" {
		// if any error except file does not exist, return it
		return "", "", err
	}
	log.Println("Successfully cleaned old /nodes.txt")

	if err = ipfs.ipfsClient.appendStringToFile("/nodes.txt", mappingsIPNS+";", nodestxtMaxSize); err != nil {
		return "", "", err
	}
	// publish IPNS name for file /nodes.txt
	nodesCID, err := ipfs.ipfsClient.getDirectoryCID("/nodes.txt")
	if err != nil {
		return "", "", err
	}
	nodesIPNS, err := ipfs.ipfsClient.publishIPNSPointer(nodesCID, "nodes")
	if err != nil {
		return "", "", err
	}
	log.Println("Successfully published nodesIPNS:", nodesIPNS)
	return nodesIPNS, mappingsIPNS, nil
}

// groupInit is the second phase of node initialization. It is an optional phase and it's absence not affect the algorithm correctness.
//
// It returns the most updated IPNS name pointing at nodes.txt. Returns existing IPNS name for nodes.txt on error.
//
// It performs the following actions:
// 1) advertise my own mappingsIPNS to other nodes, allow them to append to their nodes.txt
// 2) query 3 other nodes and aggregate their nodes.txt
// 3) append the aggregated results to my own nodes.txt
// 4) update the IPNS pointer to point at the latest nodes.txt
//
// Error semantics: return errors if this function fails before Step 3, which could be safely ignored as this function has not yet mutate any states.
// This function will block the goroutine if it fails at or after Step 3, and keep retrying until success.
func (ipfs *IPFS) groupInit(myNodesIPNS string, myMappingsIPNS string) (string, error) {
	// On error retry every 10s, blocking
	memberIPs, err := ipfs.nodeDiscoveryClient.getNMembers(3)
	if err != nil {
		log.Println("[groupInit]:Failed call nodeDiscoveryClient.getNMembers. Skip groupInit.")
		log.Println(err)
		return myNodesIPNS, err
	}
	log.Println("[groupInit]:Success call nodeDiscoveryClient.getNMembers")
	// On error no retry, non-blocking
	propagateWriteExternal(myMappingsIPNS, memberIPs)
	log.Println("[groupInit]:Success call propagateWriteExternal")

	// On error retry every 10s, blocking
	validatedMappingsIPNSs, err := ipfs.initNodestxt()
	if err != nil {
		log.Println("[groupInit]:Failed call ipfs.initNodestxt. Skip groupInit.")
		log.Println(err)
		return myNodesIPNS, err
	}
	log.Println("[groupInit]:Success call ipfs.initNodestxt")

	// Append all mappingsIPNS to my own nodes.txt
	// See parser.go for expected format of nodes.txt. Append empty string at the end so that last element also has ending ';' after Join.
	formattedContent := strings.Join(append(validatedMappingsIPNSs, ""), ";")
	// On error retry 3 times, blocking
	err = ipfs.ipfsClient.appendStringToFile("/nodes.txt", formattedContent, nodestxtMaxSize)
	for retryCount, maxRetry := 0, 3; err != nil && retryCount < maxRetry; retryCount++ {
		log.Println("[groupInit]:Failed call ipfsClient.appendStringToFile. Retry = (", retryCount, "/", maxRetry, ")")
		log.Println(err)
		err = ipfs.ipfsClient.appendStringToFile("/nodes.txt", formattedContent, nodestxtMaxSize)
	}
	if err != nil {
		log.Println("[groupInit]:Failed call ipfsClient.appendStringToFile. Skip groupInit.")
		log.Println(err)
		return myNodesIPNS, err
	}
	log.Println("[groupInit]:Success call ipfsClient.appendStringToFile")

	// On error retry every 10s, blocking
	nodesCID, err := ipfs.ipfsClient.getDirectoryCID("/nodes.txt")
	for err != nil {
		log.Println("[groupInit]:Failed call ipfsClient.getDirectoryCID. Retry after 10s.")
		time.Sleep(10 * time.Second)
		nodesCID, err = ipfs.ipfsClient.getDirectoryCID("/nodes.txt")
	}
	log.Println("[groupInit]:Success call ipfsClient.getDirectoryCID")

	// On error retry every 10s, blocking
	nodesIPNS, err := ipfs.ipfsClient.publishIPNSPointer(nodesCID, "nodes")
	for err != nil {
		log.Println("[groupInit]:Failed call ipfsClient.publishIPNSPointer. Retry after 10s.")
		time.Sleep(10 * time.Second)
		nodesIPNS, err = ipfs.ipfsClient.publishIPNSPointer(nodesCID, "nodes")
	}
	log.Println("[groupInit]:Success call ipfsClient.publishIPNSPointer")

	return nodesIPNS, err
}

// Create a gRPC server to serve requests from other ImmutableStorage-IPFS nodes.
// Blocking.
func (ipfs *IPFS) initClusterServer() {
	ipfs.clusterServer = grpc.NewServer()
	protos.RegisterImmutableStorageClusterServer(ipfs.clusterServer, NewClusterServer(ipfs))
	reflection.Register(ipfs.clusterServer)

	// On error retry every 5s. Blocking.
	listener, err := net.Listen("tcp", "127.0.0.1:3101")
	for err != nil {
		log.Println("[initClusterServer]:Failed call net.Listen. Retry after 5s.")
		log.Println(err)
		time.Sleep(5 * time.Second)
		listener, err = net.Listen("tcp", "127.0.0.1:3101")
	}
	log.Println("[initClusterServer]:Success call net.Listen")

	// On error retry every 5s. Blocking.
	err = ipfs.clusterServer.Serve(listener)
	for err != nil {
		log.Println("[initClusterServer]:Failed call grpcServer.Serve. Retry after 5s.")
		log.Println(err)
		time.Sleep(5 * time.Second)
		err = ipfs.clusterServer.Serve(listener)
	}
	log.Println("[initClusterServer]:Success call grpcServer.Serve")
}

// Store the given key and message to the IPFS. The key and value are considered immutable once this function has returned.
//
// This function will perform the following actions:
// 1) Validate the input key and message.
// 2) Store the message on IPFS, obtained a CID.
// 3) Store the Key-CID pair on the last mappings page under /mappings.
// 4) Update the IPNS pointer to point at your latest /mappings/ CID.
// 5) Call propagateWriteExternal to notify other nodes about this store event.
// 6) Update local discoverProgressProfile
//
// Error semantics: on any failure will return error. Caller is suggested to retry the action.
func (ipfs *IPFS) Store(key blueprint.Key, message []byte) error {
	// Step 1
	if !blueprint.ValidateKey(key, message) {
		return errors.New("invalid key")
	}
	// Step 2
	cid, err := ipfs.ipfsClient.addFile(message)
	if err != nil {
		return err
	}
	// Step 3
	entry := fmt.Sprintf("%s,%s;", string(key[:]), cid)
	mappingPagePath := fmt.Sprintf("/mappings/%s", mappingsPageNumberToName(ipfs.numOfFullyUsedPages))
	// appendStringToFile will create the file when it does not exists
	err = ipfs.ipfsClient.appendStringToFile(mappingPagePath, entry, mappingPageMaxSize)
	for errors.Is(err, errExceedFileSizeLimit) {
		// If the current page is full, move on to next page
		ipfs.numOfFullyUsedPages++
		mappingPagePath = fmt.Sprintf("/mappings/%s", mappingsPageNumberToName(ipfs.numOfFullyUsedPages))
		err = ipfs.ipfsClient.appendStringToFile(mappingPagePath, entry, mappingPageMaxSize)
	}
	if err != nil {
		return err
	}
	// Step 4
	mappingsCID, err := ipfs.ipfsClient.getDirectoryCID("/mappings")
	if err != nil {
		return err
	}
	mappingsIPNS, err := ipfs.ipfsClient.publishIPNSPointer(mappingsCID, "nodes")
	if err != nil {
		return err
	}
	// Step 5
	membersIP, err := ipfs.nodeDiscoveryClient.getNMembers(3)
	if err == nil {
		// This step is optional.
		propagateWriteExternal(mappingsIPNS, membersIP)
	}
	// Step 6
	ipfs.keyToCid[key] = cidProfile{
		cid:    cid,
		source: mappingsIPNS,
	}
	trackingProgress := ipfs.discoverProgress[mappingsIPNS]
	trackingProgress.lastCommitCID = mappingsCID
	ipfs.discoverProgress[mappingsIPNS] = trackingProgress

	return nil
}

// Read returns the corresponding message payload with a given key.
//
// This function performs the following actions:
// 1) check if the unknown key has been discovered, if yes, return message immediately.
// 2) if the key is undiscovered, start a job with timeout, repeatedly call sync on other nodes.
//
// Error handling: for errors happened when request sync, ignore and proceed. For any other error, return it.
func (ipfs *IPFS) Read(key blueprint.Key) ([]byte, error) {
	var cid string
	if ipfs.IsDiscovered(key) {
		// retrieve CID from local cache
		cid = ipfs.keyToCid[key].cid
	} else {
		// retrieve CID from peer nodes
		memberIPs, err := ipfs.nodeDiscoveryClient.getMembers()
		if err != nil {
			return nil, err
		}
		found, cid, mappingsIPNS := ipfs.requestSyncTimeoutJob(key, memberIPs, 3*time.Second)
		if found {
			ipfs.keyToCid[key] = cidProfile{
				cid:    cid,
				source: mappingsIPNS,
			}
		}
	}
	// retrieve message with CID
	file, err := openFileWithCID(ipfs.ipfsClient, cid)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	message, err := parseMessage(file)
	if err != nil {
		return nil, err
	}
	return message.payload, nil
}

// requestSyncTimeoutJob will repeatedly call Sync on other ImmutableStorage-IPFS nodes for the CID corresponding to input key.
// The calls are made in sequence, errors will be neglected.
func (ipfs *IPFS) requestSyncTimeoutJob(key blueprint.Key, memberIPs []string, timeout time.Duration) (bool, string, string) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	for _, memberIP := range memberIPs {
		addr := memberIP + ":3101"

		clusterClient, err := newClusterClient(addr, timeout)
		if err != nil {
			log.Println(err)
			continue
		}
		found, cid, mappingsIPNS, err := clusterClient.syncDeadline(key, ctx)
		if err != nil {
			log.Println(err)
			if errors.Is(err, context.DeadlineExceeded) {
				return false, "", ""
			} else {
				continue
			}
		}
		if found {
			return true, cid, mappingsIPNS
		}
		clusterClient.closeClusterClient()
	}
	return false, "", ""
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

func (ipfs *IPFS) checkPullIsRequired(mappingsIPNS string) (bool, error) {
	trackingProgress, ok := ipfs.discoverProgress[mappingsIPNS]
	if !ok {
		return true, nil
	}
	remoteCID, err := ipfs.ipfsClient.resolveIPNSPointer(mappingsIPNS)
	if err != nil {
		return false, err
	}
	return remoteCID != trackingProgress.lastCommitCID, nil
}

// pullRemoteState pulls all new key-cid entries under updatedMappingsIPNS and store them in local cache.
// Assumed this updatedMappingsIPNS is validated.
func (ipfs *IPFS) pullRemoteState(updatedMappingsIPNS string) error {
	// Return concrete or "zero" value trackingProgress
	trackingProgress := ipfs.discoverProgress[updatedMappingsIPNS]
	// Save the current progress
	defer func() {
		ipfs.discoverProgress[updatedMappingsIPNS] = trackingProgress
	}()
	remoteFolderCID, err := ipfs.ipfsClient.resolveIPNSPointer(updatedMappingsIPNS)
	if err != nil {
		return err
	}
	fileNameToCID, err := ipfs.ipfsClient.getDAGLinks(remoteFolderCID)
	if err != nil {
		return err
	}

	totalNumOfPages := uint(len(fileNameToCID))
	for ; trackingProgress.nextReadPage < totalNumOfPages; trackingProgress.nextReadPage++ {
		pageName := mappingsPageNumberToName(int(trackingProgress.nextReadPage))
		pageCID := fileNameToCID[pageName]

		file, err := openFileWithCID(ipfs.ipfsClient, pageCID)
		if err != nil {
			return err
		}
		mappings, err := parseMappings(file)
		if err != nil {
			return err
		}
		for _, entry := range mappings {
			// Skip if this key has already discovered
			if _, ok := ipfs.keyToCid[entry.key]; ok {
				continue
			}
			ipfs.keyToCid[entry.key] = cidProfile{
				cid:    entry.cid,
				source: updatedMappingsIPNS,
			}
		}
	}
	trackingProgress.lastCommitCID = remoteFolderCID
	return nil
}

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
func (ipfs *IPFS) propagateWriteInternal(updatedMappingsIPNS string) error {
	// Step 1, 2
	requireUpdate, err := ipfs.checkPullIsRequired(updatedMappingsIPNS)
	if err != nil {
		return err
	}
	if !requireUpdate {
		return nil
	}
	err = ipfs.pullRemoteState(updatedMappingsIPNS)
	if err != nil {
		return err
	}

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
func (ipfs IPFS) initNodestxt() ([]string, error) {

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

			externalNodestxt, err := openFileWithIPNS(ipfs.ipfsClient, nodesIPNS)
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
			ok, _ := ipfs.ipfsClient.validateMappingsIPNS(mappingsIPNS, 10)
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

// return the corresponding CID and the IPNS name of the key if discovered
func (ipfs IPFS) sync(key blueprint.Key) (string, string) {
	if ipfs.IsDiscovered(key) {
		cidProfile := ipfs.keyToCid[key]
		return cidProfile.cid, cidProfile.source
	}
	return "", ""
}
