package docintel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/automatiza-mg/docintel/poller"
)

// AzureBlobSource especifica um container (opcionalmente filtrado por prefixo) cujos
// documentos serão processados em lote.
type AzureBlobSource struct {
	ContainerURL string `json:"containerUrl"`
	Prefix       string `json:"prefix,omitempty"`
}

// AzureBlobFileListSource especifica documentos a serem processados em lote por meio de
// um arquivo JSONL armazenado na raiz do container.
type AzureBlobFileListSource struct {
	ContainerURL string `json:"containerUrl"`
	FileList     string `json:"fileList"`
}

// AnalyzeBatchParams agrupa os parâmetros para iniciar uma análise em lote.
//
// Exatamente uma fonte deve ser informada: [AzureBlobSource] (todos os documentos de um
// container ou prefixo) ou [AzureBlobFileListSource] (documentos específicos listados em
// um arquivo JSONL). Caso ambas ou nenhuma sejam informadas, [Client.AnalyzeBatch] retorna
// [ErrInvalidBatchRequest].
//
// Model e OutputFormat usam, respectivamente, [ModelLayout] e [ContentFormatMarkdown]
// quando vazios. Locale é opcional: quando vazio, a Azure detecta o idioma
// automaticamente. Esses três campos são enviados como parâmetros de query e não
// fazem parte do corpo JSON da requisição.
type AnalyzeBatchParams struct {
	AzureBlobSource         *AzureBlobSource         `json:"azureBlobSource,omitempty"`
	AzureBlobFileListSource *AzureBlobFileListSource `json:"azureBlobFileListSource,omitempty"`

	ResultContainerURL string `json:"resultContainerUrl"`
	ResultPrefix       string `json:"resultPrefix,omitempty"`
	OverwriteExisting  bool   `json:"overwriteExisting,omitempty"`

	Model        Model         `json:"-"`
	Locale       string        `json:"-"`
	OutputFormat ContentFormat `json:"-"`
}

// Valida os parâmetros de forma defensiva, retornando [ErrInvalidBatchRequest]
// com contexto quando alguma condição não é satisfeita.
func (p AnalyzeBatchParams) validate() error {
	switch {
	case p.AzureBlobSource == nil && p.AzureBlobFileListSource == nil:
		return fmt.Errorf("%w: a source is required", ErrInvalidBatchRequest)
	case p.AzureBlobSource != nil && p.AzureBlobFileListSource != nil:
		return fmt.Errorf("%w: only one source is allowed", ErrInvalidBatchRequest)
	}

	if p.AzureBlobSource != nil && p.AzureBlobSource.ContainerURL == "" {
		return fmt.Errorf("%w: azureBlobSource.containerUrl is required", ErrInvalidBatchRequest)
	}
	if p.AzureBlobFileListSource != nil {
		if p.AzureBlobFileListSource.ContainerURL == "" {
			return fmt.Errorf("%w: azureBlobFileListSource.containerUrl is required", ErrInvalidBatchRequest)
		}
		if p.AzureBlobFileListSource.FileList == "" {
			return fmt.Errorf("%w: azureBlobFileListSource.fileList is required", ErrInvalidBatchRequest)
		}
	}

	if p.ResultContainerURL == "" {
		return fmt.Errorf("%w: resultContainerUrl is required", ErrInvalidBatchRequest)
	}

	return nil
}

// AnalyzeBatch inicia a análise em lote de documentos armazenados no Azure Blob Storage,
// extraindo texto no formato configurado usando a API da Azure Document Intelligence.
//
// Os documentos não são enviados diretamente: a fonte e o destino dos resultados são
// containers do Blob Storage informados em params. Os resultados de cada documento são
// gravados como arquivos no container de destino e não fazem parte da resposta.
//
// Retorna o local da operação para ser consultado usando [Client.GetBatchResult].
// Retorna [ErrInvalidBatchRequest] caso os parâmetros sejam inválidos e
// [ErrMissingOperationLocation] caso a resposta não informe a location.
func (c *Client) AnalyzeBatch(ctx context.Context, params AnalyzeBatchParams) (string, error) {
	if err := params.validate(); err != nil {
		return "", err
	}

	body, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshal batch request: %w", err)
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
	url := fmt.Sprintf("%s/documentintelligence/documentModels/%s:analyzeBatch?%s", endpoint, model, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Retorna o erro (com o corpo da resposta) caso status seja diferente de 202.
	if res.StatusCode != http.StatusAccepted {
		return "", newStatusError(res)
	}

	operationLocation := res.Header.Get("Operation-Location")
	if operationLocation == "" {
		return "", ErrMissingOperationLocation
	}
	return operationLocation, nil
}

// GetBatchResult retorna o status e, quando concluído, o resultado da análise em lote.
//
// Retorna [ErrOperationNotFound] caso a operação não exista mais.
func (c *Client) GetBatchResult(ctx context.Context, location string) (*BatchAnalyzeOperation, error) {
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
		var op BatchAnalyzeOperation
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

// PollBatchResult consulta a operação em lote repetidamente até que ela atinja um
// status terminal, retornando o resultado da análise.
//
// O intervalo (padrão 2s) e o tempo máximo de espera (padrão 30min) são
// configurados por [WithPollInterval] e [WithPollTimeout]; o deadline de ctx
// também é respeitado. Erros HTTP temporários (ver [StatusError.Retryable]) são
// reconsultados, aguardando o header Retry-After quando informado. Retorna
// [*AnalyzeError] caso a análise termine em falha e [poller.ErrTimeout] caso o
// tempo limite seja atingido.
func (c *Client) PollBatchResult(ctx context.Context, location string, opts ...PollOption) (*BatchAnalyzeOperation, error) {
	cfg := newPollConfig(defaultBatchPollTimeout, opts)
	p := poller.New(cfg.interval, cfg.timeout, func(ctx context.Context) poller.Result[*BatchAnalyzeOperation] {
		op, err := c.GetBatchResult(ctx, location)
		if err != nil {
			var statusErr *StatusError
			if errors.As(err, &statusErr) && statusErr.Retryable() {
				return poller.Result[*BatchAnalyzeOperation]{Done: false, Delay: statusErr.RetryAfter}
			}
			return poller.Result[*BatchAnalyzeOperation]{Done: true, Err: err}
		}

		switch op.Status {
		case StatusCompleted, StatusSucceeded:
			return poller.Result[*BatchAnalyzeOperation]{Done: true, Value: op}
		case StatusFailed, StatusCanceled, StatusSkipped:
			return poller.Result[*BatchAnalyzeOperation]{Done: true, Err: &AnalyzeError{Status: op.Status, Err: op.Error}}
		case StatusRunning, StatusNotStarted:
			return poller.Result[*BatchAnalyzeOperation]{Done: false}
		default:
			return poller.Result[*BatchAnalyzeOperation]{Done: true, Err: fmt.Errorf("unexpected status: %s", op.Status)}
		}
	})

	return p.Poll(ctx)
}
