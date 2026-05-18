package prompts

import (
	"strings"
	"testing"
)

func TestNewPrompter(t *testing.T) {
	p := NewPrompter()
	if p == nil {
		t.Error("NewPrompter() should not return nil")
	}
}

func TestMarketAnalyst(t *testing.T) {
	p := NewPrompter()
	prompt := p.MarketAnalyst()

	if prompt == "" {
		t.Error("MarketAnalyst() should not return empty string")
	}
	if !strings.Contains(prompt, "market analyst") {
		t.Error("prompt should mention market analyst")
	}
	if !strings.Contains(prompt, "probability") {
		t.Error("prompt should mention probability")
	}
}

func TestSentimentAnalyzer(t *testing.T) {
	p := NewPrompter()
	prompt := p.SentimentAnalyzer("Will BTC reach $100k?", "yes")

	if prompt == "" {
		t.Error("SentimentAnalyzer() should not return empty string")
	}
	if !strings.Contains(prompt, "Will BTC reach $100k?") {
		t.Error("prompt should contain the question")
	}
	if !strings.Contains(prompt, "yes") {
		t.Error("prompt should contain the outcome")
	}
	if !strings.Contains(prompt, "sentiment") {
		t.Error("prompt should mention sentiment")
	}
}

func TestPolymarketAnalystAPI(t *testing.T) {
	p := NewPrompter()
	prompt := p.PolymarketAnalystAPI()

	if prompt == "" {
		t.Error("PolymarketAnalystAPI() should not return empty string")
	}
	if !strings.Contains(prompt, "Polymarket") {
		t.Error("prompt should mention Polymarket")
	}
	if !strings.Contains(prompt, "prediction market") {
		t.Error("prompt should mention prediction market")
	}
}

func TestFilterEventsPrompt(t *testing.T) {
	p := NewPrompter()
	prompt := p.FilterEvents()

	if prompt == "" {
		t.Error("FilterEvents() should not return empty string")
	}
	if !strings.Contains(prompt, "Filter") {
		t.Error("prompt should mention Filter")
	}
	if !strings.Contains(prompt, "events") {
		t.Error("prompt should mention events")
	}
}

func TestFilterMarketsPrompt(t *testing.T) {
	p := NewPrompter()
	prompt := p.FilterMarkets()

	if prompt == "" {
		t.Error("FilterMarkets() should not return empty string")
	}
	if !strings.Contains(prompt, "Filter") {
		t.Error("prompt should mention Filter")
	}
	if !strings.Contains(prompt, "markets") {
		t.Error("prompt should mention markets")
	}
}

func TestSuperforecaster(t *testing.T) {
	p := NewPrompter()
	prompt := p.Superforecaster("Will BTC reach $100k?", "Bitcoin price prediction", "yes")

	if prompt == "" {
		t.Error("Superforecaster() should not return empty string")
	}
	if !strings.Contains(prompt, "Superforecaster") {
		t.Error("prompt should mention Superforecaster")
	}
	if !strings.Contains(prompt, "Will BTC reach $100k?") {
		t.Error("prompt should contain the question")
	}
	if !strings.Contains(prompt, "Bitcoin price prediction") {
		t.Error("prompt should contain the description")
	}
	if !strings.Contains(prompt, "Base Rates") {
		t.Error("prompt should mention Base Rates")
	}
	if !strings.Contains(prompt, "PROBABILITY") {
		t.Error("prompt should mention PROBABILITY in response format")
	}
}

func TestOneBestTrade(t *testing.T) {
	p := NewPrompter()
	prompt := p.OneBestTrade("I predict 65% chance of yes", []string{"Yes", "No"}, `{"Yes": 0.60, "No": 0.40}`)

	if prompt == "" {
		t.Error("OneBestTrade() should not return empty string")
	}
	if !strings.Contains(prompt, "I predict 65% chance of yes") {
		t.Error("prompt should contain the prediction")
	}
	if !strings.Contains(prompt, "price:") {
		t.Error("prompt should show response format with price")
	}
	if !strings.Contains(prompt, "side:") {
		t.Error("prompt should show response format with side")
	}
}

func TestSimpleAITrader(t *testing.T) {
	p := NewPrompter()
	prompt := p.SimpleAITrader("BTC price prediction market", "Recent news suggests bullish momentum")

	if prompt == "" {
		t.Error("SimpleAITrader() should not return empty string")
	}
	if !strings.Contains(prompt, "trader") {
		t.Error("prompt should mention trader")
	}
	if !strings.Contains(prompt, "BTC price prediction market") {
		t.Error("prompt should contain market description")
	}
	if !strings.Contains(prompt, "Recent news suggests bullish momentum") {
		t.Error("prompt should contain relevant info")
	}
}

func TestPolymarketAssistant(t *testing.T) {
	p := NewPrompter()
	prompt := p.PolymarketAssistant(`{"markets": []}`, `{"events": []}`)

	if prompt == "" {
		t.Error("PolymarketAssistant() should not return empty string")
	}
	if !strings.Contains(prompt, "Polymarket") {
		t.Error("prompt should mention Polymarket")
	}
	if !strings.Contains(prompt, `{"markets": []}`) {
		t.Error("prompt should contain market data")
	}
}

