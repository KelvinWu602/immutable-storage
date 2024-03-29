package ipfs

import (
	"bufio"
	"errors"
	"log"
	"math/rand"

	"github.com/KelvinWu602/immutable-storage/blueprint"
)

func (daemon *ipfsClient) validateMapping(key blueprint.Key, cid string) (bool, error) {
	// load the content of IPFS object with CID = cid
	// compare the content header with the provided key
	// if equal, check the checksum in the key, return true if everything is correct
	//
	// About error handling mechanism, IPFS cannot differentiate whether a valid CID is:
	// 1) non-exist and created by Attacker, or
	// 2) exist but located too far away
	// For both case, daemon.readFileWithCID will return a "req timeout exceeded" error.
	//
	// Return value semantics: if returned error is not nil, it means this function is unsure that
	// whether there is some unexpected network error, or this is caused by an attacker.
	// Multiple retry on error is recommended to ensure that this is not a network error.
	file, err := daemon.readFileWithCID(cid)
	if err != nil {
		// log.Println("DEBUG: Unexpected err when read file,", err)
		switch {
		case errors.Is(err, errInvalidCID):
			// if invalid format cid is found, must be an attack
			return false, nil
		default:
			// for all other errors, could be an attack, or normal network error
			return false, err
		}
	}
	defer file.Close()

	r := bufio.NewReader(file)
	message := make([]byte, blueprint.MessageSize)
	n, err := r.Read(message)
	if err != nil {
		// memory error, suggest retry
		return false, err
	}
	// message must contains at least a key header
	if n < blueprint.KeySize {
		// log.Println("DEBUG: Unexpected err when msg < 48", err)
		return false, nil
	}

	return blueprint.ValidateKey(key, message[:n]), nil
}

func (daemon *ipfsClient) validateMappingsPage(fileName string, fileCID string, percentage int) (bool, error) {
	log.Printf("Validate %s...\n", fileName)
	file, err := daemon.readFileWithCID(fileCID)
	for retryCount, maxRetry := 0, 3; err != nil && retryCount < maxRetry; retryCount++ {
		if errors.Is(err, errInvalidCID) {
			// if invalid format cid is found, must be an attack
			return false, nil
		}
		file, err = daemon.readFileWithCID(fileCID)
	}
	// check final result
	if err != nil {
		if errors.Is(err, errInvalidCID) {
			return false, nil
		} else {
			return false, err
		}
	}
	mappings, err := parseMappings(file)
	if err != nil {
		// malformed mappings
		return false, nil
	}
	// random choose percentage mappings to validate
	populationSize := len(mappings)
	sampleSize := populationSize * percentage / 100
	samples := rand.Perm(populationSize)[:sampleSize]
	doneSuccess := make(chan bool)
	doneError := make(chan error)
	defer close(doneSuccess)
	defer close(doneError)
	aggOk := true

	debug := false
	if debug {
		log.Println("DEBUG: populationSize = ", populationSize)
		log.Println("DEBUG: sampleSize     = ", sampleSize)
		log.Println("DEBUG: samples        = ", samples)
	}

	// for each sample mapping, initiate a goroutine to validate it
	for _, sampleId := range samples {
		sample := mappings[sampleId]

		// log.Println("DEBUG: found a mapping from page:", sample.key, sample.cid)

		// Avoid using closure which consumes heap and also the running time of the goroutine is undefined
		go func(key blueprint.Key, cid string) {
			var ok bool
			var err error
			ok, err = daemon.validateMapping(key, cid)
			for retryCount, maxRetry := 0, 3; err != nil && retryCount < maxRetry; retryCount++ {
				// log.Println("DEBUG: call validateMapping", retryCount)
				ok, err = daemon.validateMapping(key, cid)
			}

			// log.Println("DEBUG: validation result,", ok, err)

			if err != nil {
				doneError <- err
			} else {
				doneSuccess <- ok
			}
		}(sample.key, sample.cid)
	}

	for completedJob := 0; completedJob < sampleSize; completedJob++ {
		select {
		case err := <-doneError:
			// return false, err if case of any error if encountered when checking, suggesting a retry on this function
			return false, err
		case ok := <-doneSuccess:
			// aggregate the individual validation results
			aggOk = aggOk && ok
		}
		// log.Println("DEBUG: Job", completedJob, "completed.", aggOk)
	}
	// return the aggregated validation results
	return aggOk, nil
}

func (daemon *ipfsClient) validateMappingsIPNS(mappingsIPNS string, percentage int) (bool, error) {
	// This function retrieves all mappings page under the /mappings directory pointed by mappingsIPNS.
	// For each mappings page, randomly select percentage% entries to validate correctness.
	// If any single entry is invalid, the entire mappingsIPNS is treated as invalid.
	//
	// Return value semantics, if returned error is not nil, it means this function is unsure that
	// whether there is some unexpected network error, or this is caused by an attacker.
	// Multiple retry on error is recommended to ensure this is not a network error.
	mappingsCID, err := daemon.resolveIPNSPointer(mappingsIPNS)
	if err != nil {
		// log.Println("DEBUG: error when resolve mappings IPNS", mappingsIPNS)
		switch {
		case errors.Is(err, errUnknownIPNS):
			// the IPNS pointer is not published or expired, no need to retry on this error
			return false, nil
		default:
			return false, err
		}
	}
	filesNameToCid, err := daemon.getDAGLinks(mappingsCID)
	if err != nil {
		// log.Println("DEBUG: error when getting filesnames and cid", err)
		switch {
		case errors.Is(err, errInvalidCID):
			// the IPNS pointer resolves to an invalid CID, no need to retry on this error
			return false, nil
		default:
			return false, err
		}
	}

	// For each mappings page, validate its correctness in parallel.
	sampleSize := len(filesNameToCid)
	doneSuccess := make(chan bool)
	doneError := make(chan error)
	defer close(doneSuccess)
	defer close(doneError)
	aggOk := true

	for fileName, fileCID := range filesNameToCid {
		go func(filename string, filecid string) {
			var ok bool
			var err error
			ok, err = daemon.validateMappingsPage(filename, filecid, percentage)
			for retryCount, maxRetry := 0, 3; err != nil && retryCount < maxRetry; retryCount++ {
				ok, err = daemon.validateMappingsPage(filename, filecid, percentage)
			}
			if err != nil {
				doneError <- err
			} else {
				doneSuccess <- ok
			}
		}(fileName, fileCID)
	}

	for completedJob := 0; completedJob < sampleSize; completedJob++ {
		select {
		case err := <-doneError:
			return false, err
		case ok := <-doneSuccess:
			aggOk = aggOk && ok
		}
	}
	return aggOk, nil
}
