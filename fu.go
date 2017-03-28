package fu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pnelson/fu/api"
)

// Client represents a fu HTTP client.
type Client struct {
	addr   string
	token  string
	client *http.Client
}

// NewClient returns a new fu HTTP client.
func NewClient(addr, token string) (*Client, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, errors.New("fu: addr must be a url")
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
		addr = u.String()
	}
	c := &Client{
		addr:   addr,
		token:  token,
		client: &http.Client{Timeout: 90 * time.Second},
	}
	return c, nil
}

// URL returns the url to view the file.
func (c *Client) URL(file *api.File) string {
	return fmt.Sprintf("%s%s", c.addr, file.Name)
}

// Upload returns the uploaded persisted file.
func (c *Client) Upload(f *os.File, name string, d time.Duration) (*api.File, error) {
	var file *api.File
	body, header, err := c.makeBody(f, name, d)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.addr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fu")
	req.Header.Set("Content-Type", header)
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(b))
	}
	err = json.Unmarshal(b, &file)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *Client) makeBody(f *os.File, name string, d time.Duration) (*bytes.Buffer, string, error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	file, err := w.CreateFormFile("file", name)
	if err != nil {
		return nil, "", err
	}
	_, err = io.Copy(file, f)
	if err != nil {
		return nil, "", err
	}
	err = w.WriteField("token", c.token)
	if err != nil {
		return nil, "", err
	}
	err = w.WriteField("duration", d.String())
	if err != nil {
		return nil, "", err
	}
	err = w.Close()
	if err != nil {
		return nil, "", err
	}
	header := w.FormDataContentType()
	return buf, header, nil
}
