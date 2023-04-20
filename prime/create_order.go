package prime

import (
	"context"
	"fmt"
)

func CreateOrder(ctx context.Context, request *CreateOrderRequest) (*CreateOrderResponse, error) {

	url := fmt.Sprintf("%s/portfolios/%s/order", primeV1ApiBaseUrl, request.PortfolioId)

	response := &CreateOrderResponse{Request: request}

	if err := PrimePost(ctx, url, request, response); err != nil {
		return nil, fmt.Errorf("unable to CreateOrder: %w", err)
	}

	return response, nil
}
