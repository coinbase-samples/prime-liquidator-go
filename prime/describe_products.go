package prime

import (
	"context"
	"fmt"
)

// TODO: Add an interator version as well

func DescribeProducts(
	ctx context.Context,
	request *DescribeProductsRequest,
) (*DescribeProductsResponse, error) {

	url := fmt.Sprintf("%s/portfolios/%s/products", primeV1ApiBaseUrl, request.PortfolioId)

	url = urlIteratorParams(url, request.IteratorParams)

	response := &DescribeProductsResponse{Request: request}

	if err := PrimeGet(ctx, url, request, response); err != nil {
		return response, fmt.Errorf("unable to DescribeProducts: %w", err)
	}

	return response, nil
}
