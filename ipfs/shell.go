package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
)

type IPFSRequest struct {
	sh      *shell.Shell
	cli     *http.Client
	timeout time.Duration
}

func NewIPFSClient(timeout time.Duration) *IPFSRequest {
	return &IPFSRequest{shell.NewShell("localhost:5001"), &http.Client{}, timeout}
}

func (req *IPFSRequest) CreateDirectory(path string) error {
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

func (req *IPFSRequest) GetDirectoryCID(path string) (string, error) {
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

func (req *IPFSRequest) GetDirectorySize(path string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()

	res, err := req.sh.FilesStat(ctx, path, func(rb *shell.RequestBuilder) error { return nil })
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return res.Size, nil
}

func (req *IPFSRequest) AppendStringToFile(
	path string,
	content string,
	fileSizeLimit uint64,
) error {
	// obtain offset by getting the file size in byte
	offset, _ := req.GetDirectorySize(path)
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

func (req *IPFSRequest) ReadFileWithPath(path string, offset uint64) (io.ReadCloser, error) {
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

func (req *IPFSRequest) ReadFileWithCID(cid string) (io.ReadCloser, error) {
	//TODO
	content, err := req.sh.Cat(cid)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return content, err
}

func (req *IPFSRequest) GetDAGLinks(cid string) (map[string]string, error) {
	//TODO : This does not work, Response only return 1 byte in body, need to investigate why
	// Since go-ipfs-api still have not completed this function, we use low level HTTP request
	log.Println(
		fmt.Sprintf("http://127.0.0.1:5001/api/v0/dag/get?arg=%s&output-codec=dag-json", cid),
	)
	r, err := http.NewRequest(
		"POST",
		fmt.Sprintf("http://127.0.0.1:5001/api/v0/dag/get?arg=%s&output-codec=dag-json", cid),
		nil,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(r.UserAgent())
	r.Header.Add("Accept", "*/*")
	r.Header.Add("User-Agent", "curl/8.1.1")

	res, err := req.cli.Do(r)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	log.Println(res.Status, res.ContentLength)

	var body [512]byte
	n, err := res.Body.Read(body[:])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(n)
	log.Println(body[:n])
	return nil, nil
}
