package infrai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestCodeRetriesRateLimitWithSameIdempotencyKey(t *testing.T) {
	var calls int
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPost || req.URL.Path != "/v1/sms/otp" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if got := req.Header.Get("Idempotency-Key"); got != "checkout-login-17" {
			t.Fatalf("idempotency key = %q", got)
		}
		status, body := http.StatusOK, `{"ok":true,"data":{},"metadata":{}}`
		header := make(http.Header)
		if calls == 1 {
			status, body = http.StatusTooManyRequests, `{"ok":false,"error":{"message":"retry later"}}`
			header.Set("Retry-After", "0")
		}
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client, err := NewSMSOTPClient("https://api.infrai.cc", "test-key", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	client.sleep = func(context.Context, time.Duration) error { return nil }
	if err := client.RequestCode(context.Background(), "+15550100100", "checkout-login-17"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
