package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	perrors "github.com/grokify/polymarket-go/internal/errors"
)

// writeJSON is a helper to write JSON responses in tests with proper error handling.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("failed to encode JSON response: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		client := NewClient()
		if client.clobURL != DefaultCLOBURL {
			t.Errorf("clobURL = %q, want %q", client.clobURL, DefaultCLOBURL)
		}
		if client.gammaURL != DefaultGammaURL {
			t.Errorf("gammaURL = %q, want %q", client.gammaURL, DefaultGammaURL)
		}
		if client.httpClient == nil {
			t.Error("httpClient is nil")
		}
	})

	t.Run("with custom CLOB URL", func(t *testing.T) {
		customURL := "https://custom-clob.example.com"
		client := NewClient(WithCLOBURL(customURL))
		if client.clobURL != customURL {
			t.Errorf("clobURL = %q, want %q", client.clobURL, customURL)
		}
	})

	t.Run("with custom Gamma URL", func(t *testing.T) {
		customURL := "https://custom-gamma.example.com"
		client := NewClient(WithGammaURL(customURL))
		if client.gammaURL != customURL {
			t.Errorf("gammaURL = %q, want %q", client.gammaURL, customURL)
		}
	})

	t.Run("with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 60 * time.Second}
		client := NewClient(WithHTTPClient(customClient))
		if client.httpClient != customClient {
			t.Error("httpClient not set correctly")
		}
	})
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		maxLen int
		want   string
	}{
		{
			name:   "short body unchanged",
			body:   "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length unchanged",
			body:   "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long body truncated",
			body:   "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "empty body",
			body:   "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateBody(tt.body, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateBody(%q, %d) = %q, want %q", tt.body, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestDoGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		writeJSON(t, w, Market{
			ID:       "test-id",
			Question: "Test question?",
		})
	}))
	defer server.Close()

	client := NewClient(WithGammaURL(server.URL))
	var market Market
	err := client.doGet(context.Background(), server.URL, &market)
	if err != nil {
		t.Fatalf("doGet failed: %v", err)
	}
	if market.ID != "test-id" {
		t.Errorf("market.ID = %q, want %q", market.ID, "test-id")
	}
}

func TestDoGet_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"error": "not found"}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(WithGammaURL(server.URL))
	var market Market
	err := client.doGet(context.Background(), server.URL, &market)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *perrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if apiErr.Service != "Polymarket" {
		t.Errorf("Service = %q, want %q", apiErr.Service, "Polymarket")
	}
}

func TestDoGet_NetworkError(t *testing.T) {
	client := NewClient()
	var market Market
	err := client.doGet(context.Background(), "http://invalid-host-that-does-not-exist.local/test", &market)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var netErr *perrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected NetworkError, got %T: %v", err, err)
	}
}

func TestDoGet_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`invalid json`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient()
	var market Market
	err := client.doGet(context.Background(), server.URL, &market)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should be a JSON decode error
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestGetMarket_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets/test-condition-id" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, Market{
			ID:          "test-id",
			ConditionID: "test-condition-id",
			Question:    "Test question?",
		})
	}))
	defer server.Close()

	client := NewClient(WithGammaURL(server.URL))
	market, err := client.GetMarket(context.Background(), "test-condition-id")
	if err != nil {
		t.Fatalf("GetMarket failed: %v", err)
	}
	if market.ConditionID != "test-condition-id" {
		t.Errorf("ConditionID = %q, want %q", market.ConditionID, "test-condition-id")
	}
}

func TestGetOrderBook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token_id") != "test-token" {
			t.Errorf("unexpected token_id: %s", r.URL.Query().Get("token_id"))
		}
		writeJSON(t, w, OrderBook{
			Market: "test-market",
			Asset:  "test-token",
			Bids:   []OrderBookLevel{{Price: "0.50", Size: "100"}},
			Asks:   []OrderBookLevel{{Price: "0.55", Size: "50"}},
		})
	}))
	defer server.Close()

	client := NewClient(WithCLOBURL(server.URL))
	book, err := client.GetOrderBook(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetOrderBook failed: %v", err)
	}
	if len(book.Bids) != 1 {
		t.Errorf("len(Bids) = %d, want 1", len(book.Bids))
	}
	if len(book.Asks) != 1 {
		t.Errorf("len(Asks) = %d, want 1", len(book.Asks))
	}
}

func TestGetPrice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]string{
			"token_id": "test-token",
			"price":    "0.55",
		})
	}))
	defer server.Close()

	client := NewClient(WithCLOBURL(server.URL))
	price, err := client.GetPrice(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetPrice failed: %v", err)
	}
	if price.Price != 0.55 {
		t.Errorf("Price = %f, want 0.55", price.Price)
	}
}

func TestGetMidPrice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]string{"mid": "0.52"})
	}))
	defer server.Close()

	client := NewClient(WithCLOBURL(server.URL))
	mid, err := client.GetMidPrice(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetMidPrice failed: %v", err)
	}
	if mid.Mid != 0.52 {
		t.Errorf("Mid = %f, want 0.52", mid.Mid)
	}
}

func TestGetSpread_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]string{"spread": "0.05"})
	}))
	defer server.Close()

	client := NewClient(WithCLOBURL(server.URL))
	spread, err := client.GetSpread(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetSpread failed: %v", err)
	}
	if spread.Spread != 0.05 {
		t.Errorf("Spread = %f, want 0.05", spread.Spread)
	}
}

func TestGetMarkets_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		q := r.URL.Query()
		if q.Get("active") != "true" {
			t.Errorf("active = %q, want 'true'", q.Get("active"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want '10'", q.Get("limit"))
		}
		if q.Get("order") != "liquidity" {
			t.Errorf("order = %q, want 'liquidity'", q.Get("order"))
		}
		if q.Get("liquidity_min") != "1000" {
			t.Errorf("liquidity_min = %q, want '1000'", q.Get("liquidity_min"))
		}
		if q.Get("text_query") != "bitcoin" {
			t.Errorf("text_query = %q, want 'bitcoin'", q.Get("text_query"))
		}

		writeJSON(t, w, []Market{
			{ID: "1", Question: "Test 1"},
			{ID: "2", Question: "Test 2"},
		})
	}))
	defer server.Close()

	client := NewClient(WithGammaURL(server.URL))
	active := true
	markets, err := client.GetMarkets(context.Background(), GetMarketsParams{
		Active:    &active,
		Limit:     10,
		Order:     "liquidity",
		LiqMin:    1000,
		TextQuery: "bitcoin",
	})
	if err != nil {
		t.Fatalf("GetMarkets failed: %v", err)
	}
	if len(markets) != 2 {
		t.Errorf("len(markets) = %d, want 2", len(markets))
	}
}
