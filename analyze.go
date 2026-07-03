package docintel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/automatiza-mg/docintel/poller"
)

// AnalyzeDocumentParams agrupa os parâmetros para analisar um documento individual.
//
// Document e ContentType são obrigatórios. Model e OutputFormat usam,
// respectivamente, [ModelLayout] e [ContentFormatMarkdown] quando vazios. Locale
// é opcional: quando vazio, a Azure detecta o idioma automaticamente.
type AnalyzeDocumentParams struct {
	Document     io.Reader
	ContentType  string
	Model        Model
	Locale       string
	OutputFormat ContentFormat
}

// Valida os parâmetros de forma defensiva, retornando [ErrInvalidAnalyzeRequest]
// com contexto quando alguma condição não é satisfeita.
func (p AnalyzeDocumentParams) validate() error {
	if p.Document == nil {
		return fmt.Errorf("%w: document is required", ErrInvalidAnalyzeRequest)
	}
	if p.ContentType == "" {
		return fmt.Errorf("%w: contentType is required", ErrInvalidAnalyzeRequest)
	}
	return nil
}

// AnalyzeDocument inicia a análise de um documento (extraindo o conteúdo no
// formato configurado) a partir de um [io.Reader] usando a API da Azure Document
// Intelligence.
//
// Retorna o local da operação para ser consultado usando [Client.GetAnalyzeResult].
// Retorna [ErrInvalidAnalyzeRequest] caso os parâmetros sejam inválidos e
// [ErrMissingOperationLocation] caso a resposta não informe a location.
func (c *Client) AnalyzeDocument(ctx context.Context, params AnalyzeDocumentParams) (string, error) {
	if err := params.validate(); err != nil {
		return "", err
	}

	model := params.Model
	if model == "" {
		model = defaultModel
	}
	outputFormat := params.OutputFormat
	if outputFormat == "" {
		outputFormat = defaultOutputFormat
	}

	q := make(url.Values)
	if params.Locale != "" {
		q.Set("locale", params.Locale)
	}
	q.Set("api-version", c.apiVersion)
	q.Set("outputContentFormat", string(outputFormat))

	endpoint := strings.TrimSuffix(c.endpoint, "/")
	url := fmt.Sprintf("%s/documentintelligence/documentModels/%s:analyze?%s", endpoint, model, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, params.Document)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", params.ContentType)

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Retorna o erro (com o corpo da resposta) caso status seja diferente de 202.
	if res.StatusCode != http.StatusAccepted {
		return "", newStatusError(res)
	}

	opLoc := res.Header.Get("Operation-Location")
	if opLoc == "" {
		return "", ErrMissingOperationLocation
	}
	return opLoc, nil
}

// GetAnalyzeResult retorna o status e, quando concluído, o resultado da análise do documento.
//
// Retorna [ErrOperationNotFound] caso a operação não exista mais.
func (c *Client) GetAnalyzeResult(ctx context.Context, location string) (*AnalyzeOperation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		var op AnalyzeOperation
		err := json.NewDecoder(res.Body).Decode(&op)
		if err != nil {
			return nil, err
		}
		return &op, nil
	case http.StatusNotFound:
		return nil, ErrOperationNotFound
	default:
		return nil, newStatusError(res)
	}
}

// PollResult consulta a operação repetidamente até que ela atinja um status
// terminal, retornando o resultado da análise.
//
// O intervalo (padrão 2s) e o tempo máximo de espera (padrão 5min) são
// configurados por [WithPollInterval] e [WithPollTimeout]; o deadline de ctx
// também é respeitado. Erros HTTP temporários (ver [StatusError.Retryable]) são
// reconsultados, aguardando o header Retry-After quando informado. Retorna
// [*AnalyzeError] caso a análise termine em falha e [poller.ErrTimeout] caso o
// tempo limite seja atingido.
func (c *Client) PollResult(ctx context.Context, location string, opts ...PollOption) (*AnalyzeOperation, error) {
	cfg := newPollConfig(defaultPollTimeout, opts)
	p := poller.New(cfg.interval, cfg.timeout, func(ctx context.Context) poller.Result[*AnalyzeOperation] {
		op, err := c.GetAnalyzeResult(ctx, location)
		if err != nil {
			var statusErr *StatusError
			if errors.As(err, &statusErr) && statusErr.Retryable() {
				return poller.Result[*AnalyzeOperation]{Done: false, Delay: statusErr.RetryAfter}
			}
			return poller.Result[*AnalyzeOperation]{Done: true, Err: err}
		}

		switch op.Status {
		case StatusSucceeded:
			return poller.Result[*AnalyzeOperation]{Done: true, Value: op}
		case StatusFailed, StatusCanceled, StatusSkipped:
			return poller.Result[*AnalyzeOperation]{Done: true, Err: &AnalyzeError{Status: op.Status, Err: op.Error}}
		case StatusRunning, StatusNotStarted:
			return poller.Result[*AnalyzeOperation]{Done: false}
		default:
			return poller.Result[*AnalyzeOperation]{Done: true, Err: fmt.Errorf("unexpected status: %s", op.Status)}
		}
	})

	return p.Poll(ctx)
}
