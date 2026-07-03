package docintel

import "net/http"

// Option configura um [Client] criado por [NewClient].
type Option func(*Client)

// WithAPIVersion define a versão da API usada nas requisições (padrão "2024-11-30").
func WithAPIVersion(version string) Option {
	return func(c *Client) {
		c.apiVersion = version
	}
}

// WithHTTPClient substitui o [http.Client] usado nas requisições.
//
// Útil para configurar timeouts, proxies ou uma autenticação diferente da API
// key padrão (ex: um [http.RoundTripper] que injeta um token do Azure AD). O
// client informado é usado como está, sem adicionar o header de API key.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.http = client
		}
	}
}
