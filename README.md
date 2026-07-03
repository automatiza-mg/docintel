# docintel

Client Go para a [Azure Document Intelligence](https://learn.microsoft.com/azure/ai-services/document-intelligence/),
usado para extrair o conteúdo textual de documentos de forma individual ou em
lote. Projetado para ser compartilhado entre múltiplos projetos.

## Instalação

```bash
go get github.com/automatiza-mg/docintel
```

Requer Go 1.26 ou superior.

## Uso

### Análise de um documento

A análise é assíncrona: `AnalyzeDocument` retorna a location da operação, que
deve ser consultada até atingir um status terminal. Use `PollResult` para
aguardar a conclusão automaticamente, ou `GetAnalyzeResult` para consultar o
status uma única vez.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/automatiza-mg/docintel"
)

func main() {
    client := docintel.NewClient(
        os.Getenv("AZURE_DOCINTEL_ENDPOINT"),
        os.Getenv("AZURE_DOCINTEL_KEY"),
    )

    f, err := os.Open("documento.pdf")
    if err != nil {
        panic(err)
    }
    defer f.Close()

    ctx := context.Background()

    location, err := client.AnalyzeDocument(ctx, docintel.AnalyzeDocumentParams{
        Document:     f,
        ContentType:  "application/pdf",
        Model:        docintel.ModelLayout,
        Locale:       "pt-BR",
        OutputFormat: docintel.ContentFormatMarkdown,
    })
    if err != nil {
        panic(err)
    }

    // Aguarda a conclusão, consultando em intervalos regulares (ver Configuração).
    op, err := client.PollResult(ctx, location)
    if err != nil {
        panic(err)
    }

    fmt.Println(op.AnalyzeResult.Content)
}
```

### Análise em lote

Para processar documentos armazenados no Azure Blob Storage sem enviá-los na
requisição, use `AnalyzeBatch` com uma fonte (`AzureBlobSource` ou
`AzureBlobFileListSource`) e um container de destino:

```go
location, err := client.AnalyzeBatch(ctx, docintel.AnalyzeBatchParams{
    AzureBlobSource: &docintel.AzureBlobSource{
        ContainerURL: "https://storage.blob.core.windows.net/in?sas",
        Prefix:       "inputDocs/",
    },
    ResultContainerURL: "https://storage.blob.core.windows.net/out?sas",
    ResultPrefix:       "batchResults/",
    OverwriteExisting:  true,
    Model:              docintel.ModelLayout,       // opcional
    OutputFormat:       docintel.ContentFormatText, // opcional
})
```

Aguarde o resultado com `PollBatchResult` (ou consulte uma vez com
`GetBatchResult`). Os resultados de cada documento são gravados no container de
destino, e não retornados na resposta:

```go
op, err := client.PollBatchResult(ctx, location)
if err != nil {
    panic(err)
}
fmt.Printf("%d succeeded, %d failed\n", op.Result.SucceededCount, op.Result.FailedCount)
```

## Configuração

### Parâmetros por chamada

O modelo, o locale e o formato de saída são definidos por chamada, via
`AnalyzeDocumentParams` e `AnalyzeBatchParams`. Quando omitidos, usam os padrões
`ModelLayout` e `ContentFormatMarkdown`; o locale vazio deixa a Azure detectar o
idioma automaticamente.

Formatos de saída disponíveis (`ContentFormat`):

- `ContentFormatText` — texto puro (`text`).
- `ContentFormatMarkdown` — Markdown (`markdown`).

Modelos prebuilt (`Model`) incluem `ModelRead`, `ModelLayout`, `ModelInvoice`,
`ModelReceipt`, `ModelIDDocument`, `ModelBusinessCard`, `ModelContract` e
`ModelTaxUSW2`. Qualquer ID de modelo custom pode ser usado convertendo a string
para `docintel.Model`.

### Client

O client controla a versão da API e o transporte HTTP:

```go
client := docintel.NewClient(endpoint, key,
    docintel.WithAPIVersion("2024-11-30"),
    docintel.WithHTTPClient(customClient), // ex: autenticação via Azure AD
)
```

Por padrão o client autentica com a API key (header
`Ocp-Apim-Subscription-Key`). Para usar autenticação via Azure AD, injete um
`*http.Client` com um `http.RoundTripper` próprio usando `WithHTTPClient`.

### Polling

`PollResult` e `PollBatchResult` consultam a operação em intervalos regulares até
um status terminal ou até o tempo limite. O intervalo e o tempo limite são
configurados por chamada, via `WithPollInterval` e `WithPollTimeout`:

```go
op, err := client.PollResult(ctx, location,
    docintel.WithPollInterval(5*time.Second),
    docintel.WithPollTimeout(10*time.Minute),
)
```

Os padrões são intervalo de 2s para ambos, tempo limite de 5min em `PollResult` e
30min em `PollBatchResult` (que processa múltiplos documentos). O deadline do
`context.Context` também é respeitado — o que ocorrer primeiro encerra o polling,
retornando `poller.ErrTimeout` no caso do tempo limite. O pacote
`github.com/automatiza-mg/docintel/poller` é público e pode ser reutilizado de
forma independente.

## Erros

- `ErrInvalidAnalyzeRequest`: parâmetros de análise de documento inválidos.
- `ErrInvalidBatchRequest`: parâmetros de análise em lote inválidos.
- `ErrOperationNotFound`: a operação consultada não existe mais.
- `*AnalyzeError`: falha no processamento de um documento (carrega o `Status`).
- `*StatusError`: resposta HTTP com status inesperado; `Retryable()` indica se
  a requisição pode ser repetida (429 e 5xx).

## Licença

[MIT](LICENSE)
