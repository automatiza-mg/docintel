// Package docintel fornece um client para a API da Azure Document Intelligence,
// permitindo extrair o conteúdo textual de documentos de forma individual
// ([Client.AnalyzeDocument]) ou em lote ([Client.AnalyzeBatch]).
//
// As operações são assíncronas: os métodos de análise retornam a location da
// operação, que deve ser consultada com [Client.GetAnalyzeResult] ou
// [Client.GetBatchResult] até que a operação atinja um status terminal.
//
// O modelo, o locale e o formato de saída são definidos por chamada, via
// [AnalyzeDocumentParams] e [AnalyzeBatchParams].
package docintel

import "net/http"

// defaultAPIVersion é a versão da API usada quando nenhuma [Option] é informada.
const defaultAPIVersion = "2024-11-30"

// Valores padrão aplicados aos parâmetros de análise quando não informados.
const (
	defaultModel        = ModelLayout
	defaultOutputFormat = ContentFormatMarkdown
)

// Client é o client da API da Azure Document Intelligence.
type Client struct {
	endpoint   string
	apiVersion string
	http       *http.Client
}

// NewClient cria um [Client] para o endpoint informado, autenticado com a API
// key (header Ocp-Apim-Subscription-Key).
//
// Sem [Option], o client usa a versão de API "2024-11-30". Use [WithAPIVersion]
// para alterá-la ou [WithHTTPClient] para injetar um [http.Client] próprio (ex:
// para autenticação via Azure AD).
func NewClient(endpoint, key string, opts ...Option) *Client {
	c := &Client{
		endpoint:   endpoint,
		apiVersion: defaultAPIVersion,
		http: &http.Client{
			Transport: &httpTransport{
				RoundTripper: http.DefaultTransport,
				apiKey:       key,
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
