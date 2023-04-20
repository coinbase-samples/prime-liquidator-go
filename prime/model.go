package prime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	WalletTypeVault   = "VAULT"
	WalletTypeTrading = "TRADING"
	WalletTypeOther   = "WALLET_TYPE_OTHER"

	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"

	OrderTypeMarket = "MARKET"
	OrderTypeLimit  = "LIMIT"
	OrderTypeTwap   = "TWAP"
	OrderTypeBlock  = "BLOCK"

	TimeInForceGoodUntilTime      = "GOOD_UNTIL_DATE_TIME"
	TimeInForceGoodUntilCancelled = "GOOD_UNTIL_CANCELLED"
	TimeInForceImmediateOrCancel  = "IMMEDIATE_OR_CANCEL"
)

type Call struct {
	Url                    string
	HttpMethod             string
	Body                   []byte
	ExpectedHttpStatusCode int
	Credentials            *Credentials
}

type Response struct {
	Call           *Call
	Body           []byte
	HttpStatusCode int
	HttpStatusMsg  string
	Error          error
}

func (r Response) IsHttpOk() bool {
	return r.HttpStatusCode == 200
}

func (r Response) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

type ErrorMessage struct {
	Message string `json:"message"`
}

type DescribeBalancesRequest struct {
	PortfolioId string   `json:"portfolio_id"`
	Type        string   `json:"balance_type"`
	Symbols     []string `json:"symbols"`
}

type DescribeBalancesResponse struct {
	Balances        []*AssetBalances         `json:"balances"`
	Type            string                   `json:"type"`
	TradingBalances *BalanceWithHolds        `json:"trading_balances"`
	VaultBalances   *BalanceWithHolds        `json:"vault_balances"`
	Request         *DescribeBalancesRequest `json:"request"`
}

type AssetBalances struct {
	Symbol               string `json:"symbol"`
	Amount               string `json:"amount"`
	Holds                string `json:"holds"`
	BondedAmount         string `json:"bonded_amount"`
	ReservedAmount       string `json:"reserved_amount"`
	UnbondingAmount      string `json:"unbonding_amount"`
	UnvestedAmount       string `json:"unvested_amount"`
	PendingRewardsAmount string `json:"pending_rewards_amount"`
	PastRewardsAmount    string `json:"past_rewards_amount"`
	BondableAmount       string `json:"bondable_amount"`
	WithdrawableAmount   string `json:"withdrawable_amount"`
}

func (b AssetBalances) AmountNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(b.Amount)
	if err != nil {
		err = fmt.Errorf("Invalid asset amount: %s - symbol: %s - msg: %v", b.Amount, b.Symbol, err)
	}
	return
}

func (b AssetBalances) HoldsNum() (holds decimal.Decimal, err error) {
	holds, err = strToNum(b.Holds)
	if err != nil {
		err = fmt.Errorf("Invalid asset holds: %s - symbol: %s - msg: %v", b.Holds, b.Symbol, err)
	}
	return
}

func (b AssetBalances) IsFiat() (f bool) {
	if strings.ToLower(b.Symbol) == "usd" {
		f = true
	}
	return
}

type BalanceWithHolds struct {
	Total string `json:"total"`
	Holds string `json:"holds"`
}

type IteratorParams struct {
	Cursor        string `json:"cursor"`
	Limit         string `json:"limit"`
	SortDirection string `json:"sort_direction"`
}

type DescribeProductsRequest struct {
	PortfolioId    string          `json:"portfolioId"`
	IteratorParams *IteratorParams `json:"iteratorParams"`
}

type DescribeProductsResponse struct {
	Products   []*Product               `json:"products"`
	Pagination *Pagination              `json:"pagination"`
	Request    *DescribeProductsRequest `json:"request"`
}

func (r DescribeProductsResponse) HasNext() bool {
	return r.Pagination != nil && r.Pagination.HasNext
}

type DescribeWalletsRequest struct {
	PortfolioId    string          `json:"string"`
	Type           string          `json:"type"`
	Symbols        []string        `json:"symbols"`
	IteratorParams *IteratorParams `json:"iteratorParams"`
}

type DescribeWalletsResponse struct {
	Wallets    []*Wallet               `json:"wallets"`
	Request    *DescribeWalletsRequest `json:"request"`
	Pagination *Pagination             `json:"pagination"`
}

func (r DescribeWalletsResponse) HasNext() bool {
	return r.Pagination != nil && r.Pagination.HasNext
}

