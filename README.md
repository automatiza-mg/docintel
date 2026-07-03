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
deve ser consultada com `GetAnalyzeResult` até atingir um status terminal.

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

    // Consulte a operação até que ela seja concluída (polling a cargo do chamador).
    for {
        op, err := client.GetAnalyzeResult(ctx, location)
        if err != nil {
            panic(err)
        }

        switch op.Status {
        case docintel.StatusSucceeded:
            fmt.Println(op.AnalyzeResult.Content)
            return
        case docintel.StatusFailed, docintel.StatusCanceled, docintel.StatusSkipped:
            panic(&docintel.AnalyzeError{Status: op.Status, Err: op.Error})
        }
        // StatusRunning / StatusNotStarted: aguarde e tente novamente.
    }
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

Consulte o resultado com `GetBatchResult`. Os resultados de cada documento são
gravados no container de destino, e não retornados na resposta.

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

## Erros

- `ErrInvalidAnalyzeRequest`: parâmetros de análise de documento inválidos.
- `ErrInvalidBatchRequest`: parâmetros de análise em lote inválidos.
- `ErrOperationNotFound`: a operação consultada não existe mais.
- `*AnalyzeError`: falha no processamento de um documento (carrega o `Status`).
- `*StatusError`: resposta HTTP com status inesperado; `Retryable()` indica se
  a requisição pode ser repetida (429 e 5xx).

## Licença

[MIT](LICENSE)
