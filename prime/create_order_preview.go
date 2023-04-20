package prime

import (
	"context"
	"fmt"
)

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
