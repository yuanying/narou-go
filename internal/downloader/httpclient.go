package downloader

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) narou-go"

// HTTPClient is a small polite HTTP wrapper for downloader implementations.
type HTTPClient struct {
	Client    *http.Client
	UserAgent string
	Wait      time.Duration
	Retries   int
	Backoff   time.Duration

	lastRequest time.Time
}

// NewHTTPClient returns a client with timeout, User-Agent, wait, and retry defaults.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client:    &http.Client{Timeout: 30 * time.Second},
		UserAgent: DefaultUserAgent,
		Wait:      700 * time.Millisecond,
		Retries:   5,
		Backoff:   2 * time.Second,
	}
}

// GetString retrieves a text resource.
func (c *HTTPClient) GetString(ctx context.Context, url string) (string, error) {
	body, _, err := c.GetBytes(ctx, url)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// GetBytes retrieves a binary resource and returns its Content-Type.
func (c *HTTPClient) GetBytes(ctx context.Context, url string) ([]byte, string, error) {
	var lastErr error
	retries := c.Retries
	for attempt := 0; attempt <= retries; attempt++ {
		if err := c.wait(ctx); err != nil {
			return nil, "", err
		}

		body, contentType, statusCode, err := c.doGet(ctx, url)
		if err == nil {
			return body, contentType, nil
		}
		lastErr = err
		if statusCode != http.StatusTooManyRequests && (statusCode < 500 || statusCode == http.StatusNotFound) {
			break
		}
		if attempt < retries {
			if err := sleepContext(ctx, c.Backoff*time.Duration(attempt+1)); err != nil {
				return nil, "", err
			}
		}
	}

	return nil, "", lastErr
}

func (c *HTTPClient) doGet(ctx context.Context, url string) ([]byte, string, int, error) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("User-Agent", c.userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, err
	}

	return body, resp.Header.Get("Content-Type"), resp.StatusCode, nil
}

func (c *HTTPClient) wait(ctx context.Context) error {
	if c.Wait <= 0 || c.lastRequest.IsZero() {
		c.lastRequest = time.Now()
		return nil
	}
	wait := c.Wait - time.Since(c.lastRequest)
	if wait > 0 {
		if err := sleepContext(ctx, wait); err != nil {
			return err
		}
	}
	c.lastRequest = time.Now()

	return nil
}

func (c *HTTPClient) userAgent() string {
	if c.UserAgent == "" {
		return DefaultUserAgent
	}

	return c.UserAgent
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// MediaTypeToExtension maps Content-Type to a file extension.
func MediaTypeToExtension(contentType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(mediaType) {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/gif":
		return ".gif", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", fmt.Errorf("unsupported media type: %s", contentType)
	}
}

// MediaTypeFromPath returns a supported image media type from a path extension.
func MediaTypeFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
