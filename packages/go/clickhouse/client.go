package clickhouse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	URL      string
	Database string
	Username string
	Password string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Exec(ctx context.Context, query string) error {
	_, err := c.do(ctx, query)
	return err
}

func (c *Client) QueryText(ctx context.Context, query string) (string, error) {
	body, err := c.do(ctx, query)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *Client) do(ctx context.Context, query string) ([]byte, error) {
	endpoint, err := c.endpoint()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(query))
	if err != nil {
		return nil, fmt.Errorf("create clickhouse request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute clickhouse query: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, fmt.Errorf("read clickhouse response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("close clickhouse response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("clickhouse status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c *Client) endpoint() (string, error) {
	parsed, err := url.Parse(c.cfg.URL)
	if err != nil {
		return "", fmt.Errorf("parse clickhouse url: %w", err)
	}
	query := parsed.Query()
	if c.cfg.Database != "" {
		query.Set("database", c.cfg.Database)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
