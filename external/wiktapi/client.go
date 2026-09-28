package wiktapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}
}

func (c *Client) LookupItalian(ctx context.Context, word string) ([]byte, error) {
	endpoint := c.baseURL + "/v1/en/word/" + url.PathEscape(word) + "?lang=it"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Nodo/0.1 (Italian learning app)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call WiktAPI: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WiktAPI returned status %d", resp.StatusCode)
	}
	return body, nil
}
