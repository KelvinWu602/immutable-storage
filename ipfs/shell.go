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

type ipfsRequest struct {
	sh      *shell.Shell
	cli     *http.Client
	timeout time.Duration
}

func newIPFSClient(host string, timeout time.Duration) *ipfsRequest {
	return &ipfsRequest{shell.NewShell(host), &http.Client{}, timeout}
}

func (req *ipfsRequest) createDirectory(path string) error {
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

func (req *ipfsRequest) getDirectoryCID(path string) (string, error) {
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

func (req *ipfsRequest) getDirectorySize(path string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()

	res, err := req.sh.FilesStat(ctx, path, func(rb *shell.RequestBuilder) error { return nil })
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return res.Size, nil
}

func (req *ipfsRequest) appendStringToFile(
	path string,
	content string,
	fileSizeLimit uint64,
) error {
	// obtain offset by getting the file size in byte
	offset, _ := req.getDirectorySize(path)
	if offset+uint64(len(content)) > fileSizeLimit {
		return errors.New("exceed file size limit")
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

func (req *ipfsRequest) addFile(content []byte) (string, error) {
	cid, err := req.sh.Add(bytes.NewReader(content))
	if err != nil {
		log.Println(err)
		return "", err
	}
	return cid, nil
}

func (req *ipfsRequest) removeFile(path string, recursive bool) error {
	// create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()

	err := req.sh.FilesRm(ctx, path, recursive)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (req *ipfsRequest) readFileWithPath(path string, offset uint64) (io.ReadCloser, error) {
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

func (req *ipfsRequest) readFileWithCID(cid string) (io.ReadCloser, error) {
	content, err := req.sh.Cat(cid)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return content, err
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
func (req *ipfsRequest) getDAGLinks(cid string) (map[string]string, error) {
	var resJson DagGetJson
	err := req.sh.DagGet(cid, &resJson)
	if err != nil {
		return nil, err
	}
	links := make(map[string]string)
	for _, v := range resJson.Links {
		if len(v.Name) > 0 {
			// files also contains links, but those links are nameless
			links[v.Name] = v.Hash.Slash
		}
	}
	return links, nil
}

func (req *ipfsRequest) publishIPNSPointer(cid string, key string) (string, error) {
	//TODO research on the effect of lifetime, see if any further actions are required to keep IPNS record alive
	res, err := req.sh.PublishWithDetails(cid, key, 100*365*24*time.Hour, 5*time.Minute, true)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return res.Name, nil
}

func (req *ipfsRequest) resolveIPNSPointer(ipns string) (string, error) {
	res, err := req.sh.Resolve(ipns)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return res, nil
}

func (req *ipfsRequest) generateKey(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	_, err := req.sh.KeyGen(ctx, name, func(rb *shell.RequestBuilder) error { return nil })
	return err
}

func (req *ipfsRequest) listKeys() ([]string, error) {
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
