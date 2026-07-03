package docintel

import (
	"testing"
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
		{name: "503 service unavailable", statusCode: 503, want: true},
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
