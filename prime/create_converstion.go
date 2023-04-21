package prime

import (
	"context"
	"fmt"
)

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

func CreateConversion(
	ctx context.Context,
	request *CreateConversionRequest,
) (*CreateConversionResponse, error) {

	url := fmt.Sprintf("%s/portfolios/%s/wallets/%s/conversion",
		primeV1ApiBaseUrl,
		request.PortfolioId,
		request.SourceWalletId,
	)

	response := &CreateConversionResponse{Request: request}

	if err := PrimePost(ctx, url, request, response); err != nil {
		return nil, fmt.Errorf("unable to CreateConversion: %w", err)
	}

	return response, nil

}
