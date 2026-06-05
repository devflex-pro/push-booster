package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type OTPDelivery interface {
	SendOTP(ctx context.Context, email string, otp string) error
}

type LogDelivery struct {
	log *slog.Logger
}

func NewLogDelivery(log *slog.Logger) *LogDelivery {
	return &LogDelivery{log: log}
}

func (d *LogDelivery) SendOTP(ctx context.Context, email string, otp string) error {
	d.log.Info("auth otp generated", slog.String("email", email), slog.String("otp", otp))
	return nil
}

type ResendConfig struct {
	APIKey       string
	APIURL       string
	From         string
	MaxAttempts  int
	RetryBackoff time.Duration
	Logger       *slog.Logger
}

type ResendDelivery struct {
	client *http.Client
	cfg    ResendConfig
}

func NewResendDelivery(cfg ResendConfig) *ResendDelivery {
	return NewResendDeliveryWithClient(http.DefaultClient, cfg)
}

func NewResendDeliveryWithClient(client *http.Client, cfg ResendConfig) *ResendDelivery {
	if client == nil {
		client = http.DefaultClient
	}
	if strings.TrimSpace(cfg.APIURL) == "" {
		cfg.APIURL = "https://api.resend.com/emails"
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 500 * time.Millisecond
	}
	return &ResendDelivery{client: client, cfg: cfg}
}

func (d *ResendDelivery) SendOTP(ctx context.Context, email string, otp string) error {
	if strings.TrimSpace(d.cfg.APIKey) == "" {
		return errors.New("resend api key is required")
	}

	var lastErr error
	for attempt := 1; attempt <= d.cfg.MaxAttempts; attempt++ {
		messageID, retryable, err := d.sendOnce(ctx, email, otp)
		if err == nil {
			d.logInfo(
				"otp email sent",
				slog.String("provider", "resend"),
				slog.String("email", email),
				slog.String("message_id", messageID),
				slog.Int("attempt", attempt),
			)
			return nil
		}
		lastErr = err
		if !retryable || attempt == d.cfg.MaxAttempts {
			d.logWarn(
				"otp email send failed",
				slog.String("provider", "resend"),
				slog.String("email", email),
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", d.cfg.MaxAttempts),
				slog.Bool("retryable", retryable),
				slog.String("error", err.Error()),
			)
			return err
		}

		timer := time.NewTimer(d.cfg.RetryBackoff)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func (d *ResendDelivery) sendOnce(ctx context.Context, email string, otp string) (string, bool, error) {
	payload := map[string]any{
		"from":    d.cfg.From,
		"to":      []string{email},
		"subject": "Push Booster login code",
		"text": fmt.Sprintf(
			"Your Push Booster login code is: %s\nThis code expires soon.",
			otp,
		),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.cfg.APIURL, bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+d.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := d.client.Do(req)
	if err != nil {
		return "", true, err
	}
	defer func() {
		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			d.logWarn("failed to drain resend response body", slog.String("error", err.Error()))
		}
		if err := res.Body.Close(); err != nil {
			d.logWarn("failed to close resend response body", slog.String("error", err.Error()))
		}
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		responseBody, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			return "", retryableResendStatus(res.StatusCode), errors.Join(
				fmt.Errorf("resend returned status %d", res.StatusCode),
				readErr,
			)
		}
		return "", retryableResendStatus(res.StatusCode), fmt.Errorf(
			"resend returned status %d: %s",
			res.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", false, err
	}
	return response.ID, false, nil
}

func retryableResendStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func (d *ResendDelivery) logInfo(message string, attrs ...slog.Attr) {
	if d.cfg.Logger != nil {
		d.cfg.Logger.LogAttrs(context.Background(), slog.LevelInfo, message, attrs...)
	}
}

func (d *ResendDelivery) logWarn(message string, attrs ...slog.Attr) {
	if d.cfg.Logger != nil {
		d.cfg.Logger.LogAttrs(context.Background(), slog.LevelWarn, message, attrs...)
	}
}
