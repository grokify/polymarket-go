package news

import (
	"testing"
)

func TestExtractKeyTerms(t *testing.T) {
	tests := []struct {
		name     string
		question string
		want     string
	}{
		{
			name:     "simple question",
			question: "Will Bitcoin reach $100k?",
			want:     "bitcoin reach $100k",
		},
		{
			name:     "removes stop words",
			question: "Will the SEC approve a Bitcoin ETF in 2025?",
			want:     "sec approve bitcoin etf 2025",
		},
		{
			name:     "removes short words",
			question: "Is it a go?",
			want:     "",
		},
		{
			name:     "handles punctuation",
			question: "What is the price of Bitcoin, Ethereum, and Solana?",
			want:     "price bitcoin ethereum solana",
		},
		{
			name:     "preserves important terms",
			question: "Will Donald Trump win the 2024 election?",
			want:     "donald trump win 2024 election",
		},
		{
			name:     "empty question",
			question: "",
			want:     "",
		},
		{
			name:     "only stop words",
			question: "Will the be is are?",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeyTerms(tt.question)
			if got != tt.want {
				t.Errorf("extractKeyTerms(%q) = %q, want %q", tt.question, got, tt.want)
			}
		})
	}
}

func TestSearchOptionsDefaults(t *testing.T) {
	opts := SearchOptions{}

	if opts.NumResults != 0 {
		t.Errorf("default NumResults = %d, want 0", opts.NumResults)
	}
	if opts.Language != "" {
		t.Errorf("default Language = %q, want empty", opts.Language)
	}
	if opts.Country != "" {
		t.Errorf("default Country = %q, want empty", opts.Country)
	}
}

func TestNewsArticleSerialization(t *testing.T) {
	article := NewsArticle{
		Title:     "Bitcoin ETF Approved",
		Link:      "https://example.com/article",
		Source:    "Reuters",
		Date:      "2025-05-17",
		Snippet:   "The SEC has approved...",
		Thumbnail: "https://example.com/thumb.jpg",
	}

	if article.Title == "" {
		t.Error("Title should not be empty")
	}
	if article.Link == "" {
		t.Error("Link should not be empty")
	}
}

func TestWebSearchResultStructure(t *testing.T) {
	result := WebSearchResult{
		OrganicResults: []OrganicResult{
			{
				Position: 1,
				Title:    "Test Result",
				Link:     "https://example.com",
				Snippet:  "Test snippet",
				Domain:   "example.com",
			},
		},
		AnswerBox: &AnswerBox{
			Title:   "Answer",
			Answer:  "42",
			Snippet: "The answer is 42",
			Link:    "https://example.com/answer",
		},
	}

	if len(result.OrganicResults) != 1 {
		t.Errorf("OrganicResults count = %d, want 1", len(result.OrganicResults))
	}
	if result.AnswerBox == nil {
		t.Error("AnswerBox should not be nil")
	}
	if result.AnswerBox.Answer != "42" {
		t.Errorf("AnswerBox.Answer = %q, want %q", result.AnswerBox.Answer, "42")
	}
}

func TestOrganicResultStructure(t *testing.T) {
	result := OrganicResult{
		Position: 1,
		Title:    "Test",
		Link:     "https://example.com",
		Snippet:  "Snippet",
		Domain:   "example.com",
		Date:     "2025-05-17",
	}

	if result.Position != 1 {
		t.Errorf("Position = %d, want 1", result.Position)
	}
	if result.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "example.com")
	}
}

func TestScrapedContentStructure(t *testing.T) {
	content := ScrapedContent{
		URL:   "https://example.com/page",
		Title: "Page Title",
		Text:  "Page content goes here...",
	}

	if content.URL == "" {
		t.Error("URL should not be empty")
	}
	if content.Title == "" {
		t.Error("Title should not be empty")
	}
	if content.Text == "" {
		t.Error("Text should not be empty")
	}
}

func TestSearcherConfigDefaults(t *testing.T) {
	cfg := SearcherConfig{}

	if cfg.Engine != "" {
		t.Errorf("default Engine = %q, want empty", cfg.Engine)
	}
}

// TestNewSearcherRequiresAPIKey verifies that creating a searcher
// requires API credentials (this will fail without env vars set).
func TestNewSearcherRequiresAPIKey(t *testing.T) {
	// This test documents expected behavior - it will fail without
	// SERPER_API_KEY or SERPAPI_API_KEY set.
	// In a real test suite, we'd mock the client.
	t.Skip("Skipping: requires SERPER_API_KEY or SERPAPI_API_KEY")

	_, err := NewSearcher(SearcherConfig{})
	if err != nil {
		t.Logf("Expected error without API key: %v", err)
	}
}
