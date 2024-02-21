package ipfs

import (
	"bufio"
	"testing"
	"time"
)

func TestReadFileWithCID(t *testing.T) {
	ipfs := newIPFSClient("localhost:5001", 2*time.Second)
	// file, err := ipfs.readFileWithCID("QmSzF7FtXFJ5rqCTK2u5oGWd2jQsojtykF6PAK1HFqnhsf")
	file, err := ipfs.readFileWithCID("fakecid")
	if err != nil {
		t.Fatal("error first read", err)
	}
	r := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	n, err := r.Read(buffer)
	if err != nil {
		t.Fatal("error second read", err)
	}
	t.Fatalf(string(buffer[:n]))

}

func TestResolveIPNSPointer(t *testing.T) {
	ipfs := newIPFSClient("localhost:5001", 2*time.Second)
	// file, err := ipfs.readFileWithCID("QmSzF7FtXFJ5rqCTK2u5oGWd2jQsojtykF6PAK1HFqnhsf")
	cid, err := ipfs.resolveIPNSPointer("fakeipns")
	if err != nil {
		t.Fatal("error first read", err)
	}
	t.Fatalf(cid)
}

func TestGetDAGLinks(t *testing.T) {
	ipfs := newIPFSClient("localhost:5001", 2*time.Second)
	// file, err := ipfs.readFileWithCID("QmSzF7FtXFJ5rqCTK2u5oGWd2jQsojtykF6PAK1HFqnhsf")
	// no error when pass in a existing valid cid
	// timeout when pass in a non-exist valid cid
	// 'invalid cid' when pass in a invalid cid
	links, err := ipfs.getDAGLinks("awfw")
	if err != nil {
		t.Fatal("error first read", err)
	}
	t.Log(links)
}
