package integrate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient wraps http.Client with retry logic
type HTTPClient struct {
	client        *http.Client
	retryAttempts int
	retryDelay    time.Duration
}

// NewHTTPClient creates a new HTTP client with retry support
func NewHTTPClient(timeout time.Duration, retryAttempts int, retryDelay time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		retryAttempts: retryAttempts,
		retryDelay:    retryDelay,
	}
}

// Get performs HTTP GET with retry logic
func (c *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.doWithRetry(req)
}

// Post performs HTTP POST with retry logic
func (c *HTTPClient) Post(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.doWithRetry(req)
}

// Put performs HTTP PUT with retry logic
func (c *HTTPClient) Put(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.doWithRetry(req)
}

// doWithRetry executes HTTP request with retry logic
func (c *HTTPClient) doWithRetry(req *http.Request) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		// Client error - don't retry
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, fmt.Errorf("client error %d: %s", resp.StatusCode, string(body))
		}

		// Server error - retry
		lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.retryAttempts+1, lastErr)
}
