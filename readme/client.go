package readme

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/xerrors"
)

type Client struct {
	apiKey  string
	version string
	url     string
}

func NewClient(ctx context.Context, apiKey string, version string) (*Client, error) {
	if apiKey == "" {
		return nil, xerrors.New("apiKey needed")
	}

	c := Client{
		apiKey:  apiKey,
		version: version,
		url:     "https://dash.readme.com",
	}

	return &c, nil
}

type doOpts struct {
	method         string
	path           string
	expectedStatus int
	body           any
}

func do[T any](c *Client, opts doOpts) (out T, err error) {
	var payload io.Reader
	if opts.body != nil {
		buffer := new(bytes.Buffer)
		if err := json.NewEncoder(buffer).Encode(opts.body); err != nil {
			return out, xerrors.Errorf(": %w", err)
		}
		payload = buffer
	}

	url := c.url + opts.path // unified host url
	req, err := http.NewRequest(opts.method, url, payload)
	if err != nil {
		return out, xerrors.Errorf(": %w", err)
	}

	if payload != nil {
		req.Header.Add("content-type", "application/json")
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-readme-version", c.version)
	req.SetBasicAuth(c.apiKey, "")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, xerrors.Errorf(": %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusNoContent {
		return out, nil
	}

	if res.StatusCode != opts.expectedStatus {
		return out, xerrors.Errorf(": %w", handleErrorResponse(res.Body))
	}

	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return out, xerrors.Errorf(": %w", err)
	}

	return out, nil
}

func doAllPages[T any](c *Client, method, path string) ([]T, error) {
	const perPage = 20
	var page = 1

	url, err := url.Parse(path)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	query := url.Query()
	query.Set("perPage", strconv.Itoa(perPage))

	var results []T
	for done := false; !done; {
		query.Set("page", strconv.Itoa(page))
		url.RawQuery = query.Encode()

		res, err := do[[]T](c, doOpts{
			method:         method,
			path:           url.String(),
			expectedStatus: http.StatusOK,
		})
		if err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}
		results = append(results, res...)

		// TODO: wastes 1 api call
		// https://docs.readme.com/main/reference/pagination
		if len(res) == 0 {
			done = true
		}
		page++
	}
	return results, nil
}
