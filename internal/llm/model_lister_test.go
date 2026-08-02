package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeTransport implements http.RoundTripper for test isolation — no real API calls.
type fakeTransport struct {
	fn func(req *http.Request) (*http.Response, error)
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.fn(req)
}

func fakeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

const validModelsJSON = `{"data":[{"id":"model-a"},{"id":"model-b"},{"id":"model-c"}]}`

// --- openaiCompatLister tests ---

func TestOpenAICompatLister_Success(t *testing.T) {
	lister := &openaiCompatLister{
		baseURL: "https://api.example.com/v1",
		apiKey:  "test-key",
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.example.com/v1/models" {
				t.Errorf("unexpected URL: %s", req.URL)
			}
			if req.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("unexpected auth header: %s", req.Header.Get("Authorization"))
			}
			return fakeResponse(200, validModelsJSON), nil
		}}},
	}

	models, err := lister.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}
	if models[0] != "model-a" || models[1] != "model-b" || models[2] != "model-c" {
		t.Errorf("unexpected model IDs: %v", models)
	}
}

func TestOpenAICompatLister_Auth401(t *testing.T) {
	lister := &openaiCompatLister{
		baseURL: "https://api.example.com/v1",
		apiKey:  "bad-key",
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			return fakeResponse(401, `{"error":"unauthorized"}`), nil
		}}},
	}

	models, err := lister.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if models != nil {
		t.Errorf("expected nil models on error, got: %v", models)
	}
}

func TestOpenAICompatLister_NetworkError(t *testing.T) {
	lister := &openaiCompatLister{
		baseURL: "https://api.example.com/v1",
		apiKey:  "key",
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}}},
	}

	models, err := lister.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
	if models != nil {
		t.Errorf("expected nil models on error, got: %v", models)
	}
}

func TestLocalLister_ConnectionRefused(t *testing.T) {
	lister := &openaiCompatLister{
		baseURL: "http://localhost:11434/v1",
		apiKey:  "",
		isLocal: true,
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF // simulate connection failure
		}}},
	}

	_, err := lister.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure, got nil")
	}
	expectedErr := "no local model runtime found at http://localhost:11434/v1 — is Ollama running?"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %q, got: %v", expectedErr, err)
	}
}

func TestOpenAICompatLister_EmptyData(t *testing.T) {
	lister := &openaiCompatLister{
		baseURL: "https://api.example.com/v1",
		apiKey:  "key",
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			return fakeResponse(200, `{"data":[]}`), nil
		}}},
	}

	models, err := lister.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("expected empty slice, got: %v", models)
	}
}

// --- anthropicLister tests ---

func TestAnthropicLister_Success(t *testing.T) {
	lister := &anthropicLister{
		apiKey: "anthro-key",
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.anthropic.com/v1/models" {
				t.Errorf("unexpected URL: %s", req.URL)
			}
			if req.Header.Get("x-api-key") != "anthro-key" {
				t.Errorf("missing or wrong x-api-key: %q", req.Header.Get("x-api-key"))
			}
			if req.Header.Get("anthropic-version") != "2023-06-01" {
				t.Errorf("missing or wrong anthropic-version: %q", req.Header.Get("anthropic-version"))
			}
			return fakeResponse(200, validModelsJSON), nil
		}}},
	}

	models, err := lister.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}
}

func TestAnthropicLister_Failure(t *testing.T) {
	lister := &anthropicLister{
		apiKey: "key",
		client: &http.Client{Transport: &fakeTransport{fn: func(req *http.Request) (*http.Response, error) {
			return fakeResponse(403, `{"error":"forbidden"}`), nil
		}}},
	}

	models, err := lister.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if models != nil {
		t.Errorf("expected nil models on error, got: %v", models)
	}
}

// --- NewModelLister tests ---

func TestNewModelLister_KnownProviders(t *testing.T) {
	known := []string{"anthropic", "openai", "groq", "grok", "mistral", "gemini"}
	for _, provider := range known {
		lister, ok := NewModelLister(provider, "key", "")
		if !ok {
			t.Errorf("expected NewModelLister(%q) to return ok=true", provider)
		}
		if lister == nil {
			t.Errorf("expected non-nil lister for provider %q", provider)
		}
	}
}

func TestNewModelLister_LocalReturnsLister(t *testing.T) {
	lister, ok := NewModelLister("local", "", "http://localhost:11434/v1")
	if !ok {
		t.Error("expected NewModelLister(\"local\") to return ok=true")
	}
	if lister == nil {
		t.Error("expected non-nil lister for local provider")
	}
}
