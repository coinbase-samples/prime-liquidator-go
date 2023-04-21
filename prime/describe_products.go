package prime

import (
	"context"
	"fmt"
)

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
