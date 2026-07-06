package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(baseURL string) *Client {
	return New(Config{BaseURL: baseURL, AdminKey: "test-key", Timeout: 5 * time.Second})
}

func TestGet_DecodesEnvelopeAndSendsHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-KEY"); got != "test-key" {
			t.Errorf("X-API-KEY = %q, want test-key", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if r.URL.Path != "/consumer_groups/g1" {
			t.Errorf("path = %q, want /consumer_groups/g1", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"key":"/apisix/consumer_groups/g1","value":{"id":"g1"}}`))
	}))
	defer srv.Close()

	resp, err := newTestClient(srv.URL).Get(context.Background(), "consumer_groups", "g1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(resp.Value) != `{"id":"g1"}` {
		t.Errorf("Value = %s", resp.Value)
	}
}

func TestGet_NotFoundSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).Get(context.Background(), "routes", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPut_SendsBody(t *testing.T) {
	var gotBody string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).Put(context.Background(), "routes", "r1", map[string]string{"id": "r1"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotBody != `{"id":"r1"}` {
		t.Errorf("body = %q", gotBody)
	}
}

func TestDelete_ForceQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if err := c.Delete(context.Background(), "routes", "r1", false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query without force = %q, want empty", gotQuery)
	}
	if err := c.Delete(context.Background(), "routes", "r1", true); err != nil {
		t.Fatalf("Delete force: %v", err)
	}
	if gotQuery != "force=true" {
		t.Errorf("query with force = %q, want force=true", gotQuery)
	}
}

func TestRetry_5xxThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"value":{"ok":true}}`))
	}))
	defer srv.Close()

	resp, err := newTestClient(srv.URL).Get(context.Background(), "routes", "r1")
	if err != nil {
		t.Fatalf("Get after retries: %v", err)
	}
	if resp == nil || string(resp.Value) != `{"ok":true}` {
		t.Errorf("unexpected response: %+v", resp)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("server saw %d calls, want 3", got)
	}
}

func TestRetry_ExhaustedReturnsAPIError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_msg":"etcd down"}`))
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).Get(context.Background(), "routes", "r1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("Status = %d, want 500", apiErr.Status)
	}
	// 4 attempts total: initial + 3 retries; the final 5xx is surfaced.
	if got := calls.Load(); got != 4 {
		t.Errorf("server saw %d calls, want 4", got)
	}
}

func TestClientError_NoRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_msg":"invalid configuration"}`))
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).Put(context.Background(), "routes", "r1", map[string]string{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server saw %d calls, want 1 (4xx must not be retried)", got)
	}
	if !strings.Contains(apiErr.Error(), "invalid configuration") {
		t.Errorf("error message %q should contain the server error_msg", apiErr.Error())
	}
}

func TestTransportError_RetriedThenFails(t *testing.T) {
	// A closed listener: every attempt gets a connection error.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := newTestClient(url).Get(context.Background(), "routes", "r1")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("transport failure must not be reported as ErrNotFound: %v", err)
	}
}

func TestContextCancellation_StopsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestClient(srv.URL).Get(ctx, "routes", "r1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestParseError_MessageFallbacks(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"error_msg preferred", `{"error_msg":"a","message":"b"}`, "a"},
		{"message fallback", `{"message":"b"}`, "b"},
		{"raw fallback", `not json at all`, "not json at all"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := parseError(400, []byte(tc.body))
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Error() = %q, want it to contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestURL_WithAndWithoutID(t *testing.T) {
	c := newTestClient("http://example/apisix/admin")
	if got := c.url("routes", "r1"); got != "http://example/apisix/admin/routes/r1" {
		t.Errorf("url with id = %q", got)
	}
	if got := c.url("routes", ""); got != "http://example/apisix/admin/routes" {
		t.Errorf("url without id = %q", got)
	}
}
