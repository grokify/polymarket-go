// Package polymarket provides a Go client for Polymarket APIs.
// This file wraps the GoPolymarket/polymarket-go-sdk for full trading capabilities.
package polymarket

import (
	"context"
	"fmt"
	"os"

	polymarket "github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
	"github.com/shopspring/decimal"
)

// SDKClient wraps the polymarket-go-sdk for full trading capabilities.
type SDKClient struct {
	client *polymarket.Client
	signer auth.Signer
	apiKey *auth.APIKey
}

// SDKConfig holds configuration for the SDK client.
type SDKConfig struct {
	// PrivateKey is the wallet private key (hex string without 0x prefix).
	// If empty, reads from POLYGON_WALLET_PRIVATE_KEY env var.
	PrivateKey string

	// APIKey credentials for authenticated endpoints.
	// If empty, reads from POLYMARKET_API_KEY, POLYMARKET_API_SECRET, POLYMARKET_API_PASSPHRASE.
	APIKey     string
	APISecret  string
	Passphrase string

	// ChainID defaults to 137 (Polygon mainnet).
	ChainID int64
}

// NewSDKClient creates a new SDK-based client with full trading capabilities.
func NewSDKClient(cfg SDKConfig) (*SDKClient, error) {
	// Default to Polygon mainnet
	if cfg.ChainID == 0 {
		cfg.ChainID = 137
	}

	// Load private key from env if not provided
	privateKey := cfg.PrivateKey
	if privateKey == "" {
		privateKey = os.Getenv("POLYGON_WALLET_PRIVATE_KEY")
	}

	// Load API credentials from env if not provided
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("POLYMARKET_API_KEY")
	}
	apiSecret := cfg.APISecret
	if apiSecret == "" {
		apiSecret = os.Getenv("POLYMARKET_API_SECRET")
	}
	passphrase := cfg.Passphrase
	if passphrase == "" {
		passphrase = os.Getenv("POLYMARKET_API_PASSPHRASE")
	}

	// Create the base client
	client, err := polymarket.NewClientE()
	if err != nil {
		return nil, fmt.Errorf("creating polymarket client: %w", err)
	}

	sdkClient := &SDKClient{
		client: client,
	}

	// Set up signer if private key is available
	if privateKey != "" {
		signer, err := auth.NewPrivateKeySigner(privateKey, cfg.ChainID)
		if err != nil {
			return nil, fmt.Errorf("creating signer: %w", err)
		}
		sdkClient.signer = signer
	}

	// Set up API key if credentials are available
	if apiKey != "" && apiSecret != "" && passphrase != "" {
		sdkClient.apiKey = &auth.APIKey{
			Key:        apiKey,
			Secret:     apiSecret,
			Passphrase: passphrase,
		}
	}

	// Apply authentication if available
	if sdkClient.signer != nil && sdkClient.apiKey != nil {
		sdkClient.client = sdkClient.client.WithAuth(sdkClient.signer, sdkClient.apiKey)
	}

	return sdkClient, nil
}

// IsAuthenticated returns true if the client has trading credentials.
func (c *SDKClient) IsAuthenticated() bool {
	return c.signer != nil && c.apiKey != nil
}

// ----- Market Data Methods -----

// GetAllMarkets fetches all markets with optional filtering.
func (c *SDKClient) GetAllMarkets(ctx context.Context, active *bool) ([]clobtypes.Market, error) {
	req := &clobtypes.MarketsRequest{}
	if active != nil {
		req.Active = active
	}
	return c.client.CLOB.MarketsAll(ctx, req)
}

// GetMarketByID fetches a single market by condition ID.
func (c *SDKClient) GetMarketByID(ctx context.Context, conditionID string) (clobtypes.Market, error) {
	resp, err := c.client.CLOB.Market(ctx, conditionID)
	if err != nil {
		return clobtypes.Market{}, err
	}
	return clobtypes.Market(resp), nil
}

// GetOrderBookSDK fetches the order book for a token.
func (c *SDKClient) GetOrderBookSDK(ctx context.Context, tokenID string) (clobtypes.OrderBook, error) {
	resp, err := c.client.CLOB.OrderBook(ctx, &clobtypes.BookRequest{
		TokenID: tokenID,
	})
	if err != nil {
		return clobtypes.OrderBook{}, err
	}
	return clobtypes.OrderBook(resp), nil
}

