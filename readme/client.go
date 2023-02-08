package readme

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

type Client struct {
	apiKey  string
	version string
}

func NewClient(ctx context.Context, apiKey string, version string) (*Client, error) {
	if apiKey == "" {
		return nil, xerrors.New("apiKey needed")
	}

	c := Client{
		apiKey:  apiKey,
		version: version,
	}

	return &c, nil
}

// caller must close response body
func (c *Client) do(method, url string, body any) (*http.Response, error) {
	var payload io.Reader
	if body != nil {
		buffer := new(bytes.Buffer)
		if err := json.NewEncoder(buffer).Encode(body); err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}
		payload = buffer
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	if payload != nil {
		req.Header.Add("content-type", "application/json")
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-readme-version", c.version)
	req.SetBasicAuth(c.apiKey, "")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}
	return res, nil
}
