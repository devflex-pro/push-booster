package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

var ErrNil = errors.New("redis nil")

type Config struct {
	Addr     string
	Password string
	DB       int
	Timeout  time.Duration
}

type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Client{cfg: cfg}
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.do(ctx, "GET", key)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *Client) SetEX(ctx context.Context, key string, value string, ttl time.Duration) error {
	seconds := int64(ttl.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	_, err := c.do(ctx, "SET", key, value, "EX", strconv.FormatInt(seconds, 10))
	return err
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	value, err := c.do(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}
	count, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse redis increment: %w", err)
	}
	return count, nil
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	seconds := int64(ttl.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	_, err := c.do(ctx, "EXPIRE", key, strconv.FormatInt(seconds, 10))
	return err
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.do(ctx, "PING")
	return err
}

func (c *Client) do(ctx context.Context, args ...string) (string, error) {
	dialer := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return "", fmt.Errorf("connect redis: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.cfg.Timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return "", fmt.Errorf("set redis deadline: %w", err)
	}

	reader := bufio.NewReader(conn)
	if c.cfg.Password != "" {
		if err := writeCommand(conn, "AUTH", c.cfg.Password); err != nil {
			return "", err
		}
		if _, err := readResponse(reader); err != nil {
			return "", fmt.Errorf("auth redis: %w", err)
		}
	}
	if c.cfg.DB > 0 {
		if err := writeCommand(conn, "SELECT", strconv.Itoa(c.cfg.DB)); err != nil {
			return "", err
		}
		if _, err := readResponse(reader); err != nil {
			return "", fmt.Errorf("select redis db: %w", err)
		}
	}
	if err := writeCommand(conn, args...); err != nil {
		return "", err
	}
	return readResponse(reader)
}

func writeCommand(conn net.Conn, args ...string) error {
	var builder strings.Builder
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(args)))
	builder.WriteString("\r\n")
	for _, arg := range args {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(len(arg)))
		builder.WriteString("\r\n")
		builder.WriteString(arg)
		builder.WriteString("\r\n")
	}
	if _, err := conn.Write([]byte(builder.String())); err != nil {
		return fmt.Errorf("write redis command: %w", err)
	}
	return nil
}

func readResponse(reader *bufio.Reader) (string, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return "", fmt.Errorf("read redis response: %w", err)
	}
	switch prefix {
	case '+':
		line, err := readLine(reader)
		if err != nil {
			return "", err
		}
		return line, nil
	case '-':
		line, err := readLine(reader)
		if err != nil {
			return "", err
		}
		return "", errors.New(line)
	case ':':
		return readLine(reader)
	case '$':
		line, err := readLine(reader)
		if err != nil {
			return "", err
		}
		size, err := strconv.Atoi(line)
		if err != nil {
			return "", fmt.Errorf("parse redis bulk size: %w", err)
		}
		if size == -1 {
			return "", ErrNil
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return "", fmt.Errorf("read redis bulk: %w", err)
		}
		return string(buf[:size]), nil
	default:
		return "", fmt.Errorf("unsupported redis response prefix %q", prefix)
	}
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read redis line: %w", err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}