// GetMidpointSDK fetches the midpoint price for a token.
func (c *SDKClient) GetMidpointSDK(ctx context.Context, tokenID string) (clobtypes.MidpointResponse, error) {
	return c.client.CLOB.Midpoint(ctx, &clobtypes.MidpointRequest{
		TokenID: tokenID,
	})
}

// GetSpreadSDK fetches the spread for a token.
func (c *SDKClient) GetSpreadSDK(ctx context.Context, tokenID string) (clobtypes.SpreadResponse, error) {
	return c.client.CLOB.Spread(ctx, &clobtypes.SpreadRequest{
		TokenID: tokenID,
	})
}

// GetPriceSDK fetches the price for a token on a specific side.
func (c *SDKClient) GetPriceSDK(ctx context.Context, tokenID, side string) (clobtypes.PriceResponse, error) {
	return c.client.CLOB.Price(ctx, &clobtypes.PriceRequest{
		TokenID: tokenID,
		Side:    side,
	})
}

// ----- Trading Methods -----

// PlaceOrder places a limit order on the CLOB.
func (c *SDKClient) PlaceOrder(ctx context.Context, params PlaceOrderParams) (clobtypes.OrderResponse, error) {
	if !c.IsAuthenticated() {
		return clobtypes.OrderResponse{}, fmt.Errorf("client not authenticated: missing private key or API credentials")
	}

	builder := clob.NewOrderBuilder(c.client.CLOB, c.signer).
		TokenID(params.TokenID).
		Side(params.Side).
		PriceDec(params.Price).
		SizeDec(params.Size).
		OrderType(params.OrderType)

	order, err := builder.BuildWithContext(ctx)
	if err != nil {
		return clobtypes.OrderResponse{}, fmt.Errorf("building order: %w", err)
	}

	return c.client.CLOB.CreateOrder(ctx, order)
}

// PlaceOrderParams holds parameters for placing an order.
type PlaceOrderParams struct {
	TokenID   string
	Side      string // "BUY" or "SELL"
	Price     decimal.Decimal
	Size      decimal.Decimal
	OrderType clobtypes.OrderType
}

// PlaceMarketOrder places a market order (FOK - Fill or Kill).
func (c *SDKClient) PlaceMarketOrder(ctx context.Context, params MarketOrderParams) (clobtypes.OrderResponse, error) {
	if !c.IsAuthenticated() {
		return clobtypes.OrderResponse{}, fmt.Errorf("client not authenticated: missing private key or API credentials")
	}

	builder := clob.NewOrderBuilder(c.client.CLOB, c.signer).
		TokenID(params.TokenID).
		Side(params.Side).
		SizeDec(params.Size).
		OrderType(clobtypes.OrderTypeFOK) // Fill or Kill for market orders

	// For market orders, use a price that will definitely fill
	if params.Side == "BUY" {
		builder = builder.PriceDec(decimal.NewFromFloat(0.99)) // Max price for buy
	} else {
		builder = builder.PriceDec(decimal.NewFromFloat(0.01)) // Min price for sell
	}

	order, err := builder.BuildWithContext(ctx)
	if err != nil {
		return clobtypes.OrderResponse{}, fmt.Errorf("building market order: %w", err)
	}

	return c.client.CLOB.CreateOrder(ctx, order)
}

// MarketOrderParams holds parameters for placing a market order.
type MarketOrderParams struct {
	TokenID string
	Side    string // "BUY" or "SELL"
	Size    decimal.Decimal
}

// CancelOrder cancels an existing order.
func (c *SDKClient) CancelOrder(ctx context.Context, orderID string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated")
	}

	_, err := c.client.CLOB.CancelOrder(ctx, &clobtypes.CancelOrderRequest{
		OrderID: orderID,
	})
	return err
}

// CancelAllOrders cancels all open orders.
func (c *SDKClient) CancelAllOrders(ctx context.Context) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated")
	}

	_, err := c.client.CLOB.CancelAll(ctx)
	return err
}

