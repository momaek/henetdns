package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wentx/henetdns/internal/errs"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	userAgent  string
	attempts   int
	backoffs   []time.Duration
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func New(baseURL string, timeout time.Duration, jar http.CookieJar, userAgent string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if userAgent == "" {
		userAgent = "henetdns/0.1"
	}
	return &Client{
		baseURL: u,
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
		userAgent: userAgent,
		attempts:  3,
		backoffs:  []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second},
	}, nil
}

func (c *Client) BaseURL() *url.URL {
	return c.baseURL
}

func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

func (c *Client) Get(ctx context.Context, path string, referer string) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, referer)
}

func (c *Client) PostForm(ctx context.Context, path string, form url.Values, referer string) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, strings.NewReader(form.Encode()), referer)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, referer string) (*Response, error) {
	var payload []byte
	if body != nil {
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		payload = b
	}

	var lastErr error
	for attempt := 1; attempt <= c.attempts; attempt++ {
		ref, err := url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("parse request path %q: %w", path, err)
		}
		reqURL := c.baseURL.ResolveReference(ref)
		var reqBody io.Reader
		if payload != nil {
			reqBody = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), reqBody)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if req.URL != nil {
				req.Header.Set("Origin", c.baseURL.Scheme+"://"+c.baseURL.Host)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if !shouldRetryError(err) || attempt == c.attempts {
				return nil, fmt.Errorf("http request failed: %w: %w", err, errs.ErrRemote)
			}
			time.Sleep(c.backoff(attempt))
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response body: %w: %w", readErr, errs.ErrRemote)
		}

		if shouldRetryStatus(resp.StatusCode) && attempt < c.attempts {
			time.Sleep(c.backoff(attempt))
			continue
		}

		if resp.StatusCode >= 400 {
			return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: respBody}, fmt.Errorf("unexpected status %d: %w", resp.StatusCode, errs.ErrRemote)
		}

		return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: respBody}, nil
	}

	return nil, fmt.Errorf("request retries exhausted: %w: %w", lastErr, errs.ErrRemote)
}

func shouldRetryStatus(status int) bool {
	return status == 429 || status >= 500
}

func shouldRetryError(err error) bool {
	if nerr, ok := err.(net.Error); ok {
		return nerr.Timeout() || nerr.Temporary()
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe")
}

func (c *Client) backoff(attempt int) time.Duration {
	idx := attempt - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(c.backoffs) {
		idx = len(c.backoffs) - 1
	}
	return c.backoffs[idx]
}
