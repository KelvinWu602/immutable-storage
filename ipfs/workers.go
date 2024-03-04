package ipfs

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

func worker(ctx context.Context, task func()) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Gracefully stopped updateIndexDirectoryWorker.")
			return
		default:
			// Perform task once
			time.Sleep(10 * time.Second)
			task()
		}
	}
}

// updateIndexDirectoryIteration will query a peer node and merge its nodes.txt with your local nodes.txt.
// Error handling: on any error, rollback the changes and return.
//
// It performs the following actions:
// 1) Choose a random peer node.
// 2) Call the GetNodestxt.
// 3) Parse the nodes.txt obtained from peer, validate its content.
// 4) Parse the local nodes.txt.
// 5) Merge the local and remote nodes.txt.
// 6) Append the new found records in local nodes.txt.
func (ipfs *IPFS) updateIndexDirectoryIteration() {
	// Step 1
	memberIP, err := ipfs.nodeDiscoveryClient.getNMembers(1)
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed call nodeDiscoveryClient.getNMembers. Skip this iteration.")
		log.Println(err)
		return
	}
	addr := memberIP[0] + ":3101"
	clusterClient, err := newClusterClient(addr, 3*time.Second)
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed create new clusterClient. Skip this iteration.")
		log.Println(err)
		return
	}
	// Step 2
	externalNodeIPNS, err := clusterClient.getNodetxtIPNS()
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed call clusterClient.getNodetxtIPNS. Skip this iteration.")
		log.Println(err)
		return
	}
	// Step 3
	externalNodestxt, err := openFileWithIPNS(ipfs.ipfsClient, externalNodeIPNS)
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed to open", externalNodestxt, "from", addr, ". error:", err)
		return
	}
	defer externalNodestxt.Close()

	mappingsIPNSs, err := parseNodestxt(externalNodestxt)
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed to parse", externalNodestxt, "from", addr, ". error:", err)
		return
	}

	var validationResults map[string]bool

	var waitValidation sync.WaitGroup
	for _, mappingsIPNS := range mappingsIPNSs {
		validationResults[mappingsIPNS] = true
		waitValidation.Add(1)
		go func() {
			// Ignore the mappingsIPNS which causes error during validation
			ok, _ := ipfs.ipfsClient.validateMappingsIPNS(mappingsIPNS, 10)
			validationResults[mappingsIPNS] = ok
			waitValidation.Done()
		}()
	}
	waitValidation.Wait()

	// Step 4
	localNodestxt, err := openFileWithIPNS(ipfs.ipfsClient, ipfs.nodesIPNS)
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed to open", ipfs.nodesIPNS, "from localhost", ". error:", err)
		return
	}
	defer localNodestxt.Close()

	localMappingsIPNSs, err := parseNodestxt(localNodestxt)
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed to parse", ipfs.nodesIPNS, "from localhost", ". error:", err)
		return
	}

	// Step 5
	var existingRecords map[string]bool
	for _, localMappingsIPNS := range localMappingsIPNSs {
		existingRecords[localMappingsIPNS] = true
	}

	var newValidRecords []string

	for mappingsIPNS, valid := range validationResults {
		// Skip invalid or existing records
		if _, alreadyExists := existingRecords[mappingsIPNS]; valid && !alreadyExists {
			// Append the record into the array
			newValidRecords = append(newValidRecords, mappingsIPNS)
		}
	}

	formattedContent := strings.Join(append(newValidRecords, ""), ";")

	// Step 6
	// On error retry 3 times, blocking
	err = ipfs.ipfsClient.appendStringToFile("/nodes.txt", formattedContent, nodestxtMaxSize)
	for retryCount, maxRetry := 0, 3; err != nil && retryCount < maxRetry; retryCount++ {
		log.Println("[updateIndexDirectoryIteration]:Failed call ipfsClient.appendStringToFile. Retry = (", retryCount, "/", maxRetry, ")")
		log.Println(err)
		err = ipfs.ipfsClient.appendStringToFile("/nodes.txt", formattedContent, nodestxtMaxSize)
	}
	if err != nil {
		log.Println("[updateIndexDirectoryIteration]:Failed call ipfsClient.appendStringToFile. Skip this iteration.")
		log.Println(err)
		return
	}
	log.Println("[updateIndexDirectoryIteration]:Merged", externalNodeIPNS)
}

// updateMappingIteration will get a copy of the local nodes.txt, and sync the state for this node with the latest remote state for each mappingsIPNS.
// Error handling: on global error, log it and return. on record-level error, log it and continue
//
// It performs the following actions:
// 1) Get local nodes.txt, on error log it and return.
// 2) Parse nodes.txt, on error log it and return.
// 3) checkPullIsRequired, on error log it and skip 4.
// 4) pullRemoteState, on error log it and continue.
//
// TODO: consider validation, new records could be malicious.
func (ipfs *IPFS) updateMappingIteration() {
	// Step 1
	nodestxt, err := openFileWithIPNS(ipfs.ipfsClient, ipfs.nodesIPNS)
	if err != nil {
		log.Println("[updateMappingIteration]:Failed to open", ipfs.nodesIPNS, "from localhost. error:", err)
		return
	}
	defer nodestxt.Close()

	// Step 2
	mappingsIPNSs, err := parseNodestxt(nodestxt)
	if err != nil {
		log.Println("[updateMappingIteration]:Failed to parse local nodes.txt from localhost. error:", err)
		return
	}

	for _, mappingsIPNS := range mappingsIPNSs {
		// Step 3
		pullRequired, err := ipfs.checkPullIsRequired(mappingsIPNS)
		if err != nil {
			log.Println("[updateMappingIteration]:Failed to resolve", mappingsIPNS, ". error:", err)
			continue
		}

		if !pullRequired {
			continue
		}
		// Step 4
		err = ipfs.pullRemoteState(mappingsIPNS)
		if err != nil {
			// this marks the pull operation is partially completed in a valid state, can be rerunnable.
			log.Println("[updateMappingIteration]:Failed to pull", mappingsIPNS, ". error:", err)
			continue
		}
	}
	log.Println("[updateMappingIteration]:Updated")
}
