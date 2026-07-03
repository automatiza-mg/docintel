package docintel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
// Retorna [ErrInvalidAnalyzeRequest] caso os parâmetros sejam inválidos.
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

	// Lê o corpo da requisição e retorna o erro caso status seja diferente de 202.
	if res.StatusCode != http.StatusAccepted {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		return "", &StatusError{StatusCode: res.StatusCode, Body: string(b)}
	}

	opLoc := res.Header.Get("Operation-Location")
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
		b, _ := io.ReadAll(res.Body)
		return nil, &StatusError{StatusCode: res.StatusCode, Body: string(b)}
	}
}
