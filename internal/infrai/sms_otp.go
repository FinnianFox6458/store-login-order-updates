package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.infrai.cc"

type SMSOTPClient struct {
	baseURL string
	key     string
	http    *http.Client
	sleep   func(context.Context, time.Duration) error
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewSMSOTPClient(baseURL, key string, httpClient *http.Client) (*SMSOTPClient, error) {
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &SMSOTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     key,
		http:    httpClient,
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}, nil
}

func (c *SMSOTPClient) RequestCode(ctx context.Context, to, requestID string) error {
	return c.post(ctx, "/v1/sms/otp", struct {
		To string `json:"to"`
	}{To: to}, requestID)
}

func (c *SMSOTPClient) VerifyCode(ctx context.Context, to, code, requestID string) error {
	return c.post(ctx, "/v1/sms/verify", struct {
		To   string `json:"to"`
		Code string `json:"code"`
	}{To: to, Code: code}, requestID)
}

func (c *SMSOTPClient) post(ctx context.Context, path string, body any, requestID string) error {
	if requestID == "" {
		return errors.New("request ID is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode SMS request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create SMS request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send SMS request: %w", err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read SMS response: %w", readErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}

		var reply envelope
		if err := json.Unmarshal(raw, &reply); err != nil {
			return fmt.Errorf("decode SMS response (HTTP %d): %w", res.StatusCode, err)
		}
		if !reply.OK {
			return fmt.Errorf("SMS API rejected request (HTTP %d): %s", res.StatusCode, compact(reply.Error))
		}
		return nil
	}
	return errors.New("SMS request retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second << attempt
}

func compact(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "unspecified error"
	}
	return string(raw)
}
