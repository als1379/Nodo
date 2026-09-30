package wiktapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"nodo/internal/dictionary"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}
}

func (c *Client) Lookup(ctx context.Context, word string) (dictionary.WordDetails, error) {
	endpoint := c.baseURL + "/v1/en/word/" + url.PathEscape(word) + "?lang=it"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return dictionary.WordDetails{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Nodo/0.1 (Italian learning app)")
	resp, err := c.http.Do(req)
	if err != nil {
		return dictionary.WordDetails{}, fmt.Errorf("call WiktAPI: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return dictionary.WordDetails{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return dictionary.WordDetails{}, dictionary.ErrWordNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return dictionary.WordDetails{}, fmt.Errorf("WiktAPI returned status %d", resp.StatusCode)
	}
	var response response
	if err := json.Unmarshal(body, &response); err != nil {
		return dictionary.WordDetails{}, fmt.Errorf("decode WiktAPI response: %w", err)
	}
	return mapResponse(word, response)
}
