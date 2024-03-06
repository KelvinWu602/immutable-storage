package ipfs

import (
	"context"
	"testing"

	"github.com/KelvinWu602/immutable-storage/blueprint"
)

var validKey blueprint.Key = blueprint.Key([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 58, 133, 171, 96, 145, 21, 81, 70, 176, 173, 9, 242, 195, 206, 27, 15, 32, 31, 201, 126, 49, 46, 203, 226, 113, 180, 10, 180, 222, 232, 106, 190})
var validMsg []byte = []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 58, 133, 171, 96, 145, 21, 81, 70, 176, 173, 9, 242, 195, 206, 27, 15, 32, 31, 201, 126, 49, 46, 203, 226, 113, 180, 10, 180, 222, 232, 106, 190, 104, 101, 108, 108, 111, 119, 111, 114, 108, 100}

// Run these scripts to run the test
// ./localtesting-setup/reset-ipfs.sh
// ./localtesting-setup/init-ipfs.sh
// docker run -d --network host --name node-discovery 3f4440ff7ef6
// go test -run TestValidateMapping -v
func TestValidateMapping(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipfsImp, err := New(ctx, "")
	if err != nil {
		t.Fatalf("Unexpected error when creating new IPFS node: %s", err)
	}
	err = ipfsImp.Store(validKey, validMsg)
	if err != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", err)
	}
	msgCid := ipfsImp.keyToCid[validKey].cid
	// test the validate mapping
	valid, errValidate := ipfsImp.ipfsClient.validateMapping(validKey, msgCid)
	if errValidate != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", errValidate)
	}
	if !valid {
		t.Fatalf("Expected valid = true, Actual = false")
	}
}

func TestValidateMappingsPage100Percent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipfsImp, err := New(ctx, "")
	if err != nil {
		t.Fatalf("Unexpected error when creating new IPFS node: %s", err)
	}
	err = ipfsImp.Store(validKey, validMsg)
	if err != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", err)
	}
	// test the validate mappings page

	mappingsCID, err := ipfsImp.ipfsClient.resolveIPNSPointer(ipfsImp.mappingsIPNS)
	if err != nil {
		t.Fatalf("Unexpected error when resolving mappingsIPNS: %s", err)
	}
	filesNameToCid, err := ipfsImp.ipfsClient.getDAGLinks(mappingsCID)
	if err != nil {
		t.Fatalf("Unexpected error when getting DAG links: %s", err)
	}
	valid, errValidate := ipfsImp.ipfsClient.validateMappingsPage("000000.txt", filesNameToCid["000000.txt"], 100)
	if errValidate != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", errValidate)
	}
	if !valid {
		t.Fatalf("Expected valid = true, Actual = false")
	}
}

func TestValidateMappingsPage10Percent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipfsImp, err := New(ctx, "")
	if err != nil {
		t.Fatalf("Unexpected error when creating new IPFS node: %s", err)
	}
	err = ipfsImp.Store(validKey, validMsg)
	if err != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", err)
	}
	// test the validate mappings page

	mappingsCID, err := ipfsImp.ipfsClient.resolveIPNSPointer(ipfsImp.mappingsIPNS)
	if err != nil {
		t.Fatalf("Unexpected error when resolving mappingsIPNS: %s", err)
	}
	filesNameToCid, err := ipfsImp.ipfsClient.getDAGLinks(mappingsCID)
	if err != nil {
		t.Fatalf("Unexpected error when getting DAG links: %s", err)
	}
	valid, errValidate := ipfsImp.ipfsClient.validateMappingsPage("000000.txt", filesNameToCid["000000.txt"], 10)
	if errValidate != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", errValidate)
	}
	if !valid {
		t.Fatalf("Expected valid = true, Actual = false")
	}
}

func TestValidateMappingsIPNS100Percent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipfsImp, err := New(ctx, "")
	if err != nil {
		t.Fatalf("Unexpected error when creating new IPFS node: %s", err)
	}
	err = ipfsImp.Store(validKey, validMsg)
	if err != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", err)
	}
	// test validate mappingsIPNS
	valid, errValidate := ipfsImp.ipfsClient.validateMappingsIPNS(ipfsImp.mappingsIPNS, 100)
	if errValidate != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", errValidate)
	}
	if !valid {
		t.Fatalf("Expected valid = true, Actual = false")
	}
}

func TestValidateMappingsIPNS10Percent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipfsImp, err := New(ctx, "")
	if err != nil {
		t.Fatalf("Unexpected error when creating new IPFS node: %s", err)
	}
	err = ipfsImp.Store(validKey, validMsg)
	if err != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", err)
	}
	// test validate mappingsIPNS
	valid, errValidate := ipfsImp.ipfsClient.validateMappingsIPNS(ipfsImp.mappingsIPNS, 10)
	if errValidate != nil {
		t.Fatalf("Unexpected error when Storing valid key msg pair: %s", errValidate)
	}
	if !valid {
		t.Fatalf("Expected valid = true, Actual = false")
	}
}
