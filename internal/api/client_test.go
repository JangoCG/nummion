package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestJSONSetsAuthenticationAndUserAgent(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "lexware-cli/test" {
			t.Fatalf("User-Agent = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"companyName":"Example"}`)
	}))
	defer server.Close()

	client, err := New(server.URL, "secret", "lexware-cli/test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.SetMinInterval(0)
	raw, err := client.JSON(context.Background(), http.MethodGet, "/v1/profile", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"companyName":"Example"}` {
		t.Fatalf("response = %s", raw)
	}
}

func TestJSONParsesRegularValidationError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotAcceptable)
		io.WriteString(w, `{"message":"Validation failed","traceId":"trace-1","details":[{"field":"lineItems[0].name","message":"darf nicht leer sein"}]}`)
	}))
	defer server.Close()

	client, err := New(server.URL, "secret", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.SetMinInterval(0)
	_, err = client.JSON(context.Background(), http.MethodPost, "/v1/invoices", nil, map[string]any{})
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	if apiErr.Message != "lineItems[0].name: darf nicht leer sein" {
		t.Fatalf("message = %q", apiErr.Message)
	}
	if apiErr.TraceID != "trace-1" {
		t.Fatalf("trace = %q", apiErr.TraceID)
	}
}

func TestJSONRetries429AndPreservesPOSTBody(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil || decoded["name"] != "Example" {
			t.Fatalf("body = %s, err = %v", body, err)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"message":"slow down"}`)
			return
		}
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"created"}`)
	}))
	defer server.Close()

	client, err := New(server.URL, "secret", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.SetMinInterval(0)
	raw, err := client.JSON(context.Background(), http.MethodPost, "/v1/contacts", nil, map[string]any{"name": "Example"})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || string(raw) != `{"id":"created"}` {
		t.Fatalf("calls = %d, response = %s", calls.Load(), raw)
	}
}

func TestPostDoesNotRetryGatewayTimeout(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusGatewayTimeout)
		io.WriteString(w, `{"message":"timeout"}`)
	}))
	defer server.Close()

	client, err := New(server.URL, "secret", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.SetMinInterval(0)
	_, err = client.JSON(context.Background(), http.MethodPost, "/v1/invoices", nil, map[string]any{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestNewRejectsUnencryptedRemoteBaseURL(t *testing.T) {
	t.Parallel()
	if _, err := New("http://example.com", "secret", "test", time.Second); err == nil {
		t.Fatal("expected insecure remote URL to be rejected")
	}
}
