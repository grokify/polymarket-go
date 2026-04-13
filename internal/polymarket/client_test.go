package polymarket

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestGetMarkets(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	active := true
	markets, err := client.GetMarkets(ctx, GetMarketsParams{
		Active: &active,
		Limit:  5,
		Order:  "liquidity",
	})
	if err != nil {
		t.Fatalf("GetMarkets failed: %v", err)
	}

	if len(markets) == 0 {
		t.Fatal("Expected at least one market")
	}

	t.Logf("Fetched %d markets", len(markets))
	for i, m := range markets {
		t.Logf("  [%d] %s (liquidity: %.2f, bid: %.2f, ask: %.2f)",
			i+1, m.Question, m.LiquidityNum, m.BestBid, m.BestAsk)
	}
}

func TestGetOrderBook(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get markets with high liquidity which are more likely to have order books
	active := true
	markets, err := client.GetMarkets(ctx, GetMarketsParams{
		Active: &active,
		Limit:  20,
		Order:  "liquidity",
		LiqMin: 10000, // At least $10k liquidity
	})
	if err != nil {
		t.Fatalf("GetMarkets failed: %v", err)
	}
	if len(markets) == 0 {
		t.Skip("No markets available with sufficient liquidity")
	}

	// Try to find a market with an active order book
	var foundBook *OrderBook
	for _, market := range markets {
		if market.ClobTokenIDs == "" || !market.EnableOrderBook {
			continue
		}

		var tokenIDs []string
		if err := json.Unmarshal([]byte(market.ClobTokenIDs), &tokenIDs); err != nil {
			continue
		}
		if len(tokenIDs) == 0 {
			continue
		}

		tokenID := tokenIDs[0]
		book, err := client.GetOrderBook(ctx, tokenID)
		if err == nil {
			t.Logf("Found order book for: %s", market.Question)
			t.Logf("Token ID: %s", tokenID)
			foundBook = book
			break
		}
	}

	if foundBook == nil {
		t.Skip("No active order books found in tested markets")
	}

	t.Logf("Order book: %d bids, %d asks", len(foundBook.Bids), len(foundBook.Asks))
	if len(foundBook.Bids) > 0 {
		t.Logf("Best bid: %s @ %s", foundBook.Bids[0].Size, foundBook.Bids[0].Price)
	}
	if len(foundBook.Asks) > 0 {
		t.Logf("Best ask: %s @ %s", foundBook.Asks[0].Size, foundBook.Asks[0].Price)
	}
}

func TestGetMidPrice(t *testing.T) {
	// Skip if we don't have a valid token ID to test with
	t.Skip("GetMidPrice requires a valid token ID - run integration tests manually")
}
