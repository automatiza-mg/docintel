package docintel

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrInvalidAnalyzeRequest é retornado quando os parâmetros de uma análise de documento são inválidos.
var ErrInvalidAnalyzeRequest = errors.New("docintel: invalid analyze request")

// ErrInvalidBatchRequest é retornado quando os parâmetros de uma análise em lote são inválidos.
var ErrInvalidBatchRequest = errors.New("docintel: invalid batch request")

// ErrOperationNotFound é retornado quando a operação consultada não existe mais.
var ErrOperationNotFound = errors.New("docintel: operation not found")

type AzureError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("docintel: unexpected status: %d (%s)", e.StatusCode, e.Body)
}

// Retryable indica se o erro pode ser repetido.
// Status 429 (Too Many Requests) e erros 5xx são considerados temporários.
func (e *StatusError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}
