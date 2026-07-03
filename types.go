package docintel

import "time"

// ContentFormat é o formato do conteúdo extraído, informado no parâmetro
// outputContentFormat das operações de análise.
type ContentFormat string

const (
	// ContentFormatText extrai o conteúdo como texto puro.
	ContentFormatText ContentFormat = "text"
	// ContentFormatMarkdown extrai o conteúdo em Markdown.
	ContentFormatMarkdown ContentFormat = "markdown"
)

// Model identifica o modelo usado na análise (modelId).
//
// Além das constantes de modelos prebuilt declaradas abaixo, qualquer ID de
// modelo custom pode ser usado convertendo a string para [Model].
type Model string

const (
	// ModelRead detecta linhas, palavras, localizações e idiomas.
	ModelRead Model = "prebuilt-read"
	// ModelLayout extrai texto, tabelas, marcas de seleção e estrutura.
	ModelLayout Model = "prebuilt-layout"
	// ModelInvoice extrai campos de notas fiscais e faturas.
	ModelInvoice Model = "prebuilt-invoice"
	// ModelReceipt extrai campos de recibos.
	ModelReceipt Model = "prebuilt-receipt"
	// ModelIDDocument extrai campos de documentos de identidade.
	ModelIDDocument Model = "prebuilt-idDocument"
	// ModelBusinessCard extrai campos de cartões de visita.
	ModelBusinessCard Model = "prebuilt-businessCard"
	// ModelContract extrai campos de contratos.
	ModelContract Model = "prebuilt-contract"
	// ModelTaxUSW2 extrai campos de formulários fiscais W-2 dos EUA.
	ModelTaxUSW2 Model = "prebuilt-tax.us.w2"
)

// Status representa o estado de uma operação de análise da Azure Document Intelligence.
type Status string

const (
	StatusNotStarted Status = "notStarted"
	StatusRunning    Status = "running"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
	StatusCanceled   Status = "canceled"
	StatusSkipped    Status = "skipped"
	// StatusCompleted é o status terminal de sucesso de uma operação em lote.
	// Os documentos individuais usam os demais status (ex: [StatusSucceeded]).
	StatusCompleted Status = "completed"
)

type AnalyzeOperation struct {
	Status              Status        `json:"status"`
	CreatedDateTime     time.Time     `json:"createdDateTime"`
	LastUpdatedDateTime time.Time     `json:"lastUpdatedDateTime"`
	AnalyzeResult       AnalyzeResult `json:"analyzeResult"`
	Error               *AzureError   `json:"error,omitempty"`
}

// AnalyzeResult representa o resultado da análise de um documento.
type AnalyzeResult struct {
	APIVersion      string                 `json:"apiVersion"`
	ModelID         string                 `json:"modelId"`
	Content         string                 `json:"content"`
	StringIndexType string                 `json:"stringIndexType"`
	ContentFormat   string                 `json:"contentFormat"`
	Pages           []DocumentPage         `json:"pages,omitempty"`
	Paragraphs      []DocumentParagraph    `json:"paragraphs,omitempty"`
	Tables          []DocumentTable        `json:"tables,omitempty"`
	Figures         []DocumentFigure       `json:"figures,omitempty"`
	Sections        []DocumentSection      `json:"sections,omitempty"`
	KeyValuePairs   []DocumentKeyValuePair `json:"keyValuePairs,omitempty"`
	Languages       []DocumentLanguage     `json:"languages,omitempty"`
	Styles          []DocumentStyle        `json:"styles,omitempty"`
	Documents       []AnalyzedDocument     `json:"documents,omitempty"`
	Warnings        []Warning              `json:"warnings,omitempty"`
}

// Span representa um intervalo (offset e comprimento) no conteúdo concatenado do resultado.
type Span struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
}

// BoundingRegion associa um conteúdo a uma região (polígono) em uma página.
type BoundingRegion struct {
	PageNumber int       `json:"pageNumber"`
	Polygon    []float64 `json:"polygon,omitempty"`
}

// DocumentPage representa uma página analisada do documento.
type DocumentPage struct {
	PageNumber int            `json:"pageNumber"`
	Angle      float64        `json:"angle,omitempty"`
	Width      float64        `json:"width,omitempty"`
	Height     float64        `json:"height,omitempty"`
	Unit       string         `json:"unit,omitempty"`
	Spans      []Span         `json:"spans,omitempty"`
	Words      []DocumentWord `json:"words,omitempty"`
	Lines      []DocumentLine `json:"lines,omitempty"`
}

// DocumentWord representa uma palavra detectada em uma página.
type DocumentWord struct {
	Content    string    `json:"content"`
	Polygon    []float64 `json:"polygon,omitempty"`
	Span       Span      `json:"span"`
	Confidence float64   `json:"confidence"`
}

// DocumentLine representa uma linha de texto detectada em uma página.
type DocumentLine struct {
	Content string    `json:"content"`
	Polygon []float64 `json:"polygon,omitempty"`
	Spans   []Span    `json:"spans,omitempty"`
}

// DocumentParagraph representa um parágrafo extraído do documento.
type DocumentParagraph struct {
	Role            string           `json:"role,omitempty"`
	Content         string           `json:"content"`
	BoundingRegions []BoundingRegion `json:"boundingRegions,omitempty"`
	Spans           []Span           `json:"spans,omitempty"`
}

