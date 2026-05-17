package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMarketToolName(t *testing.T) {
	tool := &MarketTool{}
	if name := tool.Name(); name != "get_markets" {
		t.Errorf("Name() = %q, want %q", name, "get_markets")
	}
}

func TestMarketToolDescription(t *testing.T) {
	tool := &MarketTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
	if !containsString(desc, "market") {
		t.Error("Description should mention markets")
	}
}

func TestMarketToolParameters(t *testing.T) {
	tool := &MarketTool{}
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type = %v, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"min_liquidity", "max_days_to_resolution", "category", "limit"}
	for _, prop := range expectedProps {
		if props[prop] == nil {
			t.Errorf("missing property: %s", prop)
		}
	}
}

func TestMarketToolCallInvalidJSON(t *testing.T) {
	tool := &MarketTool{}
	_, err := tool.Call(context.Background(), "not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !containsString(err.Error(), "invalid parameters") {
		t.Errorf("error should mention invalid parameters: %v", err)
	}
}

func TestOrderBookToolName(t *testing.T) {
	tool := &OrderBookTool{}
	if name := tool.Name(); name != "get_orderbook" {
		t.Errorf("Name() = %q, want %q", name, "get_orderbook")
	}
}

func TestOrderBookToolDescription(t *testing.T) {
	tool := &OrderBookTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
	if !containsString(desc, "order book") {
		t.Error("Description should mention order book")
	}
}

func TestOrderBookToolParameters(t *testing.T) {
	tool := &OrderBookTool{}
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type = %v, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	if props["token_id"] == nil {
		t.Error("missing property: token_id")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a string slice")
	}
	if len(required) != 1 || required[0] != "token_id" {
		t.Errorf("required = %v, want [token_id]", required)
	}
}

func TestOrderBookToolCallInvalidJSON(t *testing.T) {
	tool := &OrderBookTool{}
	_, err := tool.Call(context.Background(), "bad json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestOrderBookToolCallMissingTokenID(t *testing.T) {
	tool := &OrderBookTool{}
	_, err := tool.Call(context.Background(), `{}`)
	if err == nil {
		t.Error("expected error for missing token_id")
	}
	if !containsString(err.Error(), "token_id is required") {
		t.Errorf("error should mention token_id: %v", err)
	}
}

func TestPlaceOrderToolName(t *testing.T) {
	tool := &PlaceOrderTool{}
	if name := tool.Name(); name != "place_order" {
		t.Errorf("Name() = %q, want %q", name, "place_order")
	}
}

func TestPlaceOrderToolDescription(t *testing.T) {
	tool := &PlaceOrderTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
	if !containsString(desc, "order") {
		t.Error("Description should mention order")
	}
}

func TestPlaceOrderToolParameters(t *testing.T) {
	tool := &PlaceOrderTool{}
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type = %v, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"token_id", "side", "price", "size"}
	for _, prop := range expectedProps {
		if props[prop] == nil {
			t.Errorf("missing property: %s", prop)
		}
	}

	// Check side enum
	sideProps, ok := props["side"].(map[string]any)
	if !ok {
		t.Fatal("side should be a map")
	}
	sideEnum, ok := sideProps["enum"].([]string)
	if !ok {
		t.Fatal("side.enum should be a string slice")
	}
	if len(sideEnum) != 2 || sideEnum[0] != "buy" || sideEnum[1] != "sell" {
		t.Errorf("side.enum = %v, want [buy, sell]", sideEnum)
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a string slice")
	}
	if len(required) != 4 {
		t.Errorf("required count = %d, want 4", len(required))
	}
}

func TestPlaceOrderToolCallInvalidJSON(t *testing.T) {
	tool := &PlaceOrderTool{}
	_, err := tool.Call(context.Background(), "invalid")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPlaceOrderToolCallValidation(t *testing.T) {
	tool := &PlaceOrderTool{}

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "missing token_id",
			input:     `{"side":"buy","price":0.5,"size":10}`,
			wantErr:   true,
			errSubstr: "token_id is required",
		},
		{
			name:      "invalid side",
			input:     `{"token_id":"123","side":"hold","price":0.5,"size":10}`,
			wantErr:   true,
			errSubstr: "side must be",
		},
		{
			name:      "price too low",
			input:     `{"token_id":"123","side":"buy","price":0.001,"size":10}`,
			wantErr:   true,
			errSubstr: "price must be between",
		},
		{
			name:      "price too high",
			input:     `{"token_id":"123","side":"buy","price":1.5,"size":10}`,
			wantErr:   true,
			errSubstr: "price must be between",
		},
		{
			name:      "size too small",
			input:     `{"token_id":"123","side":"buy","price":0.5,"size":0.001}`,
			wantErr:   true,
			errSubstr: "size must be at least",
		},
		{
			name:    "valid order",
			input:   `{"token_id":"123","side":"buy","price":0.5,"size":10}`,
			wantErr: false,
		},
		{
			name:    "valid sell order",
			input:   `{"token_id":"abc","side":"sell","price":0.99,"size":0.01}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Call(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == "" {
				t.Error("result should not be empty")
			}
		})
	}
}

func TestPlaceOrderToolCallReturnsNotImplemented(t *testing.T) {
	tool := &PlaceOrderTool{}

	result, err := tool.Call(context.Background(), `{
		"token_id": "test123",
		"side": "buy",
		"price": 0.65,
		"size": 100
	}`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "not_implemented" {
		t.Errorf("status = %v, want not_implemented", response["status"])
	}

	order, ok := response["order"].(map[string]any)
	if !ok {
		t.Fatal("order should be a map")
	}
	if order["token_id"] != "test123" {
		t.Errorf("order.token_id = %v, want test123", order["token_id"])
	}
	if order["side"] != "buy" {
		t.Errorf("order.side = %v, want buy", order["side"])
	}
	if order["price"] != 0.65 {
		t.Errorf("order.price = %v, want 0.65", order["price"])
	}
	if order["size"] != float64(100) {
		t.Errorf("order.size = %v, want 100", order["size"])
	}
}

func TestNewMarketTool(t *testing.T) {
	tool := NewMarketTool(nil)
	if tool == nil {
		t.Error("NewMarketTool should not return nil")
	}
}

func TestNewOrderBookTool(t *testing.T) {
	tool := NewOrderBookTool(nil)
	if tool == nil {
		t.Error("NewOrderBookTool should not return nil")
	}
}

func TestNewPlaceOrderTool(t *testing.T) {
	tool := NewPlaceOrderTool(nil)
	if tool == nil {
		t.Error("NewPlaceOrderTool should not return nil")
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
