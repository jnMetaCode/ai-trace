package aitrace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the main AI-Trace client.
type Client struct {
	apiKey          string
	baseURL         string
	upstreamAPIKey  string
	upstreamBaseURL string
	httpClient      *http.Client
	timeout         time.Duration

	// Sub-clients
	Chat   *ChatService
	Events *EventsService
	Certs  *CertsService
}

// ClientOption is a function that configures the client.
type ClientOption func(*Client)

// WithBaseURL sets the base URL for the AI-Trace API.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithUpstreamAPIKey sets the upstream API key (e.g., OpenAI API key).
// This key is passed through to the upstream provider and never stored.
func WithUpstreamAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.upstreamAPIKey = apiKey
	}
}

// WithUpstreamBaseURL sets a custom upstream base URL (e.g., for proxy).
func WithUpstreamBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.upstreamBaseURL = baseURL
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new AI-Trace client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: "https://api.aitrace.cc",
		timeout: 120 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: c.timeout,
		}
	}

	// Initialize sub-clients
	c.Chat = &ChatService{client: c}
	c.Events = &EventsService{client: c}
	c.Certs = &CertsService{client: c}

	return c
}

// request performs an HTTP request.
func (c *Client) request(ctx context.Context, method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	// Build URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path

	// Encode body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	if c.upstreamAPIKey != "" {
		req.Header.Set("X-Upstream-API-Key", c.upstreamAPIKey)
	}
	if c.upstreamBaseURL != "" {
		req.Header.Set("X-Upstream-Base-URL", c.upstreamBaseURL)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return nil, &APIError{
				Code:       fmt.Sprintf("HTTP_%d", resp.StatusCode),
				Message:    string(respBody),
				StatusCode: resp.StatusCode,
			}
		}
		apiErr.StatusCode = resp.StatusCode
		return nil, &apiErr
	}

	return respBody, nil
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	// Build URL with query params
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path

	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-API-Key", c.apiKey)

	if c.upstreamAPIKey != "" {
		req.Header.Set("X-Upstream-API-Key", c.upstreamAPIKey)
	}
	if c.upstreamBaseURL != "" {
		req.Header.Set("X-Upstream-Base-URL", c.upstreamBaseURL)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return nil, &APIError{
				Code:       fmt.Sprintf("HTTP_%d", resp.StatusCode),
				Message:    string(respBody),
				StatusCode: resp.StatusCode,
			}
		}
		apiErr.StatusCode = resp.StatusCode
		return nil, &apiErr
	}

	return respBody, nil
}

// post performs a POST request.
func (c *Client) post(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	return c.request(ctx, http.MethodPost, path, body, headers)
}
