// Package polymarket provides a Go client for Polymarket APIs.
package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// DefaultCLOBURL is the default Polymarket CLOB API URL.
	DefaultCLOBURL = "https://clob.polymarket.com"

	// DefaultGammaURL is the default Polymarket Gamma API URL.
	DefaultGammaURL = "https://gamma-api.polymarket.com"
)

// Client is a Polymarket API client.
type Client struct {
	clobURL    string
	gammaURL   string
	httpClient *http.Client
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithCLOBURL sets the CLOB API URL.
func WithCLOBURL(url string) ClientOption {
	return func(c *Client) {
		c.clobURL = url
	}
}

// WithGammaURL sets the Gamma API URL.
func WithGammaURL(url string) ClientOption {
	return func(c *Client) {
		c.gammaURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewClient creates a new Polymarket client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		clobURL:  DefaultCLOBURL,
		gammaURL: DefaultGammaURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Market represents a Polymarket prediction market.
type Market struct {
	ID              string  `json:"id"`
	Question        string  `json:"question"`
	Slug            string  `json:"slug"`
	Description     string  `json:"description"`
	ConditionID     string  `json:"conditionId"`
	QuestionID      string  `json:"questionID"`
	EndDate         string  `json:"endDate"`
	EndDateISO      string  `json:"endDateIso"`
	StartDate       string  `json:"startDate"`
	StartDateISO    string  `json:"startDateIso"`
	Active          bool    `json:"active"`
	Closed          bool    `json:"closed"`
	Liquidity       string  `json:"liquidity"`
	LiquidityNum    float64 `json:"liquidityNum"`
	Volume          string  `json:"volume"`
	VolumeNum       float64 `json:"volumeNum"`
	Volume24hr      float64 `json:"volume24hr"`
	Volume1wk       float64 `json:"volume1wk"`
	Volume1mo       float64 `json:"volume1mo"`
	Volume1yr       float64 `json:"volume1yr"`
	Outcomes        string  `json:"outcomes"`      // JSON array string
	OutcomePrices   string  `json:"outcomePrices"` // JSON array string
	ClobTokenIDs    string  `json:"clobTokenIds"`  // JSON array string
	Image           string  `json:"image"`
	Icon            string  `json:"icon"`
	EnableOrderBook bool    `json:"enableOrderBook"`
	AcceptingOrders bool    `json:"acceptingOrders"`
	BestBid         float64 `json:"bestBid"`
	BestAsk         float64 `json:"bestAsk"`
	Spread          float64 `json:"spread"`
	LastTradePrice  float64 `json:"lastTradePrice"`
}

// GetMarketsParams are parameters for fetching markets.
type GetMarketsParams struct {
	Active    *bool
	Closed    *bool
	Limit     int
	Cursor    string
	TagSlug   string
	Order     string
	Ascending bool
	LiqMin    float64
	LiqMax    float64
	VolumeMin float64
	VolumeMax float64
	StartDate string
	EndDate   string
	TextQuery string
}

// GetMarkets fetches markets from the Gamma API.
func (c *Client) GetMarkets(ctx context.Context, params GetMarketsParams) ([]Market, error) {
	u, err := url.Parse(c.gammaURL + "/markets")
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	q := u.Query()

	if params.Active != nil {
		q.Set("active", strconv.FormatBool(*params.Active))
	}
	if params.Closed != nil {
		q.Set("closed", strconv.FormatBool(*params.Closed))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}
	if params.TagSlug != "" {
		q.Set("tag_slug", params.TagSlug)
	}
	if params.Order != "" {
		q.Set("order", params.Order)
	}
	if params.Ascending {
		q.Set("ascending", "true")
	}
	if params.LiqMin > 0 {
		q.Set("liquidity_min", strconv.FormatFloat(params.LiqMin, 'f', -1, 64))
	}
	if params.LiqMax > 0 {
		q.Set("liquidity_max", strconv.FormatFloat(params.LiqMax, 'f', -1, 64))
	}
	if params.TextQuery != "" {
		q.Set("text_query", params.TextQuery)
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var markets []Market
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return markets, nil
}

// GetMarket fetches a single market by condition ID.
func (c *Client) GetMarket(ctx context.Context, conditionID string) (*Market, error) {
	u := fmt.Sprintf("%s/markets/%s", c.gammaURL, conditionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var market Market
	if err := json.NewDecoder(resp.Body).Decode(&market); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &market, nil
}

// OrderBook represents the order book for a market.
type OrderBook struct {
	Market string           `json:"market"`
	Asset  string           `json:"asset_id"`
	Hash   string           `json:"hash"`
	Bids   []OrderBookLevel `json:"bids"`
	Asks   []OrderBookLevel `json:"asks"`
}

// OrderBookLevel represents a price level in the order book.
type OrderBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// GetOrderBook fetches the order book for a token.
func (c *Client) GetOrderBook(ctx context.Context, tokenID string) (*OrderBook, error) {
	u := fmt.Sprintf("%s/book?token_id=%s", c.clobURL, tokenID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var book OrderBook
	if err := json.NewDecoder(resp.Body).Decode(&book); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &book, nil
}

// Price represents the current price for a token.
type Price struct {
	TokenID string  `json:"token_id"`
	Price   float64 `json:"price,string"`
}

// GetPrice fetches the current price for a token.
func (c *Client) GetPrice(ctx context.Context, tokenID string) (*Price, error) {
	u := fmt.Sprintf("%s/price?token_id=%s", c.clobURL, tokenID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var price Price
	if err := json.NewDecoder(resp.Body).Decode(&price); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &price, nil
}

// MidPrice represents the mid price for a token.
type MidPrice struct {
	Mid float64 `json:"mid,string"`
}

// GetMidPrice fetches the mid price for a token.
func (c *Client) GetMidPrice(ctx context.Context, tokenID string) (*MidPrice, error) {
	u := fmt.Sprintf("%s/midpoint?token_id=%s", c.clobURL, tokenID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var mid MidPrice
	if err := json.NewDecoder(resp.Body).Decode(&mid); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &mid, nil
}

// Spread represents the bid-ask spread for a token.
type Spread struct {
	Spread float64 `json:"spread,string"`
}

// GetSpread fetches the spread for a token.
func (c *Client) GetSpread(ctx context.Context, tokenID string) (*Spread, error) {
	u := fmt.Sprintf("%s/spread?token_id=%s", c.clobURL, tokenID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var spread Spread
	if err := json.NewDecoder(resp.Body).Decode(&spread); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &spread, nil
}
