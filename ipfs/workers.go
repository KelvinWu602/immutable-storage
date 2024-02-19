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

// func (ipfs *IPFS) updateMappingIteration(ctx context.Context) {
// 	//TODO: allow termination of main goroutine to terminate the worker as well
// 	//TODO: consider retrying mechanism when discover a page or record fails due to network errors (using a DLQ)

// 	// for each round
// 	for {
// 		// obtain the current list of ipns from local nodes.txt
// 		ipnsRecords, err := parseNodesIPNS(ipfs.daemon, ipfs.nodesIPNS)
// 		if err != nil {
// 			// print the error, wait for 1s, continue
// 			log.Println(err)
// 			time.Sleep(1 * time.Second)
// 			continue
// 		}
// 		for _, ipns := range ipnsRecords {
// 			//check if we have created a profile for it
// 			progress, ok := ipfs.discoverProgress[ipns]
// 			if !ok {
// 				//first time seeing the ipns, initialize discover progress profile
// 				ipfs.discoverProgress[ipns] = discoverProgressProfile{
// 					nextReadPage:        0,
// 					nextReadEntryOffset: 0,
// 					lastCommitCID:       "",
// 				}
// 			}

// 			//check if the /mappings cid changed
// 			cid, err := ipfs.daemon.resolveIPNSPointer(ipns)
// 			if err != nil {
// 				log.Println(err)
// 				continue
// 			}
// 			if progress.lastCommitCID != cid {
// 				// get all mapping pages filenames and cids
// 				mappingPages, err := ipfs.daemon.getDAGLinks(cid)
// 				if err != nil {
// 					log.Println(err)
// 					continue
// 				}

// 				// while there is page haven't been read
// 				for progress.nextReadPage < uint(len(mappingPages)) {
// 					nextReadFileName := pageNumberToFileName(progress.nextReadPage)
// 					nextReadFileCid, ok := mappingPages[nextReadFileName]
// 					if !ok {
// 						// if a certain page is unfound, skip it
// 						log.Printf("Cannot find %s in IPNS %s\n", nextReadFileName, ipns)
// 						progress.nextReadPage++
// 						continue
// 					}
// 					entries, err := parseMappingsCID(ipfs.daemon, nextReadFileCid)
// 					if err != nil {
// 						// if a certain page content is invalid, skip it
// 						log.Println(err)
// 						progress.nextReadPage++
// 						continue
// 					}
// 					for row := progress.nextReadEntryOffset; row < uint(len(entries)); row++ {
// 						entry := entries[row]
// 						ok, err := validateMapping(ipfs.daemon, entry.key, entry.cid)
// 						if err != nil || !ok {
// 							log.Println("Found an invalid entry")
// 							continue
// 						}
// 						// add the entry into keyToCid
// 						ipfs.keyToCid[entry.key] = cidProfile{cid: entry.cid, source: ipns}
// 					}
// 					// update the nextReadEntryOffset
// 					progress.nextReadEntryOffset = uint(len(entries))
// 					if progress.nextReadEntryOffset >= 512 {
// 						progress.nextReadEntryOffset = 0
// 						progress.nextReadPage++
// 					}
// 				}
// 				// update the last commit CID
// 				progress.lastCommitCID = cid
// 			}
// 			ipfs.discoverProgress[ipns] = progress
// 		}
// 	}
// }
