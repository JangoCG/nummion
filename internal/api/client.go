package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://api.lexware.io"

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	userAgent  string

	limitMu     sync.Mutex
	lastRequest time.Time
	minInterval time.Duration
}

func New(baseURL, token, userAgent string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("ungültige API-Basis-URL %q", baseURL)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("API-Basis-URL darf keine Zugangsdaten enthalten")
	}
	if parsed.Scheme != "https" && !isLoopbackHost(parsed.Hostname()) {
		return nil, fmt.Errorf("API-Basis-URL muss HTTPS verwenden; unverschlüsseltes HTTP ist nur für lokale Tests erlaubt")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL:     parsed,
		token:       strings.TrimSpace(token),
		httpClient:  &http.Client{Timeout: timeout},
		userAgent:   userAgent,
		minInterval: 550 * time.Millisecond,
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) SetHTTPClient(client *http.Client) {
	if client != nil {
		c.httpClient = client
	}
}

func (c *Client) SetMinInterval(interval time.Duration) {
	c.minInterval = interval
}

func (c *Client) NewRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	relative, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("ungültiger API-Pfad: %w", err)
	}
	requestURL := c.baseURL.ResolveReference(relative)
	if query != nil {
		requestURL.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return req, nil
}

func (c *Client) JSON(ctx context.Context, method, path string, query url.Values, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("JSON konnte nicht kodiert werden: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	return c.JSONReader(ctx, method, path, query, reader)
}

func (c *Client) JSONBytes(ctx context.Context, method, path string, query url.Values, body []byte) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	return c.JSONReader(ctx, method, path, query, reader)
}

func (c *Client) JSONReader(ctx context.Context, method, path string, query url.Values, body io.Reader) (json.RawMessage, error) {
	req, err := c.NewRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("API-Antwort konnte nicht gelesen werden: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(data) {
		return nil, errors.New("Lexware API hat ungültiges JSON zurückgegeben")
	}
	return json.RawMessage(data), nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	const maxAttempts = 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := c.wait(req.Context()); err != nil {
			return nil, err
		}

		current := req.Clone(req.Context())
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			current.Body = body
		}

		resp, err := c.httpClient.Do(current)
		if err != nil {
			if attempt+1 < maxAttempts && retryableMethod(req.Method) {
				if err := sleepContext(req.Context(), backoff(attempt)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, fmt.Errorf("Lexware API ist nicht erreichbar: %w", err)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("API-Fehlerantwort konnte nicht gelesen werden: %w", readErr)
		}
		parsedErr := parseError(resp.StatusCode, req.Method, req.URL.String(), data)
		if attempt+1 < maxAttempts && shouldRetry(req.Method, resp.StatusCode) {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			if err := sleepContext(req.Context(), delay); err != nil {
				return nil, err
			}
			continue
		}
		return nil, parsedErr
	}
	return nil, errors.New("Lexware API konnte nach mehreren Versuchen nicht erreicht werden")
}

func ReadJSONResponse(resp *http.Response) (json.RawMessage, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("API-Antwort konnte nicht gelesen werden: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(data) {
		return nil, errors.New("Lexware API hat ungültiges JSON zurückgegeben")
	}
	return json.RawMessage(data), nil
}

func (c *Client) wait(ctx context.Context) error {
	if c.minInterval <= 0 {
		return nil
	}
	c.limitMu.Lock()
	defer c.limitMu.Unlock()
	wait := time.Until(c.lastRequest.Add(c.minInterval))
	if wait > 0 {
		if err := sleepContext(ctx, wait); err != nil {
			return err
		}
	}
	c.lastRequest = time.Now()
	return nil
}

func retryableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func shouldRetry(method string, status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	return retryableMethod(method) && (status == http.StatusBadGateway || status == http.StatusServiceUnavailable)
}

func backoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * 500 * time.Millisecond
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return backoff(attempt)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
