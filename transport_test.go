package docintel

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransport_InjectsHeader(t *testing.T) {
	t.Parallel()

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(apiKeyHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &httpTransport{apiKey: "secret-key"}}

	res, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer res.Body.Close()

	if got != "secret-key" {
		t.Fatalf("expected header %q = %q, got %q", apiKeyHeader, "secret-key", got)
	}
}

func TestTransport_DoesNotMutate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr := &httpTransport{apiKey: "secret-key"}
	res, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer res.Body.Close()

	if got := req.Header.Get(apiKeyHeader); got != "" {
		t.Fatalf("expected original request to be unmodified, but header was %q", got)
	}
}