func TestMultiQuery(t *testing.T) {
	p := NewPrompter()
	prompt := p.MultiQuery("What is the price of Bitcoin?")

	if prompt == "" {
		t.Error("MultiQuery() should not return empty string")
	}
	if !strings.Contains(prompt, "five different versions") {
		t.Error("prompt should mention generating five versions")
	}
	if !strings.Contains(prompt, "What is the price of Bitcoin?") {
		t.Error("prompt should contain original question")
	}
}

func TestCreateNewMarket(t *testing.T) {
	p := NewPrompter()
	prompt := p.CreateNewMarket(`[{"question": "Test market"}]`)

	if prompt == "" {
		t.Error("CreateNewMarket() should not return empty string")
	}
	if !strings.Contains(prompt, "Invent an information market") {
		t.Error("prompt should mention inventing a market")
	}
	if !strings.Contains(prompt, "6 months") {
		t.Error("prompt should mention 6 months")
	}
	if !strings.Contains(prompt, "Question:") {
		t.Error("prompt should show response format")
	}
}

func TestFormatPriceFromTrade(t *testing.T) {
	p := NewPrompter()
	prompt := p.FormatPriceFromTrade()

	if prompt == "" {
		t.Error("FormatPriceFromTrade() should not return empty string")
	}
	if !strings.Contains(prompt, "price:") {
		t.Error("prompt should show example with price")
	}
	if !strings.Contains(prompt, "0.5") {
		t.Error("prompt should show expected output")
	}
}

func TestFormatSizeFromTrade(t *testing.T) {
	p := NewPrompter()
	prompt := p.FormatSizeFromTrade()

	if prompt == "" {
		t.Error("FormatSizeFromTrade() should not return empty string")
	}
	if !strings.Contains(prompt, "size:") {
		t.Error("prompt should show example with size")
	}
	if !strings.Contains(prompt, "0.1") {
		t.Error("prompt should show expected output")
	}
}

func TestEdgeCalculation(t *testing.T) {
	p := NewPrompter()
	prompt := p.EdgeCalculation(0.65, 0.60)

	if prompt == "" {
		t.Error("EdgeCalculation() should not return empty string")
	}
	if !strings.Contains(prompt, "Fair Value") {
		t.Error("prompt should mention Fair Value")
	}
	if !strings.Contains(prompt, "Market Price") {
		t.Error("prompt should mention Market Price")
	}
	if !strings.Contains(prompt, "Edge") {
		t.Error("prompt should mention Edge")
	}
	if !strings.Contains(prompt, "Kelly") {
		t.Error("prompt should mention Kelly criterion")
	}
}

func TestRiskAssessment(t *testing.T) {
	p := NewPrompter()
	prompt := p.RiskAssessment("BTC $100k market", "BUY 100 shares at 0.65", 15.5)

	if prompt == "" {
		t.Error("RiskAssessment() should not return empty string")
	}
	if !strings.Contains(prompt, "BTC $100k market") {
		t.Error("prompt should contain market")
	}
	if !strings.Contains(prompt, "BUY 100 shares at 0.65") {
		t.Error("prompt should contain position")
	}
	if !strings.Contains(prompt, "15.50%") {
		t.Error("prompt should contain exposure percentage")
	}
	if !strings.Contains(prompt, "Resolution risk") {
		t.Error("prompt should mention resolution risk")
	}
	if !strings.Contains(prompt, "Liquidity risk") {
		t.Error("prompt should mention liquidity risk")
	}
}

func TestParseTradeResponse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPrice float64
		wantSize  float64
		wantSide  string
		wantErr   bool
	}{
		{
			name: "valid response",
			input: `RESPONSE
    price:0.65,
    size:0.1,
    side:BUY,`,
			wantPrice: 0.65,
			wantSize:  0.1,
			wantSide:  "BUY",
			wantErr:   false,
		},
		{
			name: "valid sell response",
			input: `    price:0.35,
    size:0.05,
    side:SELL,`,
			wantPrice: 0.35,
			wantSize:  0.05,
			wantSide:  "SELL",
			wantErr:   false,
		},
		{
			name: "no trailing comma on side",
			input: `price:0.50,
size:0.2,
side:BUY`,
			wantPrice: 0.50,
			wantSize:  0.2,
			wantSide:  "BUY",
			wantErr:   false,
		},
		{
			name:    "missing side",
			input:   `price:0.5, size:0.1`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "garbage input",
			input:   "This is not a valid trade response",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, size, side, err := ParseTradeResponse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if price != tt.wantPrice {
				t.Errorf("price = %f, want %f", price, tt.wantPrice)
			}
			if size != tt.wantSize {
				t.Errorf("size = %f, want %f", size, tt.wantSize)
			}
			if side != tt.wantSide {
				t.Errorf("side = %q, want %q", side, tt.wantSide)
			}
		})
	}
}
