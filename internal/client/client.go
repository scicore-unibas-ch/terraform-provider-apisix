// Package client is a thin, hardened HTTP client for the APISIX Admin API.
//
// Differences vs. the previous SDK v2 client:
//   - Update uses HTTP PUT (full replace), not PATCH (which silently merged).
//   - 404 returns a typed sentinel (ErrNotFound).
//   - Idempotent verbs (GET, PUT, DELETE) retry on transient network errors and 5xx.
//   - Error responses are parsed into typed APIError; secrets are not echoed verbatim.
//   - Optional TLS skip-verify and configurable transport.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ErrNotFound indicates the resource does not exist (HTTP 404).
var ErrNotFound = errors.New("apisix: not found")

// Config configures a Client.
type Config struct {
	BaseURL  string
	AdminKey string
	Timeout  time.Duration
	Insecure bool
}

// Client talks to the APISIX Admin API.
type Client struct {
	cfg  Config
	http *http.Client
}

// Response is the standard APISIX Admin API envelope. Only the value is
// decoded; the envelope's key field duplicates what callers already know.
type Response struct {
	Value json.RawMessage `json:"value"`
}

// APIError is a parsed APISIX error response.
type APIError struct {
	Status   int
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Raw      string `json:"-"`
}

func (e *APIError) Error() string {
	msg := e.ErrorMsg
	if msg == "" {
		msg = e.Message
	}
	if msg == "" {
		msg = e.Raw
	}
	return fmt.Sprintf("apisix: HTTP %d: %s", e.Status, msg)
}

// New returns a configured Client.
func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	transport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout, Transport: transport},
	}
}

// Put creates or fully replaces a resource.
func (c *Client) Put(ctx context.Context, kind, id string, body any) (*Response, error) {
	return c.do(ctx, http.MethodPut, c.url(kind, id), body)
}

// Get reads a resource. Returns ErrNotFound on 404.
func (c *Client) Get(ctx context.Context, kind, id string) (*Response, error) {
	return c.do(ctx, http.MethodGet, c.url(kind, id), nil)
}

// Delete removes a resource. force=true tells APISIX to delete despite references.
// A 404 is reported as ErrNotFound; callers may choose to ignore it.
func (c *Client) Delete(ctx context.Context, kind, id string, force bool) error {
	u := c.url(kind, id)
	if force {
		u += "?force=true"
	}
	_, err := c.do(ctx, http.MethodDelete, u, nil)
	return err
}

func (c *Client) url(kind, id string) string {
	if id == "" {
		return fmt.Sprintf("%s/%s", c.cfg.BaseURL, kind)
	}
	return fmt.Sprintf("%s/%s/%s", c.cfg.BaseURL, kind, id)
}

func (c *Client) do(ctx context.Context, method, url string, body any) (*Response, error) {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyBytes = b
	}

	const maxAttempts = 4
	backoff := 200 * time.Millisecond
	var lastErr error

	// Log fields shared across the request/response pair. Note: at TRACE
	// level the request and response bodies are logged in full; bodies may
	// contain sensitive material (TLS keys, plugin secrets). Operators who
	// enable TF_LOG=TRACE in shared environments should redirect logs to a
	// secure location.
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			tflog.Debug(ctx, "apisix: retrying request", map[string]any{
				"method": method, "url": url, "attempt": attempt,
			})
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		var reader io.Reader
		if bodyBytes != nil {
			reader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-KEY", c.cfg.AdminKey)

		tflog.Trace(ctx, "apisix: request", map[string]any{
			"method":    method,
			"url":       url,
			"body_size": len(bodyBytes),
			"body":      string(bodyBytes),
		})

		start := time.Now()
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			tflog.Debug(ctx, "apisix: transport error", map[string]any{
				"method":  method,
				"url":     url,
				"error":   err.Error(),
				"elapsed": time.Since(start).String(),
			})
			if isIdempotent(method) {
				continue
			}
			return nil, err
		}
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		tflog.Trace(ctx, "apisix: response", map[string]any{
			"method":    method,
			"url":       url,
			"status":    resp.StatusCode,
			"elapsed":   time.Since(start).String(),
			"body_size": len(data),
			"body":      string(data),
		})

		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		if resp.StatusCode >= 500 && isIdempotent(method) && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("apisix: HTTP %d (transient)", resp.StatusCode)
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, parseError(resp.StatusCode, data)
		}

		if len(data) == 0 {
			return &Response{}, nil
		}
		var r Response
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &r, nil
	}
	return nil, lastErr
}

func isIdempotent(method string) bool {
	return method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete
}

func parseError(status int, data []byte) error {
	e := &APIError{Status: status, Raw: string(data)}
	_ = json.Unmarshal(data, e)
	return e
}
