package ipfs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
)

var errRequestTimeout error = errors.New("req timeout exceeded")
var errConnectionRefused error = errors.New("connection refused")
var errInvalidCID error = errors.New("invalid cid")
var errUnknownIPNS error = errors.New("unknown ipns")
var errExceedFileSizeLimit error = errors.New("exceed file size limit")

type ipfsClient struct {
	sh      *shell.Shell
	cli     *http.Client
	timeout time.Duration
}

// returns a pointer to the ipfsClient object, representing a connection with the IPFS daemon.
// This is a blocking function and it returns only if the daemon is alive.
func newIPFSClient(host string, timeout time.Duration) *ipfsClient {
	// Ensure the daemon is alive
	for !isIPFSDaemonAlive(host) {
		log.Println("Failed to connect to IPFS Daemon, retry after 1s...")
		time.Sleep(time.Second)
	}
	return &ipfsClient{shell.NewShell(host), &http.Client{}, timeout}
}

func isIPFSDaemonAlive(host string) bool {
	cli := &http.Client{}
	res, err := cli.Head("http://" + host + "/api/v0/id")
	if err != nil {
		log.Println("Exception when check daemon alive", err)
		return false
	}
	switch res.StatusCode {
	case 405:
		return true
	default:
		return false
	}
}

func (req *ipfsClient) createDirectory(path string) error {
	// create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	// make HTTP request to ipfs daemon
	err := req.sh.FilesMkdir(ctx, path, func(rb *shell.RequestBuilder) error { return nil })
	// check error
	if err != nil {
		log.Println(err)
	}
	return err
}

func (req *ipfsClient) getDirectoryCID(path string) (string, error) {
	// create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	// make HTTP request to ipfs daemon
	res, err := req.sh.FilesStat(ctx, path, func(rb *shell.RequestBuilder) error { return nil })
	// check error
	if err != nil {
		log.Println(err)
		return "", err
	}
	return res.Hash, nil
}

func (req *ipfsClient) getDirectorySize(path string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()

	res, err := req.sh.FilesStat(ctx, path, func(rb *shell.RequestBuilder) error { return nil })
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return res.Size, nil
}

func (req *ipfsClient) appendStringToFile(
	path string,
	content string,
	fileSizeLimit uint64,
) error {
	// obtain offset by getting the file size in byte
	offset, _ := req.getDirectorySize(path)
	if offset+uint64(len(content)) > fileSizeLimit {
		return errExceedFileSizeLimit
	}
	options := func(rb *shell.RequestBuilder) error {
		rb.Option("create", true)   // create file if not exist
		rb.Option("offset", offset) // write from that byte
		return nil
	}
	// create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	// make HTTP request to ipfs daemon
	data := strings.NewReader(content)
	err := req.sh.FilesWrite(ctx, path, data, options)

	if err != nil {
		log.Println(err)
	}
	return err
}

func (req *ipfsClient) addFile(content []byte) (string, error) {
	cid, err := req.sh.Add(bytes.NewReader(content))
	if err != nil {
		log.Println(err)
		return "", err
	}
	return cid, nil
}

func (req *ipfsClient) removeFile(path string, recursive bool) error {
	// create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()

	err := req.sh.FilesRm(ctx, path, recursive)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (req *ipfsClient) readFileWithPath(path string, offset uint64) (io.ReadCloser, error) {
	options := func(rb *shell.RequestBuilder) error {
		rb.Option("offset", offset)
		return nil
	}
	// create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()

	content, err := req.sh.FilesRead(ctx, path, options)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return content, err
}

func (req *ipfsClient) readFileWithCID(cid string) (io.ReadCloser, error) {
	// req.sh.Cat will return error immediately for malformed CID,
	// however it does not have built-in timeout mechanism for a valid but non-exist CID.
	// it will wait for the daemon to query all nodes in the world.

	// implemented timeout function for readFileWithCID operation
	doneSuccess := make(chan io.ReadCloser, 1)
	doneError := make(chan error, 1)
	go func() {
		defer close(doneError)
		defer close(doneSuccess)
		content, err := req.sh.Cat(cid)
		if err != nil {
			doneError <- err
		} else {
			doneSuccess <- content
		}
	}()

	select {
	case content := <-doneSuccess:
		return content, nil
	case err := <-doneError:
		return nil, ipfsErrorClassifier(err)
	case <-time.After(req.timeout):
		return nil, errRequestTimeout
	}
}

type SlashData struct {
	Slash Data `json:"/"`
}

type SlashHash struct {
	Slash string `json:"/"`
}

type Data struct {
	Bytes string `json:"bytes"`
}

type Hash struct {
	Hash  SlashHash `json:"Hash"`
	Name  string    `json:"Name"`
	Tsize int       `json:"Tsize"`
}

type DagGetJson struct {
	Data  SlashData `json:"Data"`
	Links []Hash    `json:"Links"`
}

// getDAGLinks returns a map with file name or directory name as key, cid as values.
func (req *ipfsClient) getDAGLinks(cid string) (map[string]string, error) {
	// req.sh.DagGet will return error immediately for malformed CID,
	// however it does not have built-in timeout mechanism for a valid but non-exist CID.
	// it will wait for the daemon to query all nodes in the world.

	// implemented timeout function for getDAGLinks operation
	// use buffered channel to avoid blocking if the outer function returned first
	doneSuccess := make(chan map[string]string, 1)
	doneError := make(chan error, 1)
	go func() {
		defer close(doneError)
		defer close(doneSuccess)
		var resJson DagGetJson
		err := req.sh.DagGet(cid, &resJson)
		if err != nil {
			doneError <- err
		}
		links := make(map[string]string)
		for _, v := range resJson.Links {
			if len(v.Name) > 0 {
				// files also contains links, but those links are nameless
				links[v.Name] = v.Hash.Slash
			}
		}
		doneSuccess <- links
	}()

	select {
	case links := <-doneSuccess:
		return links, nil
	case err := <-doneError:
		return nil, ipfsErrorClassifier(err)
	case <-time.After(req.timeout):
		return nil, errRequestTimeout
	}
}

func (req *ipfsClient) publishIPNSPointer(cid string, key string) (string, error) {
	//TODO research on the effect of lifetime, see if any further actions are required to keep IPNS record alive
	res, err := req.sh.PublishWithDetails(cid, key, 100*365*24*time.Hour, 5*time.Minute, true)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return res.Name, nil
}

func (req *ipfsClient) resolveIPNSPointer(ipns string) (string, error) {
	res, err := req.sh.Resolve(ipns)
	if err != nil {
		log.Println(err)
		return "", ipfsErrorClassifier(err)
	}
	return res, nil
}

func (req *ipfsClient) generateKey(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	_, err := req.sh.KeyGen(ctx, name, func(rb *shell.RequestBuilder) error { return nil })
	return err
}

func (req *ipfsClient) listKeys() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	res, err := req.sh.KeyList(ctx)
	if err != nil {
		return nil, err
	}
	keynames := make([]string, len(res))
	for i, key := range res {
		keynames[i] = key.Name
	}
	return keynames, nil
}

func ipfsErrorClassifier(err error) error {
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "connection refused"):
		return errConnectionRefused
	case strings.Contains(errStr, "could not resolve name"):
		return errUnknownIPNS
	case strings.Contains(errStr, "invalid cid"):
		return errInvalidCID
	default:
		return err
	}
}
