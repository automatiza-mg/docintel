package docintel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ErrInvalidAnalyzeRequest é retornado quando os parâmetros de uma análise de documento são inválidos.
var ErrInvalidAnalyzeRequest = errors.New("docintel: invalid analyze request")

// ErrInvalidBatchRequest é retornado quando os parâmetros de uma análise em lote são inválidos.
var ErrInvalidBatchRequest = errors.New("docintel: invalid batch request")

// ErrOperationNotFound é retornado quando a operação consultada não existe
// mais. A Azure retém os resultados por tempo limitado (tipicamente 24h);
// locations expiradas também produzem este erro.
var ErrOperationNotFound = errors.New("docintel: operation not found")

// ErrMissingOperationLocation é retornado quando a Azure aceita a análise mas a
// resposta não contém o header Operation-Location.
var ErrMissingOperationLocation = errors.New("docintel: response missing Operation-Location header")

// AzureError representa o objeto de erro retornado pela API da Azure. É uma
// estrutura de dados, não um error do Go: quando uma análise falha, é carregado
// por [AnalyzeError].
type AzureError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Target  string       `json:"target,omitempty"`
	Details []AzureError `json:"details,omitempty"`
	// InnerError carrega detalhes adicionais em formato variável por modelo.
	InnerError json.RawMessage `json:"innererror,omitempty"`
}

// AnalyzeError é retornado quando há falha ao processar algum documento.
type AnalyzeError struct {
	Status Status
	Err    *AzureError
}

func (e *AnalyzeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("docintel: analyze %s: %s (%s)", e.Status, e.Err.Message, e.Err.Code)
	}
	return fmt.Sprintf("docintel: analyze %s", e.Status)
}

// StatusError representa uma resposta HTTP com status inesperado.
type StatusError struct {
	StatusCode int
	// Body contém o corpo da resposta, limitado a 64 KiB.
	Body string
	// RetryAfter é o tempo de espera informado no header Retry-After da
	// resposta, ou zero quando ausente.
	RetryAfter time.Duration
}

// maxErrorBody limita a leitura do corpo de respostas de erro.
const maxErrorBody = 64 << 10

// newStatusError cria um [StatusError] a partir de uma resposta HTTP de erro,
// lendo o corpo (limitado a 64 KiB) e o header Retry-After.
func newStatusError(res *http.Response) *StatusError {
	b, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBody))
	return &StatusError{
		StatusCode: res.StatusCode,
		Body:       string(b),
		RetryAfter: parseRetryAfter(res.Header.Get("Retry-After")),
	}
}

// parseRetryAfter interpreta o valor do header Retry-After, que pode ser um
// número de segundos ou uma data HTTP. Retorna zero quando ausente ou inválido.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("docintel: unexpected status: %d (%s)", e.StatusCode, e.Body)
}

// Retryable indica se o erro pode ser repetido. Status 429 (Too Many Requests),
// 500, 502, 503 e 504 são considerados temporários.
func (e *StatusError) Retryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}