// GetOrders fetches orders for the authenticated user.
func (c *SDKClient) GetOrders(ctx context.Context, market string, limit int) ([]clobtypes.OrderResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	req := &clobtypes.OrdersRequest{
		Limit: limit,
	}
	if market != "" {
		req.Market = market
	}

	return c.client.CLOB.OrdersAll(ctx, req)
}

// GetTrades fetches trade history for the authenticated user.
func (c *SDKClient) GetTrades(ctx context.Context, limit int) (clobtypes.TradesResponse, error) {
	if !c.IsAuthenticated() {
		return clobtypes.TradesResponse{}, fmt.Errorf("client not authenticated")
	}

	return c.client.CLOB.Trades(ctx, &clobtypes.TradesRequest{
		Limit: limit,
	})
}

// ----- Balance Methods -----

// GetBalanceAllowance fetches USDC balance and allowance.
func (c *SDKClient) GetBalanceAllowance(ctx context.Context, tokenID string) (clobtypes.BalanceAllowanceResponse, error) {
	if !c.IsAuthenticated() {
		return clobtypes.BalanceAllowanceResponse{}, fmt.Errorf("client not authenticated")
	}

	return c.client.CLOB.BalanceAllowance(ctx, &clobtypes.BalanceAllowanceRequest{
		AssetType: clobtypes.AssetTypeConditional,
		TokenID:   tokenID,
	})
}

// ----- WebSocket Methods -----

// SubscribePrices subscribes to real-time price updates for tokens.
func (c *SDKClient) SubscribePrices(ctx context.Context, tokenIDs []string) (<-chan ws.PriceChangeEvent, error) {
	return c.client.CLOBWS.SubscribePrices(ctx, tokenIDs)
}

// SubscribeOrderbook subscribes to real-time order book updates.
func (c *SDKClient) SubscribeOrderbook(ctx context.Context, tokenIDs []string) (<-chan ws.OrderbookEvent, error) {
	return c.client.CLOBWS.SubscribeOrderbook(ctx, tokenIDs)
}

// SubscribeMidpoints subscribes to real-time midpoint updates.
func (c *SDKClient) SubscribeMidpoints(ctx context.Context, tokenIDs []string) (<-chan ws.MidpointEvent, error) {
	return c.client.CLOBWS.SubscribeMidpoints(ctx, tokenIDs)
}

// SubscribeUserOrders subscribes to order status updates for the authenticated user.
func (c *SDKClient) SubscribeUserOrders(ctx context.Context, markets []string) (<-chan ws.OrderEvent, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}
	return c.client.CLOBWS.SubscribeUserOrders(ctx, markets)
}

// SubscribeUserTrades subscribes to trade events for the authenticated user.
func (c *SDKClient) SubscribeUserTrades(ctx context.Context, markets []string) (<-chan ws.TradeEvent, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}
	return c.client.CLOBWS.SubscribeUserTrades(ctx, markets)
}

// ----- Gamma API Methods -----

// GetEventsGamma fetches events from the Gamma API.
func (c *SDKClient) GetEventsGamma(ctx context.Context) ([]gamma.Event, error) {
	return c.client.Gamma.Events(ctx, nil)
}

// GetEventGamma fetches a single event by ID from the Gamma API.
func (c *SDKClient) GetEventGamma(ctx context.Context, eventID string) (*gamma.Event, error) {
	return c.client.Gamma.GetEvent(ctx, eventID)
}

// GetMarketsGamma fetches markets from the Gamma API.
func (c *SDKClient) GetMarketsGamma(ctx context.Context) ([]gamma.Market, error) {
	return c.client.Gamma.Markets(ctx, nil)
}

// ----- Underlying Client Access -----

// CLOB returns the underlying CLOB client for advanced usage.
func (c *SDKClient) CLOBClient() clob.Client {
	return c.client.CLOB
}

// CLOBWS returns the underlying WebSocket client for advanced usage.
func (c *SDKClient) CLOBWSClient() ws.Client {
	return c.client.CLOBWS
}

// GammaClient returns the underlying Gamma client for advanced usage.
func (c *SDKClient) GammaClient() gamma.Client {
	return c.client.Gamma
}

// Raw returns the underlying polymarket-go-sdk client.
func (c *SDKClient) Raw() *polymarket.Client {
	return c.client
}
