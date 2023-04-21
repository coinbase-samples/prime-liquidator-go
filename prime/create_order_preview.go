package prime

import (
	"context"
	"fmt"
)

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

func CreateOrderPreview(
	ctx context.Context,
	request *CreateOrderRequest,
) (*CreateOrderPreviewResponse, error) {

	url := fmt.Sprintf("%s/portfolios/%s/order_preview", primeV1ApiBaseUrl, request.PortfolioId)

	response := &CreateOrderPreviewResponse{Request: request}

	if err := PrimePost(ctx, url, request, response); err != nil {
		return nil, fmt.Errorf("unable to CreateOrderPreview: %w", err)
	}

	return response, nil
}
