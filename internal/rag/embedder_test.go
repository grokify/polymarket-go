package rag

import (
	"testing"
)

func TestNewOpenAIEmbedderMissingKey(t *testing.T) {
	// Don't set OPENAI_API_KEY env var
	_, err := NewOpenAIEmbedder(OpenAIEmbedderConfig{})
	// This will fail unless OPENAI_API_KEY is set in environment
	// We can't easily test the success case without mocking
	if err == nil {
		// API key is set in env, skip this test
		t.Skip("OPENAI_API_KEY is set in environment")
	}
	if err != nil && !containsString(err.Error(), "API key") {
		t.Errorf("error should mention API key: %v", err)
	}
}

func TestNewOpenAIEmbedderWithAPIKey(t *testing.T) {
	embedder, err := NewOpenAIEmbedder(OpenAIEmbedderConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if embedder == nil {
		t.Fatal("embedder should not be nil")
	}
	if embedder.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want test-key", embedder.apiKey)
	}
}

func TestNewOpenAIEmbedderDefaultModel(t *testing.T) {
	embedder, err := NewOpenAIEmbedder(OpenAIEmbedderConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if embedder.model != "text-embedding-3-small" {
		t.Errorf("model = %q, want text-embedding-3-small", embedder.model)
	}
}

func TestNewOpenAIEmbedderCustomModel(t *testing.T) {
	embedder, err := NewOpenAIEmbedder(OpenAIEmbedderConfig{
		APIKey: "test-key",
		Model:  "text-embedding-3-large",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if embedder.model != "text-embedding-3-large" {
		t.Errorf("model = %q, want text-embedding-3-large", embedder.model)
	}
}

func TestOpenAIEmbedderModel(t *testing.T) {
	embedder, _ := NewOpenAIEmbedder(OpenAIEmbedderConfig{
		APIKey: "test-key",
		Model:  "custom-model",
	})

	if embedder.Model() != "custom-model" {
		t.Errorf("Model() = %q, want custom-model", embedder.Model())
	}
}

func TestOpenAIEmbedderDimensions(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"text-embedding-3-small", 1536},
		{"text-embedding-3-large", 3072},
		{"text-embedding-ada-002", 1536},
		{"unknown-model", 1536}, // defaults to 1536
	}

	for _, tt := range tests {
		embedder, _ := NewOpenAIEmbedder(OpenAIEmbedderConfig{
			APIKey: "test-key",
			Model:  tt.model,
		})

		if got := embedder.Dimensions(); got != tt.want {
			t.Errorf("Dimensions() for %q = %d, want %d", tt.model, got, tt.want)
		}
	}
}

func TestOpenAIEmbedderConfigStructure(t *testing.T) {
	cfg := OpenAIEmbedderConfig{
		APIKey: "key",
		Model:  "model",
	}

	if cfg.APIKey != "key" {
		t.Errorf("APIKey = %q, want key", cfg.APIKey)
	}
	if cfg.Model != "model" {
		t.Errorf("Model = %q, want model", cfg.Model)
	}
}

// Note: Testing Embed and EmbedBatch would require mocking HTTP responses
// or integration testing against OpenAI's API. These tests cover the
// configuration and dimension logic which doesn't require network calls.