type Wallet struct {
	Id        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Cursor    string    `json:"cursor"`
	Symbol    string    `json:"symbol"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	PortfolioId      string `json:"portfolio_id"`
	Side             string `json:"side"`
	ClientOrderId    string `json:"client_order_id"`
	ProductId        string `json:"product_id"`
	Type             string `json:"type"`
	BaseQuantity     string `json:"base_quantity"`
	QuoteValue       string `json:"quote_value,omitempty"`
	LimitPrice       string `json:"limit_price,omitempty"`
	StartTime        string `json:"start_time,omitempty"`
	ExpiryTime       string `json:"expiry_time,omitempty"`
	TimeInForce      string `json:"time_in_force,omitempty"`
	StpId            string `json:"stp_id,omitempty"`
	DisplayQuoteSize string `json:"display_quote_size,omitempty"`
	DisplayBaseSize  string `json:"display_base_size,omitempty"`
	IsRaiseExact     string `json:"is_raise_exact,omitempty"`
}

type CreateOrderResponse struct {
	OrderId string              `json:"order_id"`
	Request *CreateOrderRequest `json:"request"`
}

type CreateOrderPreviewResponse struct {
	PortfolioId      string              `json:"portfolio_id"`
	Side             string              `json:"side"`
	ClientOrderId    string              `json:"client_order_id"`
	ProductId        string              `json:"product_id"`
	Type             string              `json:"type"`
	BaseQuantity     string              `json:"base_quantity"`
	QuoteValue       string              `json:"quote_value,omitempty"`
	LimitPrice       string              `json:"limit_price,omitempty"`
	StartTime        string              `json:"start_time,omitempty"`
	ExpiryTime       string              `json:"expiry_time,omitempty"`
	TimeInForce      string              `json:"time_in_force,omitempty"`
	StpId            string              `json:"stp_id,omitempty"`
	DisplayQuoteSize string              `json:"display_quote_size,omitempty"`
	DisplayBaseSize  string              `json:"display_base_size,omitempty"`
	IsRaiseExact     string              `json:"is_raise_exact,omitempty"`
	Commission       string              `json:"commission"`
	Slippage         string              `json:"slippage"`
	BestBid          string              `json:"best_bid"`
	BestAsk          string              `json:"best_ask"`
	AvgFillPrice     string              `json:"average_filled_price"`
	OrderTotal       string              `json:"order_total"`
	Request          *CreateOrderRequest `json:"request"`
}

type CreateConversionRequest struct {
	PortfolioId         string `json:"portfolio_id"`
	SourceWalletId      string `json:"wallet_id"`
	SourceSymbol        string `json:"source_symbol"`
	DestinationWalletId string `json:"destination"`
	DestinationSymbol   string `json:"destination_symbol"`
	IdempotencyId       string `json:"idempotency_key"`
	Amount              string `json:"amount"`
}

type CreateConversionResponse struct {
	ActivityId          string                   `json:"activity_id"`
	SourceSymbol        string                   `json:"source_symbol"`
	DestinationSymbol   string                   `json:"destination_symbol"`
	Amount              string                   `json:"amount"`
	DestinationWalletId string                   `json:"destination"`
	SourceWalletId      string                   `json:"source"`
	Request             *CreateConversionRequest `json:"request"`
}

type Credentials struct {
	AccessKey    string `json:"accessKey"`
	Passphrase   string `json:"passphrase"`
	SigningKey   string `json:"signingKey"`
	PortfolioId  string `json:"portfolioId"`
	SvcAccountId string `json:"svcAccountId"`
}

type Pagination struct {
	NextCursor    string `json:"next_cursor"`
	SortDirection string `json:"sort_direction"`
	HasNext       bool   `json:"has_next"`
}

type PortfolioCommission struct {
	Type          string `json:"type"`
	Rate          string `json:"rate"`
	TradingVolume string `json:"trading_volume"`
}

func (p PortfolioCommission) RateNum() (rate decimal.Decimal, err error) {
	rate, err = strToNum(p.Rate)
	if err != nil {
		err = fmt.Errorf("Invalid commission rate: %s - err: %w", p.Rate, err)
	}
	return
}

type Product struct {
	Id             string   `json:"id"`
	BaseIncrement  string   `json:"base_increment"`
	QuoteIncrement string   `json:"quote_increment"`
	BaseMinSize    string   `json:"base_min_size"`
	BaseMaxSize    string   `json:"base_max_size"`
	QuoteMinSize   string   `json:"quote_min_size"`
	QuoteMaxSize   string   `json:"quote_max_size"`
	Permissions    []string `json:"permissions"`
}

func (p Product) BaseMinSizeNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(p.BaseMinSize)
	if err != nil {
		err = fmt.Errorf("invalid base min: %s - id: %s - err: %w", p.BaseMinSize, p.Id, err)
	}
	return
}

func (p Product) BaseMaxSizeNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(p.BaseMaxSize)
	if err != nil {
		err = fmt.Errorf("invalid base max: %s - id: %s - err: %v", p.BaseMaxSize, p.Id, err)
	}
	return
}

func (p Product) BaseIncrementNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(p.BaseIncrement)
	if err != nil {
		err = fmt.Errorf("invalid base increment: %s - id: %s - msg: %w", p.BaseIncrement, p.Id, err)
	}
	return
}

func (p Product) QuoteMinSizeNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(p.QuoteMinSize)
	if err != nil {
		err = fmt.Errorf("invalid quote min: %s - id: %s - err: %w", p.QuoteMinSize, p.Id, err)
	}
	return
}

func (p Product) QuoteMaxSizeNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(p.QuoteMaxSize)
	if err != nil {
		err = fmt.Errorf("invalid quote max: %s - id: %s - err: %v", p.QuoteMaxSize, p.Id, err)
	}
	return
}

func (p Product) QuoteIncrementNum() (amount decimal.Decimal, err error) {
	amount, err = strToNum(p.QuoteIncrement)
	if err != nil {
		err = fmt.Errorf("invalid quote increment: %s - id: %s - msg: %w", p.QuoteIncrement, p.Id, err)
	}
	return
}

func strToNum(v string) (amount decimal.Decimal, err error) {
	amount, err = decimal.NewFromString(v)
	return
}
