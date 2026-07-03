package docintel

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStatusError_Retryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{name: "429 too many requests", statusCode: 429, want: true},
		{name: "500 internal server error", statusCode: 500, want: true},
		{name: "502 bad gateway", statusCode: 502, want: true},
		{name: "503 service unavailable", statusCode: 503, want: true},
		{name: "504 gateway timeout", statusCode: 504, want: true},
		{name: "501 not implemented", statusCode: 501, want: false},
		{name: "400 bad request", statusCode: 400, want: false},
		{name: "401 unauthorized", statusCode: 401, want: false},
		{name: "404 not found", statusCode: 404, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := &StatusError{StatusCode: tt.statusCode}
			if got := err.Retryable(); got != tt.want {
				t.Fatalf("Retryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "empty", value: "", want: 0},
		{name: "seconds", value: "5", want: 5 * time.Second},
		{name: "zero seconds", value: "0", want: 0},
		{name: "negative seconds", value: "-1", want: 0},
		{name: "invalid", value: "soon", want: 0},
		{name: "http date in the past", value: "Mon, 02 Jan 2006 15:04:05 GMT", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseRetryAfter(tt.value); got != tt.want {
				t.Fatalf("parseRetryAfter(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	t.Parallel()

	value := time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat)
	got := parseRetryAfter(value)
	if got <= 0 || got > 10*time.Second {
		t.Fatalf("parseRetryAfter(%q) = %v, want within (0s, 10s]", value, got)
	}
}

func TestNewStatusError_LimitsBody(t *testing.T) {
	t.Parallel()

	res := &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Retry-After": []string{"3"}},
		Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", maxErrorBody+1))),
	}

	err := newStatusError(res)
	if len(err.Body) != maxErrorBody {
		t.Fatalf("len(Body) = %d, want %d", len(err.Body), maxErrorBody)
	}
	if err.StatusCode != http.StatusBadGateway {
		t.Fatalf("StatusCode = %d, want %d", err.StatusCode, http.StatusBadGateway)
	}
	if err.RetryAfter != 3*time.Second {
		t.Fatalf("RetryAfter = %v, want 3s", err.RetryAfter)
	}
}
