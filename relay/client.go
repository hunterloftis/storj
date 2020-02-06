package relay

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/http2"
)

const (
	proto = "https://"
)

type waitFn func() error

type Client struct {
	addr       string
	httpClient *http.Client
}

func NewClient(addr string, insecure bool) *Client {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cert)

	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            caCertPool,
				InsecureSkipVerify: insecure,
			},
		},
	}

	return &Client{
		addr:       addr,
		httpClient: client,
	}
}

func (c *Client) Send(filename string, file io.ReadCloser) (string, waitFn, error) {
	req, err := http.NewRequest(http.MethodPost, proto+c.addr+"/send", file)
	if err != nil {
		return "", nil, fmt.Errorf("building /send request: %w", err)
	}

	req.Header.Set(filenameHeader, filename)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("posting to /send: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", nil, fmt.Errorf("bad status code on /send: %v", resp.StatusCode)
	}

	sec, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil {
		return "", nil, fmt.Errorf("reading secret from /offer: %w", err)
	}

	wait := func() error {
		defer resp.Body.Close()
		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			return fmt.Errorf("waiting for body to close: %w", err)
		}
		return nil
	}

	return strings.TrimSpace(sec), wait, nil
}

func (c *Client) Receive(sec string) (filename string, body io.ReadCloser, err error) {
	req, err := http.NewRequest(http.MethodGet, proto+c.addr+"/receive", nil)
	if err != nil {
		return "", nil, fmt.Errorf("building /receive request: %w", err)
	}

	req.Header.Set(secretHeader, sec)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("posting to /receive: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", nil, fmt.Errorf("bad status code on /receive: %v", resp.StatusCode)
	}

	filename = resp.Header.Get(filenameHeader)
	return filename, resp.Body, nil
}
