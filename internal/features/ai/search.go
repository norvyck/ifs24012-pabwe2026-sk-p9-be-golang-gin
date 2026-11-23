package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchClient interface {
	Search(query string) ([]SearchSource, error)
}

type HTTPSearchClient struct {
	Endpoint   string
	APIKey     string
	HTTPClient *http.Client
}

func NewHTTPSearchClient(endpoint, apiKey string) *HTTPSearchClient {
	return &HTTPSearchClient{
		Endpoint:   strings.TrimRight(endpoint, "/"),
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type searchAPIResponse struct {
	Results []SearchSource `json:"results"`
	Data    []SearchSource `json:"data"`
}

func (c *HTTPSearchClient) Search(query string) ([]SearchSource, error) {
	if c.Endpoint == "" {
		return []SearchSource{}, nil
	}

	reqURL := fmt.Sprintf("%s?q=%s", c.Endpoint, url.QueryEscape(query))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search engine returned status %d", resp.StatusCode)
	}

	var parsed searchAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if len(parsed.Results) > 0 {
		return parsed.Results, nil
	}
	if len(parsed.Data) > 0 {
		return parsed.Data, nil
	}
	return []SearchSource{}, nil
}