// DocumentTable representa uma tabela extraída do documento.
type DocumentTable struct {
	RowCount        int                 `json:"rowCount"`
	ColumnCount     int                 `json:"columnCount"`
	Cells           []DocumentTableCell `json:"cells,omitempty"`
	BoundingRegions []BoundingRegion    `json:"boundingRegions,omitempty"`
	Spans           []Span              `json:"spans,omitempty"`
}

// DocumentTableCell representa uma célula de uma tabela.
type DocumentTableCell struct {
	Kind            string           `json:"kind,omitempty"`
	RowIndex        int              `json:"rowIndex"`
	ColumnIndex     int              `json:"columnIndex"`
	RowSpan         int              `json:"rowSpan,omitempty"`
	ColumnSpan      int              `json:"columnSpan,omitempty"`
	Content         string           `json:"content"`
	BoundingRegions []BoundingRegion `json:"boundingRegions,omitempty"`
	Spans           []Span           `json:"spans,omitempty"`
}

// DocumentFigure representa uma figura extraída do documento.
type DocumentFigure struct {
	ID              string           `json:"id,omitempty"`
	Caption         *DocumentCaption `json:"caption,omitempty"`
	BoundingRegions []BoundingRegion `json:"boundingRegions,omitempty"`
	Spans           []Span           `json:"spans,omitempty"`
}

// DocumentCaption representa a legenda de uma figura ou tabela.
type DocumentCaption struct {
	Content         string           `json:"content"`
	BoundingRegions []BoundingRegion `json:"boundingRegions,omitempty"`
	Spans           []Span           `json:"spans,omitempty"`
}

// DocumentSection representa uma seção identificada no documento.
type DocumentSection struct {
	Spans    []Span   `json:"spans,omitempty"`
	Elements []string `json:"elements,omitempty"`
}

// DocumentKeyValuePair representa um par chave-valor extraído do documento.
type DocumentKeyValuePair struct {
	Key        DocumentKeyValueElement  `json:"key"`
	Value      *DocumentKeyValueElement `json:"value,omitempty"`
	Confidence float64                  `json:"confidence"`
}

// DocumentKeyValueElement representa a chave ou o valor de um par chave-valor.
type DocumentKeyValueElement struct {
	Content         string           `json:"content"`
	BoundingRegions []BoundingRegion `json:"boundingRegions,omitempty"`
	Spans           []Span           `json:"spans,omitempty"`
}

// DocumentLanguage representa um idioma detectado no documento.
type DocumentLanguage struct {
	Locale     string  `json:"locale"`
	Spans      []Span  `json:"spans,omitempty"`
	Confidence float64 `json:"confidence"`
}

// DocumentStyle representa um estilo de fonte detectado no documento.
type DocumentStyle struct {
	IsHandwritten   bool    `json:"isHandwritten,omitempty"`
	Color           string  `json:"color,omitempty"`
	BackgroundColor string  `json:"backgroundColor,omitempty"`
	FontStyle       string  `json:"fontStyle,omitempty"`
	FontWeight      string  `json:"fontWeight,omitempty"`
	Spans           []Span  `json:"spans,omitempty"`
	Confidence      float64 `json:"confidence"`
}

// AnalyzedDocument representa um documento extraído pelo modelo.
type AnalyzedDocument struct {
	DocType         string                   `json:"docType"`
	BoundingRegions []BoundingRegion         `json:"boundingRegions,omitempty"`
	Spans           []Span                   `json:"spans,omitempty"`
	Fields          map[string]DocumentField `json:"fields,omitempty"`
	Confidence      float64                  `json:"confidence"`
}

// DocumentField representa um campo extraído de um documento.
type DocumentField struct {
	Type            string           `json:"type"`
	Content         string           `json:"content,omitempty"`
	BoundingRegions []BoundingRegion `json:"boundingRegions,omitempty"`
	Spans           []Span           `json:"spans,omitempty"`
	Confidence      float64          `json:"confidence,omitempty"`
}

// Warning representa um aviso encontrado durante a análise.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Target  string `json:"target,omitempty"`
}

// BatchAnalyzeOperation representa o status de uma operação de análise em lote da
// Azure Document Intelligence.
//
// O resultado não contém o conteúdo extraído dos documentos: cada documento processado
// com sucesso aponta para um arquivo de resultado via [BatchResultDetail.ResultURL].
type BatchAnalyzeOperation struct {
	ResultID            string      `json:"resultId"`
	Status              Status      `json:"status"`
	PercentCompleted    int         `json:"percentCompleted"`
	CreatedDateTime     time.Time   `json:"createdDateTime"`
	LastUpdatedDateTime time.Time   `json:"lastUpdatedDateTime"`
	Result              BatchResult `json:"result"`
	Error               *AzureError `json:"error,omitempty"`
}

// BatchResult agrega o resultado de uma operação de análise em lote.
type BatchResult struct {
	SucceededCount int                 `json:"succeededCount"`
	FailedCount    int                 `json:"failedCount"`
	SkippedCount   int                 `json:"skippedCount"`
	Details        []BatchResultDetail `json:"details,omitempty"`
}

// BatchResultDetail representa o resultado do processamento de um único documento em um lote.
type BatchResultDetail struct {
	SourceURL string      `json:"sourceUrl"`
	ResultURL string      `json:"resultUrl,omitempty"`
	Status    Status      `json:"status"`
	Error     *AzureError `json:"error,omitempty"`
}
