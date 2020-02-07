package relay

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	proto = "http://"
)

// SendFn is a function that blocks until a file being sent has been completely downloaded.
type SendFn func() error

// Client can send to or receive from a relay server.
type Client struct {
	addr string
}

// NewClient creates a new Client that will communicate with the server at the specified address.
func NewClient(addr string) *Client {
	return &Client{
		addr: addr,
	}
}

// Offer offers a file, with a proposed filename, to a recipient via the relay server.
//
// It does not block on sending the file, but instead returns the file's secret immediately
// along with a blocking function to send the file's contents.
//
//	secret, send, _ := client.Offer(filename, file)
// 	fmt.Println(secret)	// immediately show the secret
//	_ = send()					// wait for the file to be sent
func (c *Client) Offer(filename string, file io.ReadCloser) (secret string, send SendFn, err error) {
	req, _ := http.NewRequest(http.MethodPost, proto+c.addr+"/file", nil)
	req.Header.Set(filenameHeader, filename)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("posting to offer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", nil, fmt.Errorf("bad status code on offer: %v", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, 100)
	secret, err = bufio.NewReader(limited).ReadString('\n')
	if err != nil {
		return "", nil, fmt.Errorf("reading secret from offer: %w", err)
	}
	secret = strings.TrimSpace(secret)

	send = func() error {
		// TODO: make this a request WithContext (req = req.WithContext(ctx))
		// Then cancel the context whenever the receiver disconnects.
		// Ditto in reverse, if that doesn't already happen from the ending of the stream...
		req, _ := http.NewRequest(http.MethodPut, proto+c.addr+"/file/"+secret, file)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	return secret, send, nil
}

// Receive receives a file stored with the given secret.
//
// It returns immediately with a proposed filename and a stream from which to read the file contents.
// The filename has been suggested by the sender and should not be trusted without validation.
func (c *Client) Receive(secret string) (filename string, stream io.ReadCloser, err error) {
	endpoint := proto + c.addr + "/file/" + secret
	resp, err := http.DefaultClient.Get(endpoint)
	if err != nil {
		return "", nil, fmt.Errorf("receiving: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", nil, fmt.Errorf("bad status code receiving: %v", resp.StatusCode)
	}

	return resp.Header.Get(filenameHeader), resp.Body, nil
}
