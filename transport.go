package docintel

import "net/http"

const apiKeyHeader = "Ocp-Apim-Subscription-Key"

// httpTransport é uma implementação de [http.RoundTripper] que adiciona o header
// com API key em toda requisição.
type httpTransport struct {
	http.RoundTripper
	apiKey string
}

func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := t.RoundTripper
	if next == nil {
		next = http.DefaultTransport
	}

	clone := req.Clone(req.Context())
	clone.Header.Set(apiKeyHeader, t.apiKey)
	return next.RoundTrip(clone)
}
