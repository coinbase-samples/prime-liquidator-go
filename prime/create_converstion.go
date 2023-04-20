package prime

import (
	"context"
	"fmt"
)

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
