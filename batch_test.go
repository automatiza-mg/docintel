package docintel

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAnalyzeBatch_Accepted(t *testing.T) {
	t.Parallel()

	const wantLocation = "https://example.com/documentintelligence/documentModels/prebuilt-layout/analyzeBatchResults/123?api-version=2024-11-30"

	var gotBody AnalyzeBatchParams
	var gotContentType string
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Operation-Location", wantLocation)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-key")

	params := AnalyzeBatchParams{
		AzureBlobSource: &AzureBlobSource{
			ContainerURL: "https://storage.blob.core.windows.net/in?sas",
			Prefix:       "inputDocs/",
		},
		ResultContainerURL: "https://storage.blob.core.windows.net/out?sas",
		ResultPrefix:       "batchResults/",
		OverwriteExisting:  true,
		Model:              ModelRead,
		Locale:             "en-US",
		OutputFormat:       ContentFormatText,
	}

	location, err := client.AnalyzeBatch(t.Context(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if location != wantLocation {
		t.Fatalf("location = %q, want %q", location, wantLocation)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotContentType)
	}

	// Model, Locale e OutputFormat vão na query, não no corpo JSON.
	if want := "/documentintelligence/documentModels/prebuilt-read:analyzeBatch"; gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}
	if got := gotQuery.Get("outputContentFormat"); got != "text" {
		t.Fatalf("outputContentFormat = %q, want text", got)
	}
	if got := gotQuery.Get("locale"); got != "en-US" {
		t.Fatalf("locale = %q, want en-US", got)
	}

	// O corpo JSON contém apenas a fonte e o destino; os campos de query são omitidos.
	want := AnalyzeBatchParams{
		AzureBlobSource: &AzureBlobSource{
			ContainerURL: "https://storage.blob.core.windows.net/in?sas",
			Prefix:       "inputDocs/",
		},
		ResultContainerURL: "https://storage.blob.core.windows.net/out?sas",
		ResultPrefix:       "batchResults/",
		OverwriteExisting:  true,
	}
	if diff := cmp.Diff(want, gotBody); diff != "" {
		t.Fatalf("request body mismatch (-want +got):\n%s", diff)
	}
}

func TestAnalyzeBatch_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params AnalyzeBatchParams
	}{
		{
			name:   "no source",
			params: AnalyzeBatchParams{ResultContainerURL: "https://out?sas"},
		},
		{
			name: "both sources",
			params: AnalyzeBatchParams{
				AzureBlobSource:         &AzureBlobSource{ContainerURL: "https://in?sas"},
				AzureBlobFileListSource: &AzureBlobFileListSource{ContainerURL: "https://in?sas", FileList: "files.jsonl"},
				ResultContainerURL:      "https://out?sas",
			},
		},
		{
			name: "blob source missing container url",
			params: AnalyzeBatchParams{
				AzureBlobSource:    &AzureBlobSource{},
				ResultContainerURL: "https://out?sas",
			},
		},
		{
			name: "file list missing container url",
			params: AnalyzeBatchParams{
				AzureBlobFileListSource: &AzureBlobFileListSource{FileList: "files.jsonl"},
				ResultContainerURL:      "https://out?sas",
			},
		},
		{
			name: "file list missing file list",
			params: AnalyzeBatchParams{
				AzureBlobFileListSource: &AzureBlobFileListSource{ContainerURL: "https://in?sas"},
				ResultContainerURL:      "https://out?sas",
			},
		},
		{
			name: "missing result container url",
			params: AnalyzeBatchParams{
				AzureBlobSource: &AzureBlobSource{ContainerURL: "https://in?sas"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called for invalid params")
			}))
			defer srv.Close()

			client := NewClient(srv.URL, "secret-key")

			_, err := client.AnalyzeBatch(t.Context(), tt.params)
			if !errors.Is(err, ErrInvalidBatchRequest) {
				t.Fatalf("error = %v, want ErrInvalidBatchRequest", err)
			}
		})
	}
}

func TestAnalyzeBatch_UnexpectedStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "bad request")
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-key")

	params := AnalyzeBatchParams{
		AzureBlobSource:    &AzureBlobSource{ContainerURL: "https://in?sas"},
		ResultContainerURL: "https://out?sas",
	}

	_, err := client.AnalyzeBatch(t.Context(), params)
	statusErr, ok := errors.AsType[*StatusError](err)
	if !ok {
		t.Fatalf("error = %v, want *StatusError", err)
	}
	if statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", statusErr.StatusCode, http.StatusBadRequest)
	}
}

func TestGetBatchResult_Completed(t *testing.T) {
	t.Parallel()

	const payload = `{
		"resultId": "result-123",
		"status": "completed",
		"percentCompleted": 100,
		"createdDateTime": "2025-01-01T00:00:00Z",
		"lastUpdatedDateTime": "2025-01-01T00:01:00Z",
		"result": {
			"succeededCount": 1,
			"failedCount": 1,
			"skippedCount": 1,
			"details": [
				{
					"sourceUrl": "https://in/doc1.pdf",
					"resultUrl": "https://out/doc1.pdf.ocr.json",
					"status": "succeeded"
				},
				{
					"sourceUrl": "https://in/doc2.pdf",
					"status": "failed",
					"error": {"code": "InvalidArgument", "message": "Invalid argument."}
				},
				{
					"sourceUrl": "https://in/doc3.pdf",
					"status": "skipped",
					"error": {"code": "OutputExists", "message": "already exists"}
				}
			]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, payload)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-key")

	op, err := client.GetBatchResult(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := &BatchResult{
		SucceededCount: 1,
		FailedCount:    1,
		SkippedCount:   1,
		Details: []BatchResultDetail{
			{SourceURL: "https://in/doc1.pdf", ResultURL: "https://out/doc1.pdf.ocr.json", Status: StatusSucceeded},
			{SourceURL: "https://in/doc2.pdf", Status: StatusFailed, Error: &AzureError{Code: "InvalidArgument", Message: "Invalid argument."}},
			{SourceURL: "https://in/doc3.pdf", Status: StatusSkipped, Error: &AzureError{Code: "OutputExists", Message: "already exists"}},
		},
	}

	if op.Status != StatusCompleted {
		t.Fatalf("Status = %q, want %q", op.Status, StatusCompleted)
	}
	if op.ResultID != "result-123" {
		t.Fatalf("ResultID = %q, want %q", op.ResultID, "result-123")
	}
	if diff := cmp.Diff(want, &op.Result); diff != "" {
		t.Fatalf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestGetBatchResult_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-key")

	_, err := client.GetBatchResult(t.Context(), srv.URL)
	if !errors.Is(err, ErrOperationNotFound) {
		t.Fatalf("error = %v, want ErrOperationNotFound", err)
	}
}

func TestGetBatchResult_UnexpectedStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-key")

	_, err := client.GetBatchResult(t.Context(), srv.URL)
	statusErr, ok := errors.AsType[*StatusError](err)
	if !ok {
		t.Fatalf("error = %v, want *StatusError", err)
	}
	if statusErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("StatusCode = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
	}
}
